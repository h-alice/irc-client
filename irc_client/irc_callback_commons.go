package irc_client

import "time"

func lastPongTracker(ircc *IrcClient, msg string) error {
	parsed_message, err := ParseIrcMessage(msg)
	if err != nil {
		return err
	}

	if parsed_message.Command == "PONG" && parsed_message.Params[0] == "tmi.twitch.tv" {
		ircc.lastPong = time.Now()
	}

	return nil
}
