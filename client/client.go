package main

import (
	proto "Assignment-03/grpc"
	"Assignment-03/utility"
	"bufio"
	"bytes"
	"context"
	"encoding/gob"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"

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
	loggerChannel    chan utility.LogStruct
	stoppedChannel   chan bool
}

func main() {
	connect, err := grpc.NewClient("localhost:5050", grpc.WithTransportCredentials(insecure.NewCredentials()))

	//Done channel is for closing the connection to the server
	doneChannel := make(chan os.Signal, 1)
	signal.Notify(doneChannel, os.Interrupt)

	//Stopped channel is to close our log writer

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
		name = strings.TrimSpace(name)
		if errName != nil {
			log.Println(errName)
			fmt.Println("[Please input a valid name]")
			continue
		}
		break
	}
	//connect to the server
	user, _ := client.Connect(context.Background(), &proto.UserName{Name: strings.TrimSpace(name)})
	clientProcess := ClientProcess{clientProfile: user, timestampChannel: make(chan int64, 1), stoppedChannel: make(chan bool), loggerChannel: make(chan utility.LogStruct, 256), active: false}

	//Log connection to server event
	clientProcess.loggerChannel <- utility.LogStruct{Timestamp: clientProcess.clientProfile.Timestamp, Component: utility.CLIENT, EventType: utility.MESSAGE_RECEIVED, Identifier: clientProcess.clientProfile.Id}
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

			//Log recieved broadcasted message
			timestamp := utility.RemoteEvent(clientProcess.clientProfile, clientProcess.timestampChannel, msg.GetProcessTimestamp())
			clientProcess.loggerChannel <- utility.LogStruct{Timestamp: timestamp, Component: utility.CLIENT, EventType: utility.MESSAGE_RECEIVED, Identifier: msg.GetProcessId(), MessageContent: msg.GetMessage()}

			switch msg.GetMessageType() {
			case int64(utility.CONNECT), int64(utility.DISCONNECT):
				fmt.Println(msg.GetMessage())
				break
			case int64(utility.NORMAL):
				fmt.Println(utility.FormatMessage(msg.GetMessage(), msg.GetTimestamp(), msg.GetProcessName()))
				break
			default:
				//TODO error handling, should not receive other types!
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
				clientProcess.loggerChannel <- utility.LogStruct{Timestamp: timestamp, Component: utility.CLIENT, EventType: utility.CLIENT_DISCONNECT, Identifier: clientProcess.clientProfile.GetId()}
				//notify server of disconnect event
				client.Disconnect(context.Background(), clientProcess.clientProfile)
				clientProcess.stoppedChannel <- true
				os.Exit(1)
				return
			default:
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
					//Log the disconnection
					clientProcess.loggerChannel <- utility.LogStruct{Timestamp: timestamp, Component: utility.CLIENT, EventType: utility.CLIENT_DISCONNECT, Identifier: clientProcess.clientProfile.GetId()}
					//notify server of disconnect event
					client.Disconnect(context.Background(), clientProcess.clientProfile)
					clientProcess.stoppedChannel <- true
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

func (client *ClientProcess) Logger() {

	//Create the directory for the client logs
	err := os.Mkdir("ClientLogs", 0750)
	if err != nil {
		return
	}

	//Create the log file for each client
	f, err := os.OpenFile("Client"+strconv.FormatInt(client.clientProfile.GetId(), 10)+"log.txt", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		panic(err)
	}

	defer f.Close()

	//Create the logger object
	logger := log.New(io.Writer(f), "", 1)

	//Write the message to the file
	wg := sync.WaitGroup{}
	wg.Go(func() {
		for {
			select {
			case msg := <-client.loggerChannel:
				b := bytes.Buffer{}
				enc := gob.NewEncoder(&b)
				if err := enc.Encode(msg); err != nil {
					panic(err)
				}
				serialized := b.Bytes()
				_, err := logger.Writer().Write(serialized)
				if err != nil {
					panic(err)
				}
			case <-client.stoppedChannel:
				wg.Done()
				return
			default:
				continue
			}
		}
	})
	wg.Wait()
}
