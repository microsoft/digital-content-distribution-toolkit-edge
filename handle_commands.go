package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"

	pb "./DownstreamCommands"
	"google.golang.org/grpc"
)

type server struct {
	pb.UnimplementedCommandServer
}

func newCommandServer() *server {
	s := &server{}
	return s
}

func (s *server) ReceiveCommand(ctx context.Context, commandParams *pb.CommandServiceRequest) (*pb.CommandServiceResponse, error) {
	fmt.Println("Received from client- Command: ", commandParams.GetCommandName())
	fmt.Println("Received from client- Payload:", commandParams.GetPayload())
	return &pb.CommandServiceResponse{Code: 1, Message: "Recieved payload for " + commandParams.GetCommandName()}, nil
}

func handleCommands(port int, wg sync.WaitGroup) {
	lis, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	fmt.Printf("[Command Server] Listening for commands on port %d .......\n", port)
	defer wg.Done()
	grpcServer := grpc.NewServer()
	pb.RegisterCommandServer(grpcServer, newCommandServer())
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
