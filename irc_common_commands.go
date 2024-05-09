package irc

func PRIVMSG(channel string, message string) IrcMessage {
	return IrcMessage{
		Command: "PRIVMSG",
		Params:  []string{channel + " "},
		Message: message,
	}
}

func JOIN(channel string) IrcMessage {
	return IrcMessage{
		Command: "JOIN",
		Params:  []string{"#" + channel},
	}
}

func PING(message string) IrcMessage {
	return IrcMessage{
		Command: "PING",
		Message: message,
	}
}
