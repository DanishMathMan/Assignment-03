package main

import (
	proto "Assignment-03/grpc"
	"Assignment-03/utility"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"time"

	context "golang.org/x/net/context"
	"google.golang.org/grpc"
)

type ChitChatServiceServer struct {
	proto.UnimplementedChitChatServiceServer
	serverProfile    *proto.Process
	timestampChannel chan int64
	chatMessages     []*proto.ChatMessage
	uuidNum          int64
	nextId           int64
	connectionPool   map[int64]Connection
	stoppedChannel   chan bool
}

/*
	Connection with inspiration from

https://medium.com/@bhadange.atharv/building-a-real-time-chat-application-with-grpc-and-go-aa226937ad3c
*/
type Connection struct {
	proto.UnimplementedChitChatServiceServer
	user           *proto.Process
	messageChan    chan *proto.ChatMessage
	connectionDown chan bool
}

func (server *ChitChatServiceServer) SendChat(ctx context.Context, in *proto.ChatMessage) (*proto.Empty, error) {
	//Logs receival of message
	tm := utility.RemoteEvent(server.serverProfile, server.timestampChannel, in.GetProcessTimestamp())
	utility.LogAsJson(utility.LogStruct{Timestamp: tm, Component: utility.SERVER, EventType: utility.MESSAGE_RECIEVED, Identifier: server.serverProfile.GetId(), MessageContent: in.GetMessage()}, true)

	wg := sync.WaitGroup{}
	//update timestamp of server in preparation for broadcasting message
	timestamp := utility.LocalEvent(server.serverProfile, server.timestampChannel)
	//before broadcasting message, update the processTimestamp field of the message as it indicates the time it was
	//broadcasted by the server allowing synchronization of timestamps in other process
	in.ProcessTimestamp = timestamp
	for _, conn := range server.connectionPool {
		wg.Go(func() {
			fmt.Printf("Sending message: %s to: %d \n", in.GetMessage(), conn.user.GetId())
			conn.messageChan <- in
		})
	}
	wg.Wait()

	//Logs when broadcasting a message to clients
	utility.LogAsJson(utility.LogStruct{Timestamp: timestamp, Component: utility.SERVER, EventType: utility.BROADCAST, Identifier: server.serverProfile.GetId(), MessageContent: in.GetMessage()}, true)
	return nil, nil
}

func (server *ChitChatServiceServer) Connect(ctx context.Context, in *proto.UserName) (*proto.Process, error) {
	//create a user and log the connection
	//update server timestamp. Note even though Connect is a rpc call and thus remote, the client has not an established
	//id or timestamp (which are provided by the server), thus this is treated as a local event
	timestamp := utility.LocalEvent(server.serverProfile, server.timestampChannel)

	//id of the new client process
	id := server.nextId
	server.nextId++
	//create a representation of the client process
	user := proto.Process{Id: id, Name: in.GetName(), Timestamp: timestamp}
	messageChannel := make(chan *proto.ChatMessage, 1)
	connection := Connection{user: &user, messageChan: messageChannel, connectionDown: make(chan bool, 1)}
	server.connectionPool[id] = connection

	//Logs when client is connected
	utility.LogAsJson(utility.LogStruct{Timestamp: timestamp, Component: utility.CLIENT, EventType: utility.CLIENT_CONNECTED, Identifier: id}, true)
	//increment timestamp in preparation for broadcasting message
	broadcastTimestamp := utility.LocalEvent(server.serverProfile, server.timestampChannel)
	broadcastMsg := proto.ChatMessage{
		Message:          utility.ConnectMessage(&user),
		Timestamp:        timestamp, //of when the connection happened
		ProcessId:        server.serverProfile.GetTimestamp(),
		ProcessName:      server.serverProfile.GetName(),
		MessageType:      int64(utility.CONNECT),
		ProcessTimestamp: broadcastTimestamp, //of when the server broadcasted the connection message
	}
	wg := sync.WaitGroup{}
	for _, conn := range server.connectionPool {
		wg.Go(func() {
			conn.messageChan <- &broadcastMsg
		})
	}
	wg.Wait()

	//Logs when client connection is broadcasted
	utility.LogAsJson(utility.LogStruct{Timestamp: broadcastTimestamp, Component: utility.SERVER, EventType: utility.BROADCAST, Identifier: server.serverProfile.GetId(), MessageContent: broadcastMsg.GetMessage()}, true)
	//return the user
	return &user, nil //error should
}

func (server *ChitChatServiceServer) Listen(client *proto.Process, stream grpc.ServerStreamingServer[proto.ChatMessage]) error {
	for {
		msg := <-server.connectionPool[client.GetId()].messageChan

		err := stream.Send(msg)
		if err != nil {
			fmt.Println("error in sending message")
		}
		if msg.GetMessageType() == int64(utility.SHUTDOWN) {
			server.connectionPool[client.GetId()].connectionDown <- true
			return nil
		}
	}
}

func (server *ChitChatServiceServer) Disconnect(ctx context.Context, in *proto.Process) (*proto.Empty, error) {
	//update the server timestamp
	timestamp := utility.RemoteEvent(server.serverProfile, server.timestampChannel, in.GetTimestamp())

	//Logs when client disconnects
	utility.LogAsJson(utility.LogStruct{Timestamp: timestamp, Component: utility.CLIENT, EventType: utility.CLIENT_DISCONNECT, Identifier: in.GetId()}, true)

	//increment timestamp in preparation for broadcasting message. This is done here because the sendChat method normally does this
	timestamp = utility.LocalEvent(server.serverProfile, server.timestampChannel)
	//create the message to be broadcast informing user disconnected
	msg := &proto.ChatMessage{
		Message:          utility.DisconnectMessage(in),
		Timestamp:        in.Timestamp, //the time at which the user disconnected
		ProcessId:        server.serverProfile.GetId(),
		ProcessName:      server.serverProfile.GetName(),
		MessageType:      int64(utility.DISCONNECT),
		ProcessTimestamp: timestamp, //the time at which the server registered the disconnect message
	}

	//Log broadcasting of server sending out disconnect message
	utility.LogAsJson(utility.LogStruct{Timestamp: timestamp, Component: utility.SERVER, EventType: utility.BROADCAST, Identifier: in.GetId(), MessageContent: msg.Message}, true)
	//broadcast user leaving
	//give the loggable information to the context, such that logging is done in one place in SendChat. This information
	//is the server sending a message to the clients informing them another client joined
	wg := sync.WaitGroup{}
	delete(server.connectionPool, in.GetId())
	for _, conn := range server.connectionPool {
		wg.Go(func() {
			conn.messageChan <- msg
		})
	}
	wg.Wait()

	return nil, nil
}

func main() {
	server := &ChitChatServiceServer{chatMessages: []*proto.ChatMessage{}}
	server.chatMessages = append(server.chatMessages, &proto.ChatMessage{})
	server.connectionPool = make(map[int64]Connection)
	serverId := server.nextId
	server.nextId++
	server.serverProfile = &proto.Process{Id: serverId, Name: "-----ChitChat-----", Timestamp: 0}
	server.timestampChannel = make(chan int64, 1)
	server.timestampChannel <- server.serverProfile.GetTimestamp()

	//Create the logger
	//go server.Logger()
	//Create the directory for the client logs
	err := os.Mkdir("ServerLogs", 0750)
	if err != nil && !os.IsExist(err) {
		log.Fatal(err)
	}
	f, err := os.OpenFile("ServerLogs/log_"+strconv.FormatInt(time.Now().Unix(), 10)+".json", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		log.Fatal(err)
	}

	//Gætter på det er her, da der er dato efterfulgt af "["
	log.SetFlags(0)
	log.SetOutput(f)
	log.Println("[")

	//Ensure stopping the server when we get the signal to stop the server
	doneChannel := make(chan os.Signal, 1)
	signal.Notify(doneChannel, os.Interrupt)

	wg := sync.WaitGroup{}
	wg.Go(func() {
		for {
			select {
			case <-doneChannel:
				//Logs when server is finished
				timestamp := utility.LocalEvent(server.serverProfile, server.timestampChannel)

				wg2 := sync.WaitGroup{}
				for _, conn := range server.connectionPool {
					wg2.Go(func() {
						conn.messageChan <- &proto.ChatMessage{Message: utility.ShutdownMessage(server.serverProfile),
							Timestamp:        timestamp,
							ProcessId:        server.serverProfile.GetId(),
							ProcessName:      server.serverProfile.GetName(),
							MessageType:      int64(utility.SHUTDOWN),
							ProcessTimestamp: timestamp}
					})
					<-conn.connectionDown
				}
				//Log broadcasting the shutdown to clients in server logs
				utility.LogAsJson(utility.LogStruct{Timestamp: timestamp, Component: utility.SERVER, EventType: utility.BROADCAST, Identifier: server.serverProfile.GetId(), MessageContent: "Server is shutting down. Goodbye!"}, true)

				//Log shutdown of server
				wg2.Wait()
				utility.LogAsJson(utility.LogStruct{Timestamp: timestamp, Component: utility.SERVER, EventType: utility.SERVER_STOP, Identifier: server.serverProfile.GetId()}, false)
				log.Println("]")
				time.Sleep(3 * time.Second)
				os.Exit(0)
				return
			}
		}
	})

	//Logs when server starts
	utility.LogAsJson(utility.LogStruct{Timestamp: 0, Component: utility.SERVER, EventType: utility.SERVER_START, Identifier: 0}, true)
	server.startServer()
	wg.Wait()
}

func (server *ChitChatServiceServer) startServer() {
	grpcServer := grpc.NewServer()
	listener, err := net.Listen("tcp", ":5050")
	if err != nil {
		log.Fatalf("Did not work")
	}
	proto.RegisterChitChatServiceServer(grpcServer, server)

	err = grpcServer.Serve(listener) // have to equal to something, runs and only returns an error if not work

	if err != nil {
		log.Fatalf("Did not work")
	}
}
