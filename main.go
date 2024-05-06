package main

import (
	"context"
	"fmt"
	irc "twitch_irc/irc_client"
)

func main() {

	sampleCallback := func(ircc *irc.IrcClient, msg string) error {
		fmt.Println("<CBLK> Received message: ", msg)
		return nil
	}

	ircc := irc.IrcClient{
		Nick: "justinfan123",
		Pass: "bruh",
	}

	ircc.RegisterMessageCallback(sampleCallback)

	ctx := context.Background()
	client_status := make(chan error)
	go func() {
		client_status <- ircc.ClientLoop(ctx)
	}()

	// Send test.
	go func() {
		ircc.SendMessage("JOIN #cfairy")
		ircc.SendMessage("PRIVMSG #cfairy :Hello World!")
	}()

	select {
	case <-client_status:
		fmt.Println("Client exited")
	}

}
