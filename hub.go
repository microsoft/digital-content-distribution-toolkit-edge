package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/gin-gonic/gin"
	ini "gopkg.in/ini.v1"

	filesys "./filesys"
	keymanager "./keymanager"
	cl "./logger"
)

// TODO: Implement remote update of code
// TODO: Implement update of part of a content
// TODO: Implement Acknowledgement of content update
// TODO: Have HUB ID (speak to Cloud which Vinod will write)
// TODO: Add unit tests to go functions
var logger *cl.Logger
var fs *filesys.FileSystem
var km *keymanager.KeyManager
var cfg *ini.File

func main() {
	cfg, err := ini.Load("hub_config.ini")
	if err != nil {
		fmt.Println("Failed to read config file: %v", err)
		os.Exit(1)
	}

	var wg sync.WaitGroup

	logFile := cfg.Section("LOGGER").Key("LOG_FILE_PATH").String()
	bufferSize, err := cfg.Section("LOGGER").Key("MEM_BUFFER_SIZE").Int()
	deviceId := cfg.Section("DEVICE_INFO").Key("DEVICE_NAME").String()
	logger = cl.MakeLogger(deviceId, logFile, bufferSize)

	downstream_grpc_port, err := cfg.Section("GRPC").Key("DOWNSTREAM_PORT").Int()

	name_length, err := cfg.Section("FILE_SYSTEM").Key("NAME_LENGTH").Int()
	home_dir_location := cfg.Section("FILE_SYSTEM").Key("HOME_DIR_LOCATION").String()
	boltdb_location := cfg.Section("FILE_SYSTEM").Key("BOLTDB_LOCATION").String()
	fs, err = filesys.MakeFileSystem(name_length, home_dir_location, boltdb_location)
	if err != nil {
		logger.Log("Error", "Filesys", map[string]string{"Message": fmt.Sprintf("Failed to setup filesys: %v", err)})
		log.Println(fmt.Sprintf("Failed to setup filesys: %v", err))
		os.Exit(1)
	}
	defer fs.Close()

	initflag, err := cfg.Section("DEVICE_INFO").Key("INIT_FILE_SYSTEM").Bool()
	if initflag {
		err = fs.InitFileSystem()
		if err != nil {
			logger.Log("Error", "Filesys", map[string]string{"Message": fmt.Sprintf("Failed to setup filesys: %v", err)})
			log.Println(fmt.Sprintf("Failed to setup filesys: %v", err))
			os.Exit(1)
		}
	}

	fmt.Println("Info", "All first level info being sent to iot-hub...")

	// launch a goroutine to handle method calls
	wg.Add(1)
	go handle_method_calls(downstream_grpc_port, wg)

	// start a concurrent background service which checks if the files on the device are tampered with
	integrityCheckInterval, err := cfg.Section("DEVICE_INFO").Key("INTEGRITY_CHECK_SCHEDULER").Int()
	go checkIntegrity(integrityCheckInterval)

	satApiCmd := cfg.Section("DEVICE_INFO").Key("SAT_API_SWITCH").String()
	getdata_interval, err := cfg.Section("DEVICE_INFO").Key("MSTORE_SCHEDULER").Int()
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
		logger.Log("Error", "Keymanager", map[string]string{"Message": fmt.Sprintf("Failed to setup keymanager: %v", err)})
		log.Println(fmt.Sprintf("Failed to setup keymanager: %v", err))
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
