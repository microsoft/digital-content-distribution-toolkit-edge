// Copyright (c) Microsoft Corporation.
// Licensed under the MIT license.

package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/natefinch/lumberjack"
	ini "gopkg.in/ini.v1"

	cpi "binehub/cloudapihandler"
	filesys "binehub/filesys"
	keymanager "binehub/keymanager"
	cl "binehub/logger"
)

/****************************** All the "fmt" statements are written on the console.
******************************* All the "log" statements are written to the log files.
*********************/

type PublicKey struct {
	TimeStamp string `json:"timestamp"`
	PublicKey string `json:"value"`
}

var logger *cl.Logger
var fs *filesys.FileSystem
var km *keymanager.KeyManager
var cfg *ini.File
var device_cfg *ini.File

func main() {
	var err, device_err error
	fmt.Println("Starting ---------")
	cfg, err = ini.Load("hub_config.ini")
	//cfg, err = ini.Load("hub_config_tmp.ini")
	fmt.Println("loaded hub_config ini file")
	device_ini := cfg.Section("HUB_AUTHENTICATION").Key("DEVICE_DETAIL_FILE").String()
	device_cfg, device_err = ini.Load(device_ini)
	codeLogsFile := cfg.Section("LOGGER").Key("CODE_LOGS_FILE_PATH").String()
	codeLogsFileMaxSize, err := cfg.Section("LOGGER").Key("CODE_LOGS_FILE_MAX_SIZE").Int()
	codeLogsFileMaxBackups, err := cfg.Section("LOGGER").Key("CODE_LOGS_FILE_MAX_BACKUPS").Int()
	codeLogsFileMaxAge, err := cfg.Section("LOGGER").Key("CODE_LOGS_FILE_MAX_AGE").Int()

	log.SetOutput(&lumberjack.Logger{
		Filename:   codeLogsFile,
		MaxSize:    codeLogsFileMaxSize,    // megabytes after which new file is created
		MaxBackups: codeLogsFileMaxBackups, // number of backups
		MaxAge:     codeLogsFileMaxAge,     //days
	})
	fmt.Printf("Logs written to %v\n: ", codeLogsFile)
	if err != nil {
		fmt.Printf("Failed to read config file: %v", err)
		os.Exit(1)
	}
	if device_err != nil {
		fmt.Printf("Failed to read config file: %v", device_err)
		os.Exit(1)
	}

	var wg sync.WaitGroup

	logFile := cfg.Section("LOGGER").Key("TEMP_LOG_FILE_PATH").String()
	bufferSize, err := cfg.Section("LOGGER").Key("MEM_BUFFER_SIZE").Int()
	deviceId := device_cfg.Section("DEVICE_DETAIL").Key("deviceId").String()
	serviceId := cfg.Section("MSTORE_SERVICE").Key("serviceId").String()
	applicationName := cfg.Section("APP_INFO").Key("APPLICATION_NAME").String()
	applicationVersion := cfg.Section("APP_INFO").Key("APPLICATION_VERSION").String()
	upstreamAddress := cfg.Section("GRPC").Key("UPSTREAM_ADDRESS").String()
	logger = cl.MakeLogger(deviceId, logFile, bufferSize, applicationName, applicationVersion, upstreamAddress)

	downstream_grpc_port, err := cfg.Section("GRPC").Key("DOWNSTREAM_PORT").Int()

	home_dir_location := cfg.Section("FILE_SYSTEM").Key("HOME_DIR_LOCATION").String()
	boltdb_location := cfg.Section("FILE_SYSTEM").Key("BOLTDB_LOCATION").String()
	fs, err = filesys.MakeFileSystem(home_dir_location, boltdb_location)
	if err != nil {
		logger.Log_old("Error", "Filesys", map[string]string{"Message": fmt.Sprintf("Failed to setup filesys: %v", err)})
		fmt.Println("Failed to setup filesys: %v", err)
		log.Println(fmt.Sprintf("Failed to setup filesys: %v", err))
		os.Exit(1)
	}
	defer fs.Close()
	initflag, err := cfg.Section("DEVICE_INFO").Key("INIT_FILE_SYSTEM").Bool()
	if initflag {
		err = fs.InitFileSystem()
		if err != nil {
			logger.Log_old("Critical", "Filesys", map[string]string{"Message": fmt.Sprintf("Failed to setup filesys: %v", err)})
			fmt.Println("Failed to setup filesys: %v", err)
			log.Println(fmt.Sprintf("Failed to setup filesys: %v", err))
			os.Exit(1)
		}
	}
	filters := device_cfg.Section("DEVICE_DETAIL").Key("FILTERS").String()
	if filters == "" {
		device_cfg.Section("DEVICE_DETAIL").Key("FILTERS").SetValue("Invalid")
		writeErr := device_cfg.SaveTo(device_ini)
		if writeErr != nil {
			fmt.Printf("Error in saving filters to invalid to the ini file: %v\n", err)
			log.Println(fmt.Sprintf("Error in saving filters to invalid to the ini file: %v\n", err))

		}
		filters = "Invalid"
	}
	fmt.Println("Setting up the filters on the startup of the hub module:", filters)
	filters = "/" + strings.Trim(filters, "/")
	if err = callSetkeywords(serviceId, filters); err != nil {
		fmt.Printf("Could not set persisted filters:%v\n", err)
		log.Println(fmt.Sprintf("Could not set persisted filters:%v\n", err))
	} else {
		fmt.Println("filters set successfully on the device!")
	}

	fmt.Println("Info", "All first level info being sent to iot-hub...")
	fs.PrintBuckets()
	// launch a goroutine to handle method calls
	wg.Add(6)
	go handleCommands(downstream_grpc_port, wg)

	// start a concurrent background service which checks if the files on the device are tampered with
	//integrityCheckInterval, err := cfg.Section("DEVICE_INFO").Key("INTEGRITY_CHECK_SCHEDULER").Int()
	//go checkIntegrity(integrityCheckInterval)

	getdata_interval, err := cfg.Section("DEVICE_INFO").Key("MSTORE_SCHEDULER").Int()
	deviceStorage := cfg.Section("DEVICE_INFO").Key("MSTORE_DEVICE_STORAGE").String()
	containerStorage := cfg.Section("DEVICE_INFO").Key("MSTORE_CONTAINER_STORAGE").String()
	go pollMstore(getdata_interval, containerStorage, deviceStorage)

	telemetry_interval, err := cfg.Section("DEVICE_INFO").Key("LIVENESS_SCHEDULER").Int()
	mountedVolpath := cfg.Section("DEVICE_INFO").Key("MSTORE_CONTAINER_STORAGE").String()
	//go liveness(telemetry_interval)
	go telemetryScheduler(telemetry_interval, mountedVolpath)
	//deletion_interval, err := cfg.Section("DEVICE_INFO").Key("DELETION_SCHEDULER").Int()
	//go deleteContent(deletion_interval)
	//go mock_telelmetry(180)
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
			logger.Log_old("Error", "Keymanager", map[string]string{"Message": fmt.Sprintf("Failed to make public keys directory: %v", err)})
			log.Println(fmt.Sprintf("Failed to make public keys directory: %v", err))
		}

	}

	// get public keys from blob storage
	resp, err := http.Get(storage_url)
	if err != nil {
		logger.Log_old("Error", "Keymanager", map[string]string{"Message": fmt.Sprintf("Failed to get keys from blob storage: %v", err)})
		log.Println(fmt.Sprintf("Failed to fetch blob storage url: %v", err))
	}
	// This condition added to bypass the error thrown if the storage url did not exist.
	if resp != nil {
		defer resp.Body.Close()

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			logger.Log_old("Error", "Keymanager", map[string]string{"Message": fmt.Sprintf("Failed to decode blob storage keys json: %v", err)})
			log.Println(fmt.Sprintf("Failed to decode blob storage response: %v", err))
		}

		log.Println(fmt.Sprintf("got body %v from blob storage", string(body)))

		var keys []PublicKey
		err = json.Unmarshal(body, &keys)
		if err != nil {
			logger.Log_old("Error", "Keymanager", map[string]string{"Message": fmt.Sprintf("Failed to decode blob storage keys json: %v", err)})
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
			logger.Log_old("Critical", "Keymanager", map[string]string{"Message": fmt.Sprintf("Failed to setup keymanager: %v", err)})
			log.Println(fmt.Sprintf("Failed to setup keymanager: %v", err))
			//os.Exit(1)
		}
	}

	var provisionData cpi.ApiData
	provisionData.Id = deviceId
	provisionData.ApiType = cpi.Provisioned
	provisionData.OperationTime = time.Now().UTC()
	provisionData.RetryCount = 0

	provisionDataByteArr, err := json.Marshal(provisionData)

	if err != nil {
		fmt.Printf("Failed to get byte array of provisioned data: %v", err)
		log.Println(fmt.Sprintf("Failed to get byte array of provisioned data: %v", err))
	} else {
		fmt.Printf("Setting provisioned status  %v", provisionData)
		log.Println(fmt.Sprintf("Setting provisioned status  %v", provisionData))
		fs.AddProvisionedStatus(deviceId, provisionDataByteArr)
	}

	handle_interval, err := cfg.Section("BLENDNET").Key("HANDLE_INTERVAL").Int()

	if err != nil {
		fmt.Printf("unable to get handle interval from ini file: %v", err)
		log.Println(fmt.Sprintf("unable to get handle interval from ini file: %v", err))
		handle_interval = 10
	}

	cpi.InitAPIHandler(*fs, deviceId, logger)

	downloadBatchMessageSize, err := cfg.Section("BLENDNET").Key("DOWNLOAD_TELEMETRY_MESSAGE_SIZE").Int()

	if err != nil {
		fmt.Printf("unable to get download batch message size from ini file: %v", err)
		downloadBatchMessageSize = 10
	}
	deleteBatchMessageSize, err := cfg.Section("BLENDNET").Key("DELETE_TELEMETRY_MESSAGE_SIZE").Int()

	if err != nil {
		fmt.Printf("unable to get delete batch message size from ini file: %v", err)
		deleteBatchMessageSize = 10
	}

	go cpi.HandleApiRequests(handle_interval, downloadBatchMessageSize, deleteBatchMessageSize)

	cpi.InitAssetMapCall(device_ini, device_cfg)
	messageSize, err := cfg.Section("BLENDNET").Key("ASSETMAP_MESSAGE_SIZE").Int()

	if err != nil {
		fmt.Printf("unable to get assetmap message size from ini file: %v", err)
		log.Println(fmt.Sprintf("unable to get assetmap message size from ini file: %v", err))
		messageSize = 25
	}
	// this interval is for the goroutine that sends the assetmap. Every hour it checks whether elapsed time is greater than 24 hour. If it is then it sends the Assetmap telemetry
	sleepInterval, err := cfg.Section("BLENDNET").Key("ASSETMAP_SLEEP_INTERVAL").Int()

	if err != nil {
		fmt.Printf("unable to get assetmap message size from ini file: %v", err)
		log.Println(fmt.Sprintf("unable to get assetmap message size from ini file: %v", err))
		sleepInterval = 60
	}

	go cpi.HandleAssetMapRequest(sleepInterval, messageSize)
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
