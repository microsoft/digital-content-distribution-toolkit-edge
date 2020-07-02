package main

import (
	"context"
	"errors"
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

func DownloadFile(filepath, url string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	f, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer f.Close()
	fmt.Println("Downloading file from : " + url)
	fmt.Println("Downloading to::::", filepath)
	fileLen, err := strconv.Atoi(resp.Header.Get("Content-Length"))
	progressWriter := &ProgressWriter{}
	progressWriter.Total = int64(fileLen / 1024 / 1024)
	_, err = io.Copy(f, io.TeeReader(resp.Body, progressWriter))
	return err
}

func (s *relayCommandServer) Download(ctx context.Context, download_params *pb.DownloadParams) (*pb.Response, error) {

	log.Println(download_params.GetFolderpath())
	hierarchy := strings.Split(strings.Trim(download_params.GetFolderpath(), "/"), "/")
	log.Println(hierarchy)
	//
	fmt.Println("Printing buckets")
	fs.PrintBuckets()
	fmt.Println("Printing file sys")
	fs.PrintFileSystem()
	fmt.Println("====================")

	deadline := download_params.GetDeadline()
	addToExisting := download_params.GetAddtoexisting()
	metafilesLen := len(download_params.GetMetadatafiles())
	bulkfilesLen := len(download_params.GetBulkfiles())
	fileInfos := make([][]string, metafilesLen+bulkfilesLen+1)
	for i, x := range download_params.GetMetadatafiles() {
		log.Println("\t", (*x).Name)
		fileInfos[i] = make([]string, 5)
		fileInfos[i][0] = (*x).Name
		fileInfos[i][1] = (*x).Cdn
		fileInfos[i][2] = (*x).Hashsum
		fileInfos[i][3] = "metadata"
		fileInfos[i][4] = strconv.FormatInt(deadline, 10)
	}
	for i, x := range download_params.GetBulkfiles() {
		log.Println("\t", (*x).Name)
		fileInfos[metafilesLen+i] = make([]string, 5)
		fileInfos[metafilesLen+i][0] = (*x).Name
		fileInfos[metafilesLen+i][1] = (*x).Cdn
		fileInfos[metafilesLen+i][2] = (*x).Hashsum
		fileInfos[metafilesLen+i][3] = "bulkfile"
		fileInfos[metafilesLen+i][4] = strconv.FormatInt(deadline, 10)
	}
	fileInfos[metafilesLen+bulkfilesLen] = make([]string, 5)
	fileInfos[metafilesLen+bulkfilesLen][4] = strconv.FormatInt(deadline, 10)
	if addToExisting {
		if actualPath, _ := fs.GetActualPathForAbstractedPath(download_params.GetFolderpath()); actualPath != "" {
			err := DownloadFiles(actualPath, fileInfos)
			if err != nil {
				logger.Log("Error", "HandleMethodCalls", map[string]string{"Message": err.Error()})
				logger.Log("Telemetry", "AdditinalFileSyncInfo", map[string]string{"DownloadStatus": "FAIL", "FolderPath": download_params.GetFolderpath(), "Channel": "LAN"})
				log.Println("Error", fmt.Sprintf("%s", err))
				// delete the partial download request
				deleteFilesFromExistingFolder(actualPath, fileInfos)
				return &pb.Response{Responsemessage: "Additional files could not be downloaded"}, err
			}
			logger.Log("Telemetry", "AdditinalFileSyncInfo", map[string]string{"DownloadStatus": "SUCCESS", "FolderPath": download_params.GetFolderpath(), "Size": strconv.FormatInt(getDirSizeinMB(actualPath), 10) + " MB", "Channel": "LAN"})
			return &pb.Response{Responsemessage: "Additional Files downloaded"}, nil
		}
		logger.Log("Error", "HandleMethodCalls", map[string]string{"Message": fmt.Sprintf("Folder path does not already exist")})
		logger.Log("Telemetry", "AdditinalFileSyncInfo", map[string]string{"DownloadStatus": "FAIL", "FolderPath": download_params.GetFolderpath(), "Channel": "LAN"})
		return &pb.Response{Responsemessage: "Additional files could not be downloaded"}, errors.New("Folder does not exist")
	}
	err := fs.CreateDownloadNewFolder(hierarchy, DownloadFiles, fileInfos)
	if err != nil {
		logger.Log("Error", "HandleMethodCalls", map[string]string{"Message": err.Error()})
		logger.Log("Telemetry", "ContentSyncInfo", map[string]string{"DownloadStatus": "FAIL", "FolderPath": download_params.GetFolderpath(), "Channel": "LAN"})
		log.Println("Error", fmt.Sprintf("%s", err))
		return &pb.Response{Responsemessage: "Folder not downloaded"}, err
	}
	log.Println("")
	fs.PrintBuckets()
	fs.PrintFileSystem()
	log.Println("")
	// //???
	// if len(fileInfos) == 1 {
	// 	//logger.Log("Info", "Folder created. Download request does not have file infos ")
	// 	return &pb.Response{Responsemessage: "Folder created. No files to download"}, nil
	// }
	path, _ := fs.GetActualPathForAbstractedPath(download_params.GetFolderpath())
	logger.Log("Telemetry", "ContentSyncInfo", map[string]string{"DownloadStatus": "SUCCESS", "FolderPath": download_params.GetFolderpath(), "Size": strconv.FormatInt(getDirSizeinMB(path), 10) + " MB", "Channel": "LAN"})
	return &pb.Response{Responsemessage: "Folder downloaded"}, nil
}
func deleteFilesFromExistingFolder(path string, fileInfos [][]string) error {
	for i, x := range fileInfos {
		if i == len(fileInfos)-1 {
			break
		}
		var deletepath string
		switch x[3] {
		case "metadata":
			deletepath = filepath.Join(path, cfg.Section("DEVICE_INFO").Key("METADATA_FOLDER").String(), x[0])
		case "bulkfile":
			deletepath = filepath.Join(path, cfg.Section("DEVICE_INFO").Key("BULKFILE_FOLDER").String(), x[0])
		}
		// if path exist-- delete the file
		if _, err := os.Stat(deletepath); os.IsNotExist(err) {
			continue
		}
		err := os.Remove(deletepath)
		if err != nil {
			logger.Log("Error", "HandleMethodCalls", map[string]string{"Message": err.Error()})
			log.Println(err)
		}
	}
	return nil
}
func DownloadFiles(filePath string, fileInfos [][]string) error {
	for i, x := range fileInfos {
		if i == len(fileInfos)-1 {
			break
		}
		var downloadpath string
		switch x[3] {
		case "metadata":
			downloadpath = filepath.Join(filePath, cfg.Section("DEVICE_INFO").Key("METADATA_FOLDER").String(), x[0])
		case "bulkfile":
			downloadpath = filepath.Join(filePath, cfg.Section("DEVICE_INFO").Key("BULKFILE_FOLDER").String(), x[0])
		default:
			log.Println("Invalid File type: ", x[0])
			continue
		}
		if err := os.MkdirAll(filepath.Dir(downloadpath), 0700); err != nil {
			log.Println("Error", fmt.Sprintf("%s", err))
			return err
		}
		err := DownloadFile(downloadpath, x[1])
		if err != nil {
			log.Println("Error", fmt.Sprintf("%s", err))
			return err
		}
		err = matchSHA256(downloadpath, x[2])
		if err != nil {
			log.Println("Error", fmt.Sprintf("Hashsum did not match: %s", err.Error()))
			return errors.New(fmt.Sprintf("Hashsum did not match: %s", err.Error()))
		}
		//store it in a file
		if err := storeHashsum(downloadpath, x[2]); err != nil {
			log.Println("Error", fmt.Sprintf("Could not store Hashsum in the text file: %s", err))
			return errors.New(fmt.Sprintf("Could not store Hashsum in the text file: %s", err))
		}
	}
	// store the deadline for the created folder
	//handled if no files to be downloaded-- only folder created and deadline.txt
	if err := storeDeadline(filePath, fileInfos[0][4]); err != nil {
		fmt.Println("Error", fmt.Sprintf("Could not store validity end date: %s", err))
		return errors.New(fmt.Sprintf("Could not store validity end date: %s", err))
	}
	return nil
}

func storeHashsum(path, hash string) error {
	fileHashStr := filepath.Base(path) + "=>" + hash + "\n"
	pathdir := filepath.Dir(path)
	f, err := os.OpenFile(filepath.Join(pathdir, "hashsum.txt"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(fileHashStr)
	if err != nil {
		return err
	}
	return nil
}

func (s *relayCommandServer) Delete(ctx context.Context, delete_params *pb.DeleteParams) (*pb.Response, error) {
	folder_path := delete_params.GetFolderpath()
	log.Println(folder_path)
	path, _ := fs.GetActualPathForAbstractedPath(folder_path)
	if delete_params.GetRecursive() {
		err := fs.RecursiveDeleteFolder(strings.Split(folder_path, "/"))
		if err != nil {
			logger.Log("Error", "HandleMethodCalls", map[string]string{"Message": err.Error()})
			logger.Log("Telemetry", "ContentDeleteInfo", map[string]string{"DeleteStatus": "FAIL", "FolderPath": delete_params.GetFolderpath()})
			log.Println("Error", fmt.Sprintf("%s", err))
			return &pb.Response{Responsemessage: "Folder not deleted"}, err
		}
	} else {
		err := fs.DeleteFolder(strings.Split(folder_path, "/"))
		if err != nil {
			logger.Log("Error", "HandleMethodCalls", map[string]string{"Message": err.Error()})
			logger.Log("Telemetry", "ContentDeleteInfo", map[string]string{"DeleteStatus": "FAIL", "FolderPath": delete_params.GetFolderpath()})
			log.Println("Error", fmt.Sprintf("%s", err))
			return &pb.Response{Responsemessage: "Folder not deleted"}, err
		}
	}
	logger.Log("Telemetry", "ContentDeleteInfo", map[string]string{"DeleteStatus": "SUCCESS", "FolderPath": delete_params.GetFolderpath(), "Size": strconv.FormatInt(getDirSizeinMB(path), 10) + " MB"})
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
