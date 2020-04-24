package main

import (
	"fmt"

	"github.com/gin-gonic/gin"
)

func main() {
	// call this first time before to setup database locally
	setupDbForTesting()

	// Instantiate database connection to serve requests
	if !createDatabaseConnection() {
		fmt.Println("Could not create database connection, no point starting the server")
		return
	}

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
