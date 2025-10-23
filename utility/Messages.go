package utility

import (
	proto "Assignment-03/grpc"
	"fmt"
)

func FormatMessage(chatMessage *proto.ChatMessage) string {
	return fmt.Sprintf("[Client: %d at LT: %d] - %s ", chatMessage.Client.Id, chatMessage.LogicalTimestamp, chatMessage.Message)
}
