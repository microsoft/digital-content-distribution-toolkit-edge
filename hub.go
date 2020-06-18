package main

import (
	"fmt"
	"os"
	"sync"

	"github.com/gin-gonic/gin"
	"gopkg.in/ini.v1"

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
	cfg, err := ini.Load("hub_config.ini")
	if err != nil {
		fmt.Printf("Fail to read config file: %v", err)
		os.Exit(1)
	}

	var wg sync.WaitGroup

	upstream_grpc_port, err := cfg.Section("GRPC").Key("UPSTREAM_PORT").Int()
	logger = cl.MakeLogger(upstream_grpc_port)

	downstream_grpc_port, err := cfg.Section("GRPC").Key("DOWNSTREAM_PORT").Int()

	name_length, err := cfg.Section("FILE_SYSTEM").Key("NAME_LENGTH").Int()
	home_dir_location := cfg.Section("FILE_SYSTEM").Key("HOME_DIR_LOCATION").String()
	boltdb_location := cfg.Section("FILE_SYSTEM").Key("BOLTDB_LOCATION").String()
	fs, err = filesys.MakeFileSystem(name_length, home_dir_location, boltdb_location)
	if err != nil {
		logger.Log("Error", fmt.Sprintf("%s", err))
		os.Exit(1)
	}
	defer fs.Close()

	initflag, err := cfg.Section("DEVICE_INFO").Key("INIT_FILE_SYSTEM").Bool()
	if initflag {
		err = fs.InitFileSystem()
		if err != nil {
			logger.Log("Error", fmt.Sprintf("%s", err))
			os.Exit(1)
		}
	}

	logger.Log("Info", "All first level info being sent to iot-hub...")
	fmt.Println("Info", "All first level info being sent to iot-hub...")
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
	//go check()
	go checkForVOD()

	// set up the web server and routes
	router := gin.Default()
	fmt.Println("Setting up routes")
	setupRoutes(router)
	fmt.Println("Server starting ....")

	// start the gin web server
	gin_port, err := cfg.Section("GIN").Key("PORT").Int()
	router.Run(fmt.Sprintf("0.0.0.0:%d", gin_port))
	wg.Wait()
}
