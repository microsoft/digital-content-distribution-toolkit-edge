package main

import (
	"fmt"
	"sync"

	"github.com/gin-gonic/gin"

	filesys "./filesys"
	cl "./logger"
)

// TODO: Implement remote update of code
// TODO: Implement update of part of a content
// TODO: Implement Acknowledgement of content update
// TODO: Have HUB ID (speak to Cloud which Vinod will write)
// TODO: Add unit tests to go functions
var logger cl.Logger
var fs *filesys.FileSystem

func main() {
	// if you change the port here, please make a correspoding change in device_sdk/downstream.py file
	var wg sync.WaitGroup

	upstream_grpc_port := 50051
	logger = cl.MakeLogger(upstream_grpc_port)

	downstream_grpc_port := 50052

	var err error
	fs, err = filesys.MakeFileSystem(4, "./")
	if err != nil {
		logger.Log("Error", fmt.Sprintf("%s", err))
	}
	defer fs.Close()

	// fmt.Println("Creating buckets...")
	fs.CreateBucket("Tree")
	fs.CreateBucket("FolderNameMapping")

	err = fs.CreateHome()
	if err != nil {
		logger.Log("Error", fmt.Sprintf("%s", err))
	}
	fs.PrintBuckets()
	fs.PrintFileSystem()

	logger.Log("Info", "All first level info being sent to iot-hub...")

	// Instantiate database connection to serve requests
	if !createDatabaseConnection() {
		//logger.Log("Critical", "Could not create database connection, no point starting the server")
		fmt.Println("Critical", "Could not create database connection, no point starting the server")
		return
	}
	//setupDbForTesting()
	// testCloudSyncServiceDownload()
	// start a concurrent background service which checks if the files on the device are tampered with
	wg.Add(1)
	go handle_method_calls(downstream_grpc_port, wg)
	go check()

	// set up the web server and routes
	router := gin.Default()
	fmt.Println("Setting up routes")
	setupRoutes(router)
	fmt.Println("Server starting ....")

	// start the web server at port 5000
	router.Run("0.0.0.0:5000")
	wg.Wait()
}
