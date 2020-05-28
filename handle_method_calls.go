package main

import (
    "net"
	"context"

	"fmt"
	"log"
	"sync"
	"strings"
	"path/filepath"

	"io"
	"net/http"
	"os"

	"google.golang.org/grpc"

	pb "./pbcommands"
)



type relayCommandServer struct {
	pb.UnimplementedRelayCommandServer
}

func DownloadFile(filepath string, url string) error {
	// This is just a place holder, Archie will replace it
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

func (s *relayCommandServer) Download(ctx context.Context, download_params *pb.DownloadParams) (*pb.Response, error) {
	log.Println(download_params.GetFolderpath());
	hierarchy := strings.Split(strings.Trim(download_params.GetFolderpath(), "/"), "/")
	log.Println(hierarchy)
	folder_name, err := fs.CreateFolder(hierarchy)
	if(err != nil) {
		logger.Log("Error", fmt.Sprintf("%s", err))
		return &pb.Response{Responsemessage: "Folder not downloaded"}, err
	}

    log.Println("metadata files >")
    for i, x := range(download_params.GetMetadatafiles()) {
		log.Println("\t", (*x).Name)
		err := DownloadFile(filepath.Join(fs.GetHomeFolder(), folder_name, fmt.Sprintf("doge_metadata%d.jpg", i)), "https://wallpaperplay.com/walls/full/0/8/0/1532.jpg")
		if(err != nil) {
			logger.Log("Error", fmt.Sprintf("%s", err))
			return &pb.Response{Responsemessage: "Folder not downloaded"}, err
		}
    }

    log.Println("bulk files >")
    for i, x := range(download_params.GetBulkfiles()) {
		log.Println("\t", (*x).Name)
		err := DownloadFile(filepath.Join(fs.GetHomeFolder(), folder_name, fmt.Sprintf("doge_bulk%d.jpg", i)), "https://wallpaperplay.com/walls/full/0/8/0/1532.jpg")
		if(err != nil) {
			logger.Log("Error", fmt.Sprintf("%s", err))
			return &pb.Response{Responsemessage: "Folder not downloaded"}, err
		}
    }

    log.Println("")
	fs.PrintBuckets()
	fs.PrintFileSystem()
    log.Println("")

	return &pb.Response{Responsemessage: "Folder downloaded"}, nil
}

func (s *relayCommandServer) Delete(ctx context.Context, delete_params *pb.DeleteParams) (*pb.Response, error) {
	folder_path := delete_params.GetFolderpath()
	log.Println(folder_path);
	if(delete_params.GetRecursive()) {
		err := fs.RecursiveDeleteFolder(strings.Split(folder_path, "/"))
		if(err != nil) {
			logger.Log("Error", fmt.Sprintf("%s", err))
			return &pb.Response{Responsemessage: "Folder not deleted"}, err
		}
	} else {
		err := fs.DeleteFolder(strings.Split(folder_path, "/"))
		if(err != nil) {
			logger.Log("Error", fmt.Sprintf("%s", err))
			return &pb.Response{Responsemessage: "Folder not deleted"}, err
		}
	}

	return &pb.Response{Responsemessage: "Folder deleted"}, nil
}

func newServer() *relayCommandServer {
	s := &relayCommandServer{}
	return s
}

func handle_method_calls(port int, wg sync.WaitGroup) {
	lis, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", port))
    if err != nil {
        log.Fatalf("failed to listen: %v", err)
    }

	defer wg.Done()
	grpcServer := grpc.NewServer()
	pb.RegisterRelayCommandServer(grpcServer, newServer())
	grpcServer.Serve(lis)
}