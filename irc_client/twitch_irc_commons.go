package irc_client

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