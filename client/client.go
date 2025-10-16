package main

import (
	proto "Assignment-03/grpc"
	"Assignment-03/utility"
	"bufio"
	"context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"log"
	"os"
)

//type ChitChatServiceClient struct {
//	proto.UnimplementedChitChatServiceServer
//	message []string
//}

//Client should be able to make an ID
//Client should connect to the server via rpc Connect and get a stream of chat messages
//Client should write a message to the server via rpc SendChat
//Client should disconnect from the server via rpc Disconnect

func main() {
	connect, err := grpc.NewClient("localhost:5050", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Not working")
	}
	defer connect.Close()

	// generate connection as new client
	client := proto.NewChitChatServiceClient(connect)

	//go routine listening for a user input (message) to send to the server
	go func() {
		for {
			reader := bufio.NewReader(os.Stdin)
			msg, errSend := reader.ReadString('\n')
			if errSend != nil {
				log.Println(errSend)
			}
			time_stamp := 1 //place holder
			logMessage := proto.LogMessage{
				ComponentName:    string(utility.CLIENT),
				LogicalTimestamp: int64(time_stamp),
				EventType:        string(utility.CLIENT_MESSAGE_SEND)}
			chatMessage := proto.ChatMessage{Message: msg, LogicalTimestamp: int64(time_stamp), LogMessage: &logMessage}
			client.SendChat(context.Background(), &chatMessage)
		}
	}()

	//go routine listening for server messages
	go func() {
		//HEY LISTEN!
		for {

		}
	}()

	//TODO REFACTOR
	for {

	}

}
