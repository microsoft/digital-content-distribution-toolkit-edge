package main

import (
	"fmt"

	"github.com/gin-gonic/gin"
)

func test() {

	// Call this the first time to setup testing
	// setupDbForTesting()
	// testCloudSyncServiceDownload()
	// navigate("root")
}

func main() {

	// Instantiate database connection to serve requests
	if !createDatabaseConnection() {
		fmt.Println("Could not create database connection, no point starting the server")
		return
	}
	// testCloudSyncServiceDownload()
	// start a concurrent background service which checks if the files on the device are tampered with
	go check()
	go checkForVOD()
	// set up the web server and routes
	router := gin.Default()
	fmt.Println("Setting up routes")
	setupRoutes(router)
	fmt.Println("Server starting ....")

	// start the web server at port 5000
	router.Run("0.0.0.0:5000")
}
