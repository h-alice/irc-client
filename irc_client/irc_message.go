package irc_client

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
