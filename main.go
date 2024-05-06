package main

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

type IrcMessageCallback func(string)
type IrcClient struct {
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
	clientLoopCondition *sync.Cond
	rwWaitGroup         *sync.WaitGroup // WaitGroup for the sender and receiver
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
				callback(string(msg))
			}
		}
	}
}

func (ircc *IrcClient) clientReadWriteLoop(ctx context.Context) {
	ircc.rwWaitGroup = &sync.WaitGroup{}

	// The wait value is 1 since if any of sender or receiver exits,
	// the client should exit as well.
	ircc.rwWaitGroup.Add(1)

	rwctx, cancel := context.WithCancel(ctx) // Create a new context for the sender and receiver
	go ircc.receiverLoop(rwctx)              // Start the receiver loop
	go ircc.senderLoop(rwctx)                // Start the sender loop
	go ircc.receiverCallbackInvoker(rwctx)   // Start the receiver callback invoker

	ircc.rwWaitGroup.Wait()

	// If we reach here, it means that the sender or receiver has exited.
	cancel()                             // Cancel the context to stop the other goroutine.
	ircc.clientLoopCondition.Broadcast() // Notify that the client has exited.
}

func (ircc *IrcClient) sendMessageInternal(msg []byte) {
	msg = append(msg, '\r', '\n') // Append the CRLF to the message.
	ircc.send <- []byte(msg)      // Send the message to the sender channel.
}

func (ircc *IrcClient) pingPong() {
	ircc.sendMessageInternal([]byte("PING :tmi.twitch.tv"))
}

func (ircc *IrcClient) RegisterMessageCallback(callback IrcMessageCallback) {
	ircc.messageCallbacks = append(ircc.messageCallbacks, callback)
}

func (ircc *IrcClient) SendMessage(msg string) {
	ircc.sendMessageInternal([]byte(msg))
}

func (ircc *IrcClient) SendLogin() {
	ircc.SendMessage("PASS " + ircc.Pass)
	ircc.SendMessage("NICK " + ircc.Nick)
}

func (ircc *IrcClient) connect(ctx context.Context) error {
	connection, err := net.Dial("tcp4", "irc.chat.twitch.tv:6667")
	if err != nil {
		return err
	}

	ircc.conn = connection
	return nil
}

func (ircc *IrcClient) Init() error {
	ircc.recv = make(chan []byte, 8192) // Size is for test
	ircc.send = make(chan []byte, 8192) // Size is for test

	init_ctx := context.Background()
	err := ircc.connect(init_ctx)
	if err != nil {
		return err
	}

	ircc.clientLoopCondition = sync.NewCond(&sync.Mutex{})
	go ircc.clientReadWriteLoop(init_ctx)

	// Send PASS and NICK
	ircc.SendLogin()

	// wait
	ircc.clientLoopCondition.Wait()

	return nil

}
func main() {

	sampleCallback := func(msg string) {
		fmt.Println("<CBLK> Received message: ", msg)
	}

	ircc := IrcClient{
		Nick: "justinfan123",
		Pass: "bruh",
	}

	ircc.RegisterMessageCallback(sampleCallback)
	err := ircc.Init()
	if err != nil {
		log.Fatal(err)
	}

}
