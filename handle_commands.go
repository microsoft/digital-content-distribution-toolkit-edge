package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"
	"time"
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
	command := commandParams.GetCommandName()
	payload := commandParams.GetPayload()
	fmt.Println("Received from client- Command: ", command)
	fmt.Println("Received from client- Payload:", payload)
	switch(command) {
	case "Download":
		go handleDownload(payload)
	case "Delete":
		go handleDelete(payload)
	case "SetFilters":
		go handleSetFilters(payload)
	default:
		fmt.Println("Command not supported")
	}
	fmt.Println("Returning back the response to the proxy.....")
	return &pb.CommandServiceResponse{Code: 1, Message: "Recieved payload for " + command}, nil
}

func handleSetFilters(payload string) {
	time.Sleep(60 * time.Second)
	fmt.Println("in another thread after sleep")
}

func handleDelete(payload string) {

}

func handleDownload(payload string){

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
