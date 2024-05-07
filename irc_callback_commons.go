package irc

func pingHandler(ircc *IrcClient, msg string) error {
	parsed_message, err := ParseIrcMessage(msg)
	if err != nil {
		return err
	}

	if parsed_message.Command == "PING" {
		response := IrcMessage{
			Command: "PONG",
			Message: parsed_message.Message,
		}
		ircc.SendRawMessage(response.String())
	}
	return nil
}
