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

func main() {

	// Instantiate database connection to serve requests
	if !createDatabaseConnection() {
		fmt.Println("Could not create database connection, no point starting the server")
		return
	}
	//setupDbForTesting()
	// testCloudSyncServiceDownload()
	// start a concurrent background service which checks if the files on the device are tampered with
	go check()
	//go checkForVOD()
	go dummyTest()
	// this starts ZMQ dealer
	//fmt.Println("Setting up IPC")
	//go setupIPC()

	// set up the web server and routes
	router := gin.Default()
	fmt.Println("Setting up routes")
	setupRoutes(router)
	fmt.Println("Server starting ....")

	// start the web server at port 5000
	router.Run("0.0.0.0:5000")
}
