package irc_client

import (
	"errors"
	"strings"
)

var (
	ErrorInvalidMessage = errors.New("invalid message")
)

type IrcMessageTags map[string]string

type IrcMessageParams []string

type IrcMessagePrefix struct {
	Nickname string
	Username string
	Hostname string
}

type IrcMessage struct {
	Tags    map[string]string
	Prefix  IrcMessagePrefix
	Command string
	Params  []string
	Message string
}

func (irct IrcMessageTags) String() string {
	current_string := "@"
	for key, value := range irct {
		current_string += key + "=" + value + ";"
	}

	return current_string
}

func (ircp IrcMessagePrefix) String() string {
	current_string := ":"
	current_string += ircp.Nickname
	if ircp.Username != "" {
		current_string += "!" + ircp.Username
	}
	if ircp.Hostname != "" {
		current_string += "@" + ircp.Hostname
	}

	return current_string
}

func ParseIrcMessage(irc_message_string string) (IrcMessage, error) {
	irc_message_struct := IrcMessage{}                        // Initialize the struct.
	previous := irc_message_string                            // Save the original string.
	current, after, _ := strings.Cut(irc_message_string, " ") // Cut the string by space.

	// Check tag part.
	if current[0] != '@' { // Tag field signature.
		irc_message_struct.Tags = nil // No tags.
		current = previous            // Reset current.
	} else {
		// Try to parse tags.
		tags := make(map[string]string)            // Initialize the tag holder.
		tags_string := current[1:]                 // Remove the '@' character.
		tags_kv := strings.Split(tags_string, ";") // Split by ';', now we have key=value pairs.

		for _, kv := range tags_kv { // Iterate over key=value pairs.
			kv_split := strings.Split(kv, "=") // Split by '='.
			if len(kv_split) != 2 {
				// Invalid key=value pair.
				continue // Skip this pair.
			}
			tags[kv_split[0]] = kv_split[1] // Add to the tag holder.
		}

		irc_message_struct.Tags = tags // Save the tags.
		current = after                // Move to the next part.
	}

	// Check prefix part.
	previous = current                            // Save the previous part.
	current, after, _ = strings.Cut(current, " ") // Cut the string by space.

	if current[0] != ':' { // Prefix field signature.
		irc_message_struct.Prefix = IrcMessagePrefix{} // No prefix.
		current = previous                             // Reset current.
	} else {
		// Try to parse prefix.
		prefix := current[1:]                             // Remove the ':' character.
		nickname, prefix, _ := strings.Cut(prefix, "!")   // Cut by '!', now we have nickname.
		username, hostname, _ := strings.Cut(prefix, "@") // Cut by '@', now we have username and hostname.

		irc_message_struct.Prefix = IrcMessagePrefix{ // Save the prefix.
			Nickname: nickname,
			Username: username,
			Hostname: hostname,
		}

		current = after // Move to the next part.
	}

	// Check command part.
	command, after, found := strings.Cut(current, " ") // Cut the string by space.
	if !found {
		// This is a one-command-only message.
		// Check if the command has CRLF at the end.
		if command[len(command)-2:] != "\r\n" {
			return IrcMessage{}, ErrorInvalidMessage // Invalid message.
		} else {
			// Remove the CRLF.
			command = command[:len(command)-2]
		}

		irc_message_struct.Command = command // Save the command.
		return irc_message_struct, nil       // Return the struct.
	}

	irc_message_struct.Command = command // Save the command.

	// We expect there's still more to parse.
	// After command, there is either channel name followed by message, or a bunch of parameters.

	// Cut the string by ':', now we have parameters and trailing message.
	param_string, trailing_message, found := strings.Cut(after, ":")

	if !found {
		// There's no trailing message.
		// Remove the trailing CRLF.
		param_string = after[:len(after)-2]
	} else {
		// There's a trailing message.
		// Remove the trailing CRLF.
		if trailing_message[len(trailing_message)-2:] == "\r\n" { // NOTE: There might no need to check this.
			trailing_message = trailing_message[:len(trailing_message)-2]
		}
		irc_message_struct.Message = trailing_message // Save the trailing message.
	}

	params := strings.Split(param_string, " ") // Split the parameters by space.
	irc_message_struct.Params = params         // Save the parameters.

	return irc_message_struct, nil // Return the struct.
}
