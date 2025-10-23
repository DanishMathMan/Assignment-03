package main

import (
	proto "Assignment-03/grpc"
	"errors"
	"fmt"
	"log"
	"net"
	"sync"
	"unicode/utf8"

	context "golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ChitChatServiceServer struct {
	proto.UnimplementedChitChatServiceServer
	chatMessages     []*proto.ChatMessage
	logicalTimestamp int
	uuidNum          int32
	nextId           int32
	connectionPool   map[int32]Connection
	wg               sync.WaitGroup
}

/*
	Connection with inspiration from

https://medium.com/@bhadange.atharv/building-a-real-time-chat-application-with-grpc-and-go-aa226937ad3c
*/
type Connection struct {
	proto.UnimplementedChitChatServiceServer
	stream      proto.ChitChatService_ListenClient
	user        *proto.User
	messageChan chan *proto.ChatMessage
}

func (server *ChitChatServiceServer) SendChat(ctx context.Context, in *proto.ChatMessage) (*proto.Empty, error) {

	fmt.Println(in.Message)

	//Check validity of message
	if !validMessage(in) {
		return nil, errors.New("invalid message")
	}

	for _, conn := range server.connectionPool {
		fmt.Printf("Sending message: %s to: %d \n", in.Message, conn.user.Id)
		conn.messageChan <- in
	}
	return nil, nil
}

func (server *ChitChatServiceServer) Connect(ctx context.Context, in *proto.Empty) (*proto.User, error) {
	//create a user and log the connection
	id := server.nextId
	server.nextId++
	user := proto.User{Id: id}
	channel := make(chan *proto.ChatMessage, 1)
	connection := Connection{user: &user, messageChan: channel}
	server.connectionPool[id] = connection
	//return the user
	return &user, nil //error should
}

func (server *ChitChatServiceServer) Listen(client *proto.User, stream grpc.ServerStreamingServer[proto.ChatMessage]) error {

	// TODO listen on message channel and send to stream what the message was
	for {
		msg := <-server.connectionPool[client.Id].messageChan

		fmt.Printf("Message in channel: %s \n", msg.Message)

		//TODO check if message is a closing message i.e. the client has disconnected, then stop go routine if it is
		err := stream.Send(msg)
		if err != nil {
			fmt.Println("error in sending message")
		}

		fmt.Println("Send message via stream")
	}
}

func (server *ChitChatServiceServer) Disconnect(ctx context.Context, in *proto.User) (*proto.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Disconnect not implemented")
}

func main() {
	server := &ChitChatServiceServer{chatMessages: []*proto.ChatMessage{}}
	server.chatMessages = append(server.chatMessages, &proto.ChatMessage{})
	server.connectionPool = make(map[int32]Connection)
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

func validMessage(message *proto.ChatMessage) bool {

	if len(message.Message) > 128 {
		return false
	}

	if !utf8.ValidString(message.Message) {
		return false
	}

	return true
}
