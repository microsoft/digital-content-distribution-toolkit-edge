package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/gin-gonic/gin"
	"gopkg.in/ini.v1"

	filesys "./filesys"
	keymanager "./keymanager"
	cl "./logger"
)

// TODO: Implement remote update of code
// TODO: Implement update of part of a content
// TODO: Implement Acknowledgement of content update
// TODO: Have HUB ID (speak to Cloud which Vinod will write)
// TODO: Add unit tests to go functions
var logger cl.Logger
var fs *filesys.FileSystem
var km *keymanager.KeyManager

func main() {
	cfg, err := ini.Load("hub_config.ini")
	if err != nil {
		fmt.Printf("Failed to read config file: %v", err)
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
		logger.Log("Error", fmt.Sprintf("Failed to setup filesys: %s", err))
		os.Exit(1)
	}
	defer fs.Close()

	initflag, err := cfg.Section("DEVICE_INFO").Key("INIT_FILE_SYSTEM").Bool()
	if initflag {
		err = fs.InitFileSystem()
		if err != nil {
			logger.Log("Error", fmt.Sprintf("Failed to setup filesys: %s", err))
			os.Exit(1)
		}
	}

	// logger.Log("Info", "All first level info being sent to iot-hub...")
	fmt.Println("Info", "All first level info being sent to iot-hub...")
	// Instantiate database connection to serve requests
	if !createDatabaseConnection() {
		logger.Log("Critical", "Could not create database connection, no point starting the server")
		fmt.Println("Critical", "Could not create database connection, no point starting the server")
		return
	}

	//setupDbForTesting()
	// testCloudSyncServiceDownload()
	// start a concurrent background service which checks if the files on the device are tampered with
	wg.Add(1)
	go handle_method_calls(downstream_grpc_port, wg)
	integrityCheckInterval, err := cfg.Section("DEVICE_INFO").Key("INTEGRITY_CHECK_SCHEDULER").Duration()
	go checkIntegrity(integrityCheckInterval)
	satApiCmd := cfg.Section("DEVICE_INFO").Key("SAT_API_SWITCH").String()
	getdata_interval, err := cfg.Section("DEVICE_INFO").Key("MSTORE_SCHEDULER").Duration()
	switch satApiCmd {
	case "noovo":
		go pollNoovo(getdata_interval)
	case "mstore":
		go pollMstore(getdata_interval)
	}
	//go pollMstore()
	//testMstore()
	//go check()

	// setup key manager and load keys
	pubkeys_dir := cfg.Section("APP_AUTHENTICATION").Key("PUBLIC_KEY_STORE_PATH").String()
	keys_cache_size, err := cfg.Section("APP_AUTHENTICATION").Key("KEY_MANAGER_CACHE_SIZE").Int()

	km, _ = keymanager.MakeKeyManager(keys_cache_size)
	err = filepath.Walk(pubkeys_dir, func(path string, info os.FileInfo, err error) error {
		if filepath.Ext(path) == ".pem" {
			km.AddKey(path)
		}
		return nil
	})

	if err != nil {
		logger.Log("Error", fmt.Sprintf("Failed to setup keymanager: %s", err))
		os.Exit(1)
	}

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
