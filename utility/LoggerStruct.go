package utility

type LogStruct struct {
	Timestamp      int64
	Component      ComponentType
	EventType      Event
	Identifier     int64 //identity interpretation is dependent on event type and component type
	MessageContent string
}
