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
	"strconv"
	"strings"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

type ClientProcess struct {
	clientProfile    *proto.Process
	timestampChannel chan int64
	messageChannel   chan *proto.ChatMessage
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
	//add metadata of id for a context
	md := metadata.Pairs("id", strconv.FormatInt(clientProcess.clientProfile.GetId(), 10))
	//create context for chat rpc call
	ctx := metadata.NewOutgoingContext(context.TODO(), md)
	stream, err := client.Chat(ctx)
	if err != nil {
		fmt.Printf("Error in Listen")
	}

	//go method for listening on stream for messages
	wg := sync.WaitGroup{}
	errChan := make(chan error, 1)
	wg.Go(func() {
		for {
			//take out the timestamp temporarily blocking other routines until message has been handled to prevent race conditions on timestamp
			timestamp := <-clientProcess.timestampChannel
			in, err := stream.Recv()
			if err == io.EOF {
				clientProcess.timestampChannel <- timestamp //nothing was received so it gets back the same timestamp
				continue
			}
			//TODO look into possibility of getting a special message that would indicate that the user disconnected or closed stream
			if err != nil {
				clientProcess.timestampChannel <- timestamp //an error was received so it gets back the same timestamp TODO should probably be incremented and used for log
				errChan <- err
				return
			}
			//remote message was well received, increment lamport clock accordingly
			timestamp = max(timestamp, in.GetTimestamp()) + 1
			clientProcess.timestampChannel <- timestamp
			fmt.Println(in.GetMessage())
		}
	})

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
			if err := stream.Send(&chatMessage); err != nil {
				errChan <- err
				return
			}
		}
	}()

	wg.Wait()
	log.Println(<-errChan)
}
