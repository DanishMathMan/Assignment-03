package utility

import (
	"encoding/json"
	"log"
)

type LogStruct struct {
	Timestamp      int64
	Component      ComponentType
	EventType      Event
	Identifier     int64 //identity interpretation is dependent on event type and component type
	MessageContent string
}

// LogAsJson logs a log struct as a json string to the standard logger which in our case is the log file
func LogAsJson(logStruct LogStruct, trailingComma bool) {
	j, err := json.MarshalIndent(logStruct, "", "\t")
	if err != nil {
		log.Panicln(err)
	}
	w := log.Writer()
	w.Write(j)
	var trail string
	if trailingComma {
		trail = ",\n"
	} else {
		trail = "\n"
	}
	w.Write([]byte(trail))
}
