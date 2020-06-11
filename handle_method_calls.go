package main

import (
	"context"
	"io/ioutil"
	"net"
	"path/filepath"
	"strconv"

	"fmt"
	"log"
	"strings"
	"sync"

	"io"
	"net/http"
	"os"

	"google.golang.org/grpc"

	pb "./pbcommands"
)

type relayCommandServer struct {
	pb.UnimplementedRelayCommandServer
}

func DownloadFile(filepath, url, actualSha256 string) error {
	// This is just a place holder, Archie will replace it
	println("Downloading file from : " + url)
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	fileOutputStream, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer fileOutputStream.Close()
	fileLengthString := resp.Header.Get("Content-Length")
	fileLength, err := strconv.Atoi(fileLengthString)
	progressWriter := &ProgressWriter{}
	progressWriter.Total = int64(fileLength / 1024 / 1024)
	_, err = io.Copy(fileOutputStream, io.TeeReader(resp.Body, progressWriter))
	return err
}

func (s *relayCommandServer) Download(ctx context.Context, download_params *pb.DownloadParams) (*pb.Response, error) {
	log.Println(download_params.GetFolderpath())
	hierarchy := strings.Split(strings.Trim(download_params.GetFolderpath(), "/"), "/")
	log.Println(hierarchy)
	// Assuming one node created at a time
	folderName, err := fs.CreateFolder(hierarchy)
	if err != nil {
		logger.Log("Error", fmt.Sprintf("%s", err))
		return &pb.Response{Responsemessage: "Folder not downloaded"}, err
	}
	tempFolder := "tmp"
	log.Println("metadata files >")
	for i, x := range download_params.GetMetadatafiles() {
		log.Println("\t", (*x).Name)
		//err := DownloadFile(filepath.Join(fs.GetHomeFolder(), folder_name, fmt.Sprintf("doge_metadata%d.jpg", i)), "https://wallpaperplay.com/walls/full/0/8/0/1532.jpg")
		err := DownloadFile(filepath.Join(fs.GetHomeDirLocation(), tempFolder, (*x).Name), (*x).Cdn, (*x).Hashsum)
		if err != nil {
			logger.Log("Error", fmt.Sprintf("%s", err))
			//delete temp folder
			delete_folder(filepath.Join(fs.GetHomeDirLocation(), tempFolder))
			return &pb.Response{Responsemessage: "Folder not downloaded"}, err
		}
	}
	//movefile
	if err := moveFiles(filepath.Join(fs.GetHomeDirLocation(), tempFolder), download_params.GetFolderpath(), "metadatafile"); err != nil {
		logger.Log("Error", fmt.Sprintf("%s", err))
		return &pb.Response{Responsemessage: "Could mot move temp folder to filsys"}, err
	}

	log.Println("bulk files >")
	for i, x := range download_params.GetBulkfiles() {
		log.Println("\t", (*x).Name)
		err := DownloadFile(filepath.Join(fs.GetHomeDirLocation(), tempFolder, (*x).Name), (*x).Cdn, (*x).Hashsum)
		if err != nil {
			logger.Log("Error", fmt.Sprintf("%s", err))
			delete_folder(filepath.Join(fs.GetHomeDirLocation(), tempFolder))
			return &pb.Response{Responsemessage: "Folder not downloaded"}, err
		}
	}
	if err := moveFiles(tempFolder, download_params.GetFolderpath(), "bulkfile"); err != nil {
		logger.Log("Error", fmt.Sprintf("%s", err))
		return &pb.Response{Responsemessage: "Could not move temp folder to filsys"}, err
	}
	log.Println("")
	fs.PrintBuckets()
	fs.PrintFileSystem()
	log.Println("")

	return &pb.Response{Responsemessage: "Folder downloaded"}, nil
}
func moveFiles(sourceFolder, destFolder, fileType string) error {
	files, err := ioutil.ReadDir(sourceFolder)
	if err != nil {
		return err
	}
	for _, file := range files {
		err := fs.MoveFile(filepath.Join(sourceFolder, file.Name()), destFolder, fileType)
		if err != nil {
			return err
		}
	}
	return nil
}
func delete_folder(path string) error {
	err := os.RemoveAll(path)
	return err
}

func (s *relayCommandServer) Delete(ctx context.Context, delete_params *pb.DeleteParams) (*pb.Response, error) {
	folder_path := delete_params.GetFolderpath()
	log.Println(folder_path)
	if delete_params.GetRecursive() {
		err := fs.RecursiveDeleteFolder(strings.Split(folder_path, "/"))
		if err != nil {
			logger.Log("Error", fmt.Sprintf("%s", err))
			return &pb.Response{Responsemessage: "Folder not deleted"}, err
		}
	} else {
		err := fs.DeleteFolder(strings.Split(folder_path, "/"))
		if err != nil {
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
