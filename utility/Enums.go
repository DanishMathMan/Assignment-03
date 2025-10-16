package utility

type Event string

const (
	CLIENT_CONNECTED    Event = "Connected"
	CLIENT_DISCONNECT   Event = "Disconnected"
	CLIENT_MESSAGE_SEND Event = "Message Send"
)

type ComponentType string

const (
	SERVER ComponentType = "[SERVER]"
	CLIENT ComponentType = "[CLIENT]"
)
