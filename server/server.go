package main

import (
	proto "Assignment-03/grpc"
	"Assignment-03/utility"
	"io"
	"log"
	"net"
	"strconv"
	"sync"

	context "golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type ChitChatServiceServer struct {
	proto.UnimplementedChitChatServiceServer
	serverProfile    *proto.Process
	timestampChannel chan int64
	chatMessages     []*proto.ChatMessage
	uuidNum          int64
	nextId           int64
	connectionPool   map[int64]*Connection
}

/*
	Connection with inspiration from

https://medium.com/@bhadange.atharv/building-a-real-time-chat-application-with-grpc-and-go-aa226937ad3c
*/
type Connection struct {
	proto.UnimplementedChitChatServiceServer
	stream            proto.ChatMessage
	user              *proto.Process
	clientMessageChan chan *proto.ChatMessage
	serverMessageChan chan *proto.ChatMessage
}

func (server *ChitChatServiceServer) Chat(stream proto.ChitChatService_ChatServer) error {
	//get the meta data associated with the stream, to get its id associated with it
	ctx := stream.Context()
	var id int64
	var err error
	md, ok := metadata.FromIncomingContext(ctx)
	if ok {
		id, err = strconv.ParseInt(md.Get("id")[0], 10, 64)
	}
	if err != nil {
		return err
	}

	wg := sync.WaitGroup{}
	errChan := make(chan error)

	//Go routine listening on message from client
	wg.Go(func() {
		for {
			//take out the timestamp temporarily blocking other routines until message has been handled to prevent race conditions on timestamp
			timestamp := <-server.timestampChannel
			in, err := stream.Recv()
			if err == io.EOF {
				server.timestampChannel <- timestamp //nothing was received so it gets back the same timestamp
				errChan <- nil
				return
			}
			//TODO look into possibility of getting a special message that would indicate that the user disconnected or closed stream
			if err != nil {
				server.timestampChannel <- timestamp //an error was received so it gets back the same timestamp TODO should probably be incremented and used for log
				errChan <- err
				return
			}
			//remote message was well received, increment lamport clock accordingly
			timestamp = max(timestamp, in.GetTimestamp()) + 1
			server.timestampChannel <- timestamp
			//put message from client on its connection channel so the server has access to it
			server.connectionPool[id].clientMessageChan <- in
		}
	})

	//go routine to send the message on the clientMessageChan to the other connections' channels
	wg.Go(func() {
		wg2 := sync.WaitGroup{}
		for {
			msg := <-server.connectionPool[id].clientMessageChan
			//for each connection, start a temporary go routine to put the message on its channel
			for _, conn := range server.connectionPool {
				wg2.Go(func() {
					conn.serverMessageChan <- msg
				})
			}
			//TODO make a shutdown channel and get its value here in a select/default to stop the outer for loop in this go routine
		}
		//wait for the last messages to be sent
		wg2.Wait()
	})

	//Go routine sending messages to client
	wg.Go(func() {
		for {
			msg := <-server.connectionPool[id].serverMessageChan
			//take out the timestamp temporarily blocking other routines until message has been handled to prevent race conditions on timestamp
			timestamp := <-server.timestampChannel
			if err = stream.Send(msg); err != nil {
				server.timestampChannel <- timestamp //an error was received so it gets back the same timestamp TODO should probably be incremented and used for log
				errChan <- err
				return
			}
			//message was sent successfully, increment lamport clock accordingly
			server.timestampChannel <- timestamp + 1
		}
	})
	//TODO defer logging of error
	return <-errChan
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
	clientMessageChannel := make(chan *proto.ChatMessage, 1)
	serverMessageChannel := make(chan *proto.ChatMessage, 1)
	connection := Connection{user: &user, clientMessageChan: clientMessageChannel, serverMessageChan: serverMessageChannel}
	server.connectionPool[id] = &connection
	//create message which is to be broadcasted to all
	broadcastMsg := proto.ChatMessage{
		Message:     utility.ConnectMessage(&user),
		Timestamp:   timestamp,
		ProcessId:   server.serverProfile.GetTimestamp(),
		ProcessName: server.serverProfile.GetName(),
	}

	//put the message on all connections
	wg := sync.WaitGroup{}
	for _, conn := range server.connectionPool {
		wg.Go(func() {
			conn.serverMessageChan <- &broadcastMsg
		})
	}
	wg.Wait()
	//return the user
	return &user, nil //error should be different
}

func (server *ChitChatServiceServer) Disconnect(ctx context.Context, in *proto.Process) (*proto.Empty, error) {
	//TODO log event
	//update the server timestamp
	//timestamp := utility.RemoteEvent(server.serverProfile, server.timestampChannel, in.Timestamp)
	//create the message to be broadcast informing user disconnected
	//msg := &proto.ChatMessage{
	//	Message:     utility.DisconnectMessage(in),
	//	Timestamp:   timestamp,
	//	ProcessId:   server.serverProfile.GetId(),
	//	ProcessName: server.serverProfile.GetName()}
	//broadcast user leaving
	//server.SendChat(ctx, msg)
	return nil, nil
}

func main() {
	server := &ChitChatServiceServer{chatMessages: []*proto.ChatMessage{}}
	server.chatMessages = append(server.chatMessages, &proto.ChatMessage{})
	server.connectionPool = make(map[int64]*Connection)
	serverId := server.nextId
	server.nextId++
	server.serverProfile = &proto.Process{Id: serverId, Name: "-----ChitChat-----", Timestamp: 0}
	server.timestampChannel = make(chan int64, 1)
	server.timestampChannel <- server.serverProfile.GetTimestamp()
	//TODO log server startup
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
