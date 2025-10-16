package main

import (
	proto "Assignment-03/grpc"
	"fmt"
	"log"
	"net"

	context "golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ChitChatServiceServer struct {
	proto.UnimplementedChitChatServiceServer
	chatMessages []*proto.ChatMessage // hello :) no :C wup wup wup :'(
}

func (server *ChitChatServiceServer) SendChat(ctx context.Context, in *proto.ChatMessage) (*proto.Empty, error) {

	fmt.Println(in.Message)

	return nil, nil
}

func (server *ChitChatServiceServer) Connect(client *proto.Client, stream proto.ChitChatService_ConnectServer) error {
	//for _, msg := range server.chatMessages {
	//}
	//return status.Errorf(codes.Unimplemented, "method Connect not implemented")
	return nil
}

func (server *ChitChatServiceServer) Disconnect(ctx context.Context, in *proto.Client) (*proto.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Disconnect not implemented")
}

func main() {
	server := &ChitChatServiceServer{chatMessages: []*proto.ChatMessage{}}
	server.chatMessages = append(server.chatMessages, &proto.ChatMessage{})

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
