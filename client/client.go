package main

import (
	proto "Assignment-03/grpc"
	"Assignment-03/utility"
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

//Client should be able to make an ID
//Client should connect to the server via rpc Connect and get a stream of chat messages
//Client should write a message to the server via rpc SendChat
//Client should disconnect from the server via rpc Disconnect

type ClientProcess struct {
	clientProfile    *proto.Process
	timestampChannel chan int64
	active           bool
}

func main() {
	connect, err := grpc.NewClient("localhost:5050", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Not working")
	}
	defer connect.Close()

	// generate connection as new client
	client := proto.NewChitChatServiceClient(connect)
	// ask for a username:
	fmt.Println("Enter your screen name:")
	reader := bufio.NewReader(os.Stdin)
	var name string
	var errName error
	for {
		name, errName = reader.ReadString('\n')
		if errName != nil {
			log.Println(errName)
			fmt.Println("[Please input a valid name]")
			continue
		}
		break
	}
	//connect to the server
	user, _ := client.Connect(context.Background(), &proto.UserName{Name: strings.TrimSpace(name)})
	clientProcess := ClientProcess{clientProfile: user, timestampChannel: make(chan int64, 1)}
	clientProcess.timestampChannel <- clientProcess.clientProfile.Timestamp
	//go routine for listening for messages from the server using a stream
	go func() {
		stream, err := client.Listen(context.Background(), clientProcess.clientProfile)
		//make sure context is properly shut down
		defer stream.Context().Done()
		if err != nil {
			fmt.Printf("Error in Listen")
		}
		for {
			msg, err := stream.Recv()

			if err == io.EOF {
				continue
			}
			if err != nil {
				break
			}
			fmt.Println(msg.GetMessage())
			utility.RemoteEvent(clientProcess.clientProfile, clientProcess.timestampChannel, msg.GetTimestamp())
		}
	}()

	//go routine listening for a user input (message) to send to the server
	go func() {
		for {
			reader := bufio.NewReader(os.Stdin)
			msg, errSend := reader.ReadString('\n')
			if errSend != nil {
				log.Println(errSend)
			}
			if !utility.ValidMessage(msg) {
				fmt.Println("[Message is too long]") //TODO better error message
			}
			timestamp := utility.LocalEvent(clientProcess.clientProfile, clientProcess.timestampChannel)
			if strings.Contains(msg, "--exit") {
				fmt.Println("Bye")
				//notify server of disconnect event
				client.Disconnect(context.Background(), clientProcess.clientProfile)
				os.Exit(0)
			}
			chatMessage := proto.ChatMessage{
				Message:     utility.FormatMessage(msg, timestamp, clientProcess.clientProfile.Name),
				Timestamp:   timestamp,
				ProcessId:   clientProcess.clientProfile.GetId(),
				ProcessName: clientProcess.clientProfile.GetName()}
			client.SendChat(context.Background(), &chatMessage)
		}
	}()

	select {}
}
