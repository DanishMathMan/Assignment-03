package main

import (
	proto "Assignment-03/grpc"
	"Assignment-03/utility"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"

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
	loggerChannel    chan utility.LogStruct
}

/*
	Connection with inspiration from

https://medium.com/@bhadange.atharv/building-a-real-time-chat-application-with-grpc-and-go-aa226937ad3c
*/
type Connection struct {
	proto.UnimplementedChitChatServiceServer
	stream      proto.ChitChatService_ListenClient
	user        *proto.Process
	messageChan chan *proto.ChatMessage
}

func (server *ChitChatServiceServer) SendChat(ctx context.Context, in *proto.ChatMessage) (*proto.Empty, error) {
	//log receival of message
	utility.RemoteEvent(server.serverProfile, server.timestampChannel, in.GetProcessTimestamp())
	//TODO use context if there is any information in it. Finally log the event
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
	connection := Connection{user: &user, messageChan: messageChannel}
	server.connectionPool[id] = connection

	server.loggerChannel <- utility.LogStruct{Timestamp: timestamp, Component: utility.CLIENT, EventType: utility.CLIENT_CONNECTED, Identifier: id}
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
	//return the user
	return &user, nil //error should
}

func (server *ChitChatServiceServer) Listen(client *proto.Process, stream grpc.ServerStreamingServer[proto.ChatMessage]) error {

	// TODO listen on message channel and send to stream what the message was
	for {
		msg := <-server.connectionPool[client.GetId()].messageChan
		err := stream.Send(msg)
		if err != nil {
			fmt.Println("error in sending message")
		}
	}
}

func (server *ChitChatServiceServer) Disconnect(ctx context.Context, in *proto.Process) (*proto.Empty, error) {
	//update the server timestamp
	timestamp := utility.RemoteEvent(server.serverProfile, server.timestampChannel, in.GetTimestamp())

	server.loggerChannel <- utility.LogStruct{Timestamp: timestamp, Component: utility.CLIENT, EventType: utility.CLIENT_DISCONNECT, Identifier: in.GetId()}

	//create the message to be broadcast informing user disconnected
	msg := &proto.ChatMessage{
		Message:          utility.DisconnectMessage(in),
		Timestamp:        in.Timestamp, //the time at which the user disconnected
		ProcessId:        server.serverProfile.GetId(),
		ProcessName:      server.serverProfile.GetName(),
		MessageType:      int64(utility.DISCONNECT),
		ProcessTimestamp: timestamp, //the time at which the server registered the disconnect message
	}
	//broadcast user leaving
	//give the loggable information to the context, such that logging is done in one place in SendChat. This information
	//is the server sending a message to the clients informing them another client joined
	wg := sync.WaitGroup{}
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
	server.loggerChannel = make(chan utility.LogStruct, 100)

	//Ensure stopping the server when we get the signal to stop the server
	doneChannel := make(chan os.Signal, 1)
	signal.Notify(doneChannel, os.Interrupt)

	select {
	case <-doneChannel:
		server.loggerChannel <- utility.LogStruct{Timestamp: 0, Component: utility.SERVER, EventType: utility.SERVER_STOP, Identifier: 0}
		os.Exit(0)
		return
	default:

	}

	server.loggerChannel <- utility.LogStruct{Timestamp: 0, Component: utility.SERVER, EventType: utility.SERVER_START, Identifier: 0}

	server.startServer()
}

func (server *ChitChatServiceServer) startServer() {
	grpcServer := grpc.NewServer()
	listener, err := net.Listen("tcp", ":5050")
	if err != nil {
		log.Fatalf("Did not work")
	}

	log.Printf("ITU server listening at %v", listener.Addr())

	proto.RegisterChitChatServiceServer(grpcServer, server)

	err = grpcServer.Serve(listener) // have to equal to something, runs and only returns an error if not work

	if err != nil {
		log.Fatalf("Did not work")
	}
}

//TODO SERVER SHUTDOWN AND START UP and how it is logged and notified to clients without crashes or errors

func (server *ChitChatServiceServer) Logger() {

}
