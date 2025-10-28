package utility

type Event string

const (
	CLIENT_CONNECTED  Event = "Connected"
	CLIENT_DISCONNECT Event = "Disconnected"
	SERVER_START      Event = "Server Started"
	SERVER_STOP       Event = "Server Stopped"
	MESSAGE_SEND      Event = "Message Send"
	MESSAGE_RECEIVED  Event = "Message Recieved"
)

type ComponentType string

const (
	SERVER ComponentType = "[SERVER]"
	CLIENT ComponentType = "[CLIENT]"
)

type MessageType int64

const (
	NORMAL     MessageType = 0
	CONNECT    MessageType = 1
	DISCONNECT MessageType = 2
)
