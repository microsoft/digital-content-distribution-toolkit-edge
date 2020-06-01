package main

import (
	"fmt"

	"github.com/gin-gonic/gin"
)

// TODO: Implement remote update of code
// TODO: Implement update of part of a content
// TODO: Implement Acknowledgement of content update
// TODO: Have HUB ID (speak to Cloud which Vinod will write)
// TODO: Add unit tests to go functions
// var logger cl.Logger

func main() {
	// if you change the port here, please make a correspoding change in device_sdk/downstream.py file
	// grpc_port := 50051
	// logger = cl.MakeLogger(grpc_port)

	// logger.Log("Info", "All first level info being sent to iot-hub...")

	// Instantiate database connection to serve requests
	if !createDatabaseConnection() {
		//logger.Log("Critical", "Could not create database connection, no point starting the server")
		fmt.Println("Critical", "Could not create database connection, no point starting the server")
		return
	}
	//setupDbForTesting()
	// testCloudSyncServiceDownload()
	// start a concurrent background service which checks if the files on the device are tampered with
	//go check()
	go dummyTest()

	// set up the web server and routes
	router := gin.Default()
	fmt.Println("Setting up routes")
	setupRoutes(router)
	fmt.Println("Server starting ....")

	// start the web server at port 5000
	router.Run("0.0.0.0:5000")
}
