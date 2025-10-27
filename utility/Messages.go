package utility

import (
	proto "Assignment-03/grpc"
	"fmt"
	"unicode/utf8"
)

func FormatMessage(message string, timestamp int64, name string) string {
	return fmt.Sprintf("[%s at LT: %d] - %s ", name, timestamp, message)
}

func DisconnectMessage(in *proto.Process) string {
	return fmt.Sprintf("Participant %s left Chit Chat at logical time %d", in.GetName(), in.GetTimestamp())
}

func ConnectMessage(in *proto.Process) string {
	return fmt.Sprintf("Participant %s joined Chit Chat at logical time %d", in.GetName(), in.GetTimestamp())
}

// ValidMessage TODO should return an error
func ValidMessage(message string) bool {

	if len(message) > 128 {
		return false
	}

	if !utf8.ValidString(message) {
		return false
	}

	return true
}
