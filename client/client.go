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
	"os/signal"
	"strconv"
	"strings"
	"time"

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

	//Done channel is for closing the connection to the server
	doneChannel := make(chan os.Signal, 1)
	signal.Notify(doneChannel, os.Interrupt)

	//Create the directory for the client logs
	err2 := os.Mkdir("ClientLogs", 0750)
	if err2 != nil && !os.IsExist(err2) {
		panic(err2)
	}

	//Stopped channel is to close our log writer

	if err != nil {
		panic(err)
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
		name = strings.TrimSpace(name)
		if errName != nil {
			fmt.Println("[Please input a valid name]")
			continue
		}
		break
	}

	//connect to the server
	user, _ := client.Connect(context.Background(), &proto.UserName{Name: strings.TrimSpace(name)})
	clientProcess := ClientProcess{clientProfile: user, timestampChannel: make(chan int64, 1)}

	//Setup logging file and ensure open file
	f, err := os.OpenFile("ClientLogs/Client"+strconv.FormatInt(clientProcess.clientProfile.GetId(), 10)+"_"+strconv.FormatInt(time.Now().Unix(), 10)+".json", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		log.Fatal(err)
	}
	log.SetFlags(0)
	log.SetOutput(f)
	log.Println("[")

	//Log connection to server event
	utility.LogAsJson(utility.LogStruct{Timestamp: clientProcess.clientProfile.GetTimestamp(), Component: utility.CLIENT, EventType: utility.CLIENT_CONNECTED, Identifier: clientProcess.clientProfile.GetId()}, true)
	clientProcess.timestampChannel <- clientProcess.clientProfile.GetTimestamp()

	//go routine for listening for messages from the server using a stream
	go func() {
		//we consider the Listen rpc to be happening at the same timestamp as the connection,
		//because of the asynchronous nature of the server and client go routines, but ideally should be logged separately
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

			//Log recieved broadcasted message
			timestamp := utility.RemoteEvent(clientProcess.clientProfile, clientProcess.timestampChannel, msg.GetProcessTimestamp())
			utility.LogAsJson(utility.LogStruct{Timestamp: timestamp, Component: utility.CLIENT, EventType: utility.MESSAGE_RECIEVED, Identifier: msg.GetProcessId(), MessageContent: msg.GetMessage()}, true)

			switch msg.GetMessageType() {
			case int64(utility.CONNECT), int64(utility.DISCONNECT):
				fmt.Println(msg.GetMessage())
				break
			case int64(utility.NORMAL):
				fmt.Println(utility.FormatMessage(msg.GetMessage(), msg.GetTimestamp(), msg.GetProcessName()))
				break
			case int64(utility.SHUTDOWN):
				fmt.Println(msg.GetMessage())
				timestamp = utility.LocalEvent(clientProcess.clientProfile, clientProcess.timestampChannel)
				utility.LogAsJson(utility.LogStruct{Timestamp: timestamp, Component: utility.SERVER, EventType: utility.SERVER_STOP, Identifier: msg.GetProcessId()}, false)
				log.Println("]")
				os.Exit(1)
			default:
				panic("Unknown message type")
			}
		}
	}()

	//go routine listening for a user input (message) to send to the server
	go func() {
		for {
			select {
			case <-doneChannel:
				//timestamp the local event of closing
				timestamp := utility.LocalEvent(clientProcess.clientProfile, clientProcess.timestampChannel)
				fmt.Println("Bye")

				//Log the disconnection
				utility.LogAsJson(utility.LogStruct{Timestamp: timestamp, Component: utility.CLIENT, EventType: utility.CLIENT_DISCONNECT, Identifier: clientProcess.clientProfile.GetId()}, false)
				log.Println("]")
				//notify server of disconnect event
				client.Disconnect(context.Background(), clientProcess.clientProfile)
				os.Exit(1)
				return
			default:
				reader := bufio.NewReader(os.Stdin)
				msg, errSend := reader.ReadString('\n')
				if errSend != nil {
					//should ideally be handled
					continue
				}
				if !utility.ValidMessage(msg) {
					fmt.Println("[Message is too long]") //TODO better error message
				}
				timestamp := utility.LocalEvent(clientProcess.clientProfile, clientProcess.timestampChannel)
				if strings.Contains(msg, "--exit") {
					fmt.Println("Bye")
					//Log the disconnection
					utility.LogAsJson(utility.LogStruct{Timestamp: timestamp, Component: utility.CLIENT, EventType: utility.CLIENT_DISCONNECT, Identifier: clientProcess.clientProfile.GetId()}, false)
					log.Println("]")
					//notify server of disconnect event
					client.Disconnect(context.Background(), clientProcess.clientProfile)
					os.Exit(0)
				}
				chatMessage := proto.ChatMessage{
					Message:          msg,
					Timestamp:        timestamp,
					ProcessId:        clientProcess.clientProfile.GetId(),
					ProcessName:      clientProcess.clientProfile.GetName(),
					MessageType:      int64(utility.NORMAL),
					ProcessTimestamp: timestamp,
				}
				client.SendChat(context.Background(), &chatMessage)
			}
		}
	}()

	select {}

}
