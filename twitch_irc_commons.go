package irc

import (
	"time"
)

const (
	// Capability to request membership state.
	CapabilityMembership = "twitch.tv/membership"

	// Capability to request tags.
	CapabilityTags = "twitch.tv/tags"

	// Capability to request commands.
	CapabilityCommands = "twitch.tv/commands"
)

func IsEndOfTwitchWelcomeMessage(msg IrcMessage) bool {
	return msg.Command == "376" && msg.Prefix.Nickname == "tmi.twitch.tv"
}

func endOfTwitchBannerCallback(ircc *IrcClient, msg string) error {
	parsed_message, _ := ParseIrcMessage(msg)

	if IsEndOfTwitchWelcomeMessage(parsed_message) {
		ircc.Ready()
	}
	return nil
}

func lastPongTracker(ircc *IrcClient, msg string) error {
	parsed_message, err := ParseIrcMessage(msg)
	if err != nil {
		return err
	}

	if parsed_message.Command == "PONG" && parsed_message.Params[0] == "tmi.twitch.tv" {
		// Debug print.
		//log.Println("Received PONG from server.")
		ircc.lastPong = time.Now()
	}

	return nil
}
