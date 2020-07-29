package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"sync"
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/gin-gonic/gin"
	ini "gopkg.in/ini.v1"

	filesys "./filesys"
	keymanager "./keymanager"
	cl "./logger"
)


type PublicKey struct {
	TimeStamp string  `json:"timestamp"`
	PublicKey string `json:"value"`
}


var logger *cl.Logger
var fs *filesys.FileSystem
var km *keymanager.KeyManager
var cfg *ini.File
var device_cfg *ini.File

func main() {
	var err, device_err error
	cfg, err = ini.Load("hub_config.ini")
	device_cfg, device_err := ini.Load(cfg.Section("HUB_AUTHENTICATION").Key("DEVICE_DETAIL_FILE").String())

	if err != nil {
		logger.Log("Critical", "hub_config.ini", map[string]string{"Message": fmt.Sprintf("Failed to read config file: %s", err)})
		fmt.Printf("Failed to read config file: %v", err)
		os.Exit(1)
	}
	if device_err != nil {
		logger.Log("Critical", "device_detail.ini", map[string]string{"Message": fmt.Sprintf("Failed to read config file: %s", device_err)})
		fmt.Printf("Failed to read config file: %v", device_err)
		os.Exit(1)
	}

	var wg sync.WaitGroup

	logFile := cfg.Section("LOGGER").Key("LOG_FILE_PATH").String()
	bufferSize, err := cfg.Section("LOGGER").Key("MEM_BUFFER_SIZE").Int()
	deviceId := device_cfg.Section("DEVICE_DETAIL").Key("deviceId").String()
	logger = cl.MakeLogger(deviceId, logFile, bufferSize)

	downstream_grpc_port, err := cfg.Section("GRPC").Key("DOWNSTREAM_PORT").Int()

	home_dir_location := cfg.Section("FILE_SYSTEM").Key("HOME_DIR_LOCATION").String()
	boltdb_location := cfg.Section("FILE_SYSTEM").Key("BOLTDB_LOCATION").String()
	fs, err = filesys.MakeFileSystem(home_dir_location, boltdb_location)
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
			logger.Log("Critical", "Filesys", map[string]string{"Message": fmt.Sprintf("Failed to setup filesys: %v", err)})
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
	liveness_interval, err := cfg.Section("DEVICE_INFO").Key("LIVENESS_SCHEDULER").Int()
	go liveness(liveness_interval)
	deletion_interval, err := cfg.Section("DEVICE_INFO").Key("DELETION_SCHEDULER").Int()
	go deleteContent(deletion_interval)
	
	// setup key manager and load keys
	storage_url := cfg.Section("APP_AUTHENTICATION").Key("BLOB_STORAGE_KEYS_GET_URL").String()
	pubkeys_dir := cfg.Section("APP_AUTHENTICATION").Key("PUBLIC_KEY_STORE_PATH").String()
	keys_cache_size, err := cfg.Section("APP_AUTHENTICATION").Key("KEY_MANAGER_CACHE_SIZE").Int()

	km, _ = keymanager.MakeKeyManager(keys_cache_size, pubkeys_dir)

	_, err = os.Stat(pubkeys_dir)
 
	if os.IsNotExist(err) {
		log.Println("Making public keys directory")
		errDir := os.MkdirAll(pubkeys_dir, 0755)
		if errDir != nil {
			logger.Log("Error", "Keymanager", map[string]string{"Message": fmt.Sprintf("Failed to make public keys directory: %v", err)})
			log.Println(fmt.Sprintf("Failed to make public keys directory: %v", err))
		}
 
	}

	// get public keys from blob storage
	resp, err := http.Get(storage_url)
	if(err != nil) {
		logger.Log("Error", "Keymanager", map[string]string{"Message": fmt.Sprintf("Failed to get keys from blob storage: %v", err)})
		log.Println(fmt.Sprintf("Failed to fetch blob storage url: %v", err))
	}
	defer resp.Body.Close()
	
	body, err := ioutil.ReadAll(resp.Body)
	if(err != nil) {
		logger.Log("Error", "Keymanager", map[string]string{"Message": fmt.Sprintf("Failed to decode blob storage keys json: %v", err)})
		log.Println(fmt.Sprintf("Failed to decode blob storage response: %v", err))
	}
	

	var keys []PublicKey
	err = json.Unmarshal(body, &keys)
	if(err != nil) {
		logger.Log("Error", "Keymanager", map[string]string{"Message": fmt.Sprintf("Failed to decode blob storage keys json: %v", err)})
		log.Println(fmt.Sprintf("Failed to decode blob storage response: %v", err))
	}
	
	log.Println(fmt.Sprintf("Got %v keys from blob storage", len(keys)))
	for _, key := range keys {
		filePath := filepath.Join(km.PubkeysDir, fmt.Sprintf("%v.pem", key.TimeStamp))
		f, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Println("[AddNewPublicKey] Error", fmt.Sprintf("%s", err))
		}
		defer f.Close()

		_, err = f.WriteString(key.PublicKey)
		if err != nil {
			log.Println("[AddNewPublicKey] Error", fmt.Sprintf("%s", err))
		}
	}


	file_times := make([]int64, 0)
	err = filepath.Walk(pubkeys_dir, func(path string, info os.FileInfo, err error) error {
		ext := filepath.Ext(path)
		if ext == ".pem" {
			file_full_name := filepath.Base(path)
			file_name := file_full_name[:len(file_full_name)-len(ext)]
			file_time, err := strconv.ParseInt(file_name, 10, 64)
			if err != nil {
				log.Println(fmt.Sprintf("Key file with name %v is not in correct format", file_full_name))
			} else {
				file_times = append(file_times, file_time)
			}
		}
		return nil
	})

	sort.Slice(file_times, func(i, j int) bool { return file_times[i] < file_times[j] })
	for _, file_time := range file_times {
		err = km.AddKey(fmt.Sprintf("%v.pem", file_time))
		if err != nil {
			log.Println(fmt.Sprintf("Failed to add key: %v", err))
		}
	}
	log.Println(fmt.Sprintf("Read a total of %v public keys", len(km.GetKeyList())))

	if len(km.GetKeyList()) == 0 {
		logger.Log("Critical", "Keymanager", map[string]string{"Message": fmt.Sprintf("Failed to setup keymanager: %v", err)})
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
