package irc

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"time"
)

type IrcMessageCallback func(*IrcClient, string) error
type IrcClient struct {
	server string
	port   int

	Nick string
	Pass string

	// Connection settings
	PingFrequency time.Duration
	PingTimeout   time.Duration

	// Underlying message queue
	recv chan []byte
	send chan []byte

	// Underlying connection
	conn net.Conn

	// List of message callbacks
	messageCallbacks []IrcMessageCallback

	// WaitGroup for the sender and receiver
	rwWaitGroup *sync.WaitGroup

	// Condition indicating if the client is successfully initialized.
	initialized *sync.WaitGroup

	// Condition indicating if the client got welcome message.
	readyToSend *sync.WaitGroup

	// Timestamp for last received PONG.
	lastPong time.Time
}

func (ircc *IrcClient) senderLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			ircc.rwWaitGroup.Done()
			return
		case msg := <-ircc.send:

			_, err := ircc.conn.Write(msg)
			if err != nil {
				log.Println("Error writing to connection:", err)
				ircc.rwWaitGroup.Done()
				return
			}

		}
	}
}

func (ircc *IrcClient) receiverLoop(ctx context.Context) {

	for {
		select {
		case <-ctx.Done(): // If the context is cancelled, then exit the loop.
			ircc.rwWaitGroup.Done() // Decrement the wait group.
			return                  // Exit the loop.
		default:
			temp_line := bytes.NewBuffer(nil) // Create a new buffer for each line.
			one_byte_buf := make([]byte, 1)   // Create a buffer of size 1 to read one byte at a time.
			// Read, one byte at a time, until we reach a newline.
			for {
				n, err := ircc.conn.Read(one_byte_buf) // Read one byte.
				if err != nil {                        // Check for errors.
					if err == io.EOF { // If EOF, then the connection is closed by the server.
						fmt.Println("Connection closed by server") // Print a message and return.
						ircc.rwWaitGroup.Done()                    // Decrement the wait group.
						return                                     // Exit the loop.
					} else { // Some other error occurred.
						fmt.Println("Error reading from connection:", err) // Print the error.
						ircc.rwWaitGroup.Done()                            // Decrement the wait group.
						return                                             // Exit the loop.
					}
				}

				if n == 0 { // If no bytes were read, then continue.
					continue
				}

				temp_line.Write(one_byte_buf) // Write the byte to the line buffer.

				if one_byte_buf[0] == '\n' { // If the byte is a newline, then break.
					break // Break the loop.
				}
			}
			ircc.recv <- temp_line.Bytes() // Send the line to the receiver channel.
		}
	}
}

func (ircc *IrcClient) receiverCallbackInvoker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-ircc.recv:
			for _, callback := range ircc.messageCallbacks {
				err := callback(ircc, string(msg))
				if err != nil {
					log.Println("Error occurred and ignored in callback:", err)
				}
			}
		}
	}
}

func (ircc *IrcClient) clientReadWriteLoop(ctx context.Context) {

	rwctx, cancel := context.WithCancel(ctx) // Create a new context for the sender and receiver
	ircc.rwWaitGroup = &sync.WaitGroup{}     // Initialize the wait group
	// The wait value is 1 since if any of sender or receiver exits,
	// the client should exit as well.
	ircc.rwWaitGroup.Add(1)

	// Start the sender and receiver goroutines.
	go ircc.receiverLoop(rwctx)            // Start the receiver loop
	go ircc.senderLoop(rwctx)              // Start the sender loop
	go ircc.receiverCallbackInvoker(rwctx) // Start the receiver callback invoker

	ircc.rwWaitGroup.Wait()

	// If we reach here, it means that the sender or receiver has exited.
	cancel() // Cancel the context to stop the other goroutine.

}

func (ircc *IrcClient) sendMessageInternal(msg []byte) {
	ircc.send <- []byte(msg) // Send the message to the sender channel.
}

func (ircc *IrcClient) RegisterMessageCallback(callback IrcMessageCallback) {
	ircc.messageCallbacks = append(ircc.messageCallbacks, callback)
}

// Message sent by this function has first priority.
func (ircc *IrcClient) sendRawMessagePrivileged(msg string) {

	msg = msg + "\r\n" // Append the CRLF to the message.
	go ircc.sendMessageInternal([]byte(msg))
}

func (ircc *IrcClient) SendRawMessage(msg string) {

	msg = msg + "\r\n" // Append the CRLF to the message.

	go func() {
		for {
			if ircc.initialized != nil && ircc.readyToSend != nil {
				ircc.initialized.Wait()
				ircc.readyToSend.Wait()
				ircc.sendMessageInternal([]byte(msg))
				break
			} else {
				continue
			}
		}
	}()
}

func (ircc *IrcClient) SendMessage(msg IrcMessage) {
	ircc.SendRawMessage(msg.String())
}

func (ircc *IrcClient) SendCapabilityRequest(capability string) {
	ircc.SendRawMessage("CAP REQ :" + capability)
}

func (ircc *IrcClient) SendLogin() {
	ircc.sendRawMessagePrivileged("PASS " + ircc.Pass)
	ircc.sendRawMessagePrivileged("NICK " + ircc.Nick)
}

func (ircc *IrcClient) connect(ctx context.Context) error {

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		connection, err := net.Dial("tcp4", fmt.Sprintf("%s:%d", ircc.server, ircc.port))
		if err != nil {
			return err
		}

		ircc.conn = connection
		return nil
	}
}

func (ircc *IrcClient) ClientLoop(ctx context.Context) error {
	ircc.recv = make(chan []byte, 8192) // Size is for test
	ircc.send = make(chan []byte, 8192) // Size is for test

	ircc.initialized = &sync.WaitGroup{}
	ircc.initialized.Add(1)

	ircc.readyToSend = &sync.WaitGroup{}
	ircc.readyToSend.Add(1)

	connection_result := make(chan error)
	ctx, cancel := context.WithCancel(ctx)

	go func() {
		connection_result <- ircc.connect(ctx)
	}()

	select {
	case <-ctx.Done():
		cancel()
		return ctx.Err()
	case err := <-connection_result:
		if err != nil {
			return err
		}
	}

	rw_ctx, cancel := context.WithCancel(ctx)
	rw_loop_result := make(chan interface{})
	go func() {
		ircc.clientReadWriteLoop(rw_ctx)
		rw_loop_result <- nil
	}()

	// Send PASS and NICK
	ircc.SendLogin()

	// Set client as initialized.
	ircc.initialized.Done()

	select {
	case <-ctx.Done():
		cancel()
		return ctx.Err()
	case <-rw_loop_result:
		cancel()
		return nil
	}
}

func (ircc *IrcClient) Ready() {
	ircc.readyToSend.Done()
}

func NewTwitchIrcClient(nick string, pass string) *IrcClient {
	ircc := IrcClient{
		server: "irc.chat.twitch.tv",
		port:   6667,
		Nick:   nick,
		Pass:   pass,
	}

	ircc.RegisterMessageCallback(endOfTwitchBannerCallback)
	ircc.RegisterMessageCallback(pingHandler)
	ircc.RegisterMessageCallback(lastPongTracker)

	return &ircc
}
