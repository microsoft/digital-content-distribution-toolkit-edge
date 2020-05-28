package main

import (
	"encoding/json"
	"fmt"
	"log"
	"reflect"
	"strings"
	"sync"
	"time"

	"gopkg.in/zeromq/goczmq.v4"
)

// TODO: Implement deletion

//Log this structure represents logs sent by hub to azure blob
type Log struct {
	Level   int
	Message string
}

var logsContainer []Log
var mutex *sync.Mutex = new(sync.Mutex)
var dealer *goczmq.Sock = nil
var downloaderChannel = make(chan []byte, 100) // Maximum 100 download command queued.

const downloadCommandPrefix string = "DOWNLOAD_CMD"
const deleteCommandPrefix string = "DELETE_CMD"
const commandDelimiteer string = ":"

func emptyLogs() {
	for _, storedLog := range logsContainer {
		encoded, err := json.Marshal(storedLog)
		if err != nil {
			fmt.Println("Error while converting to JSON")
		}
		dealer.SendFrame(encoded, goczmq.FlagNone)
	}
	logsContainer = logsContainer[:0]
}

func appendLog(log Log) {
	mutex.Lock()
	logsContainer = append(logsContainer, log)
	if len(logsContainer) == 100 {
		emptyLogs()
		fmt.Println("Done emptying logs")
	}
	mutex.Unlock()
}

func downloaderThread() {
	for true {
		fmt.Println("Waiting for downloader channel")
		downloadCommand := <-downloaderChannel
		fmt.Println("Issuing download command")
		err := downloadFolder(downloadCommand)
		if err != nil {
			println(err.Error())
		}
	}
}

func messageReceiver() {
	for true {
		fmt.Println("Waiting to receive message")
		message, err := dealer.RecvMessage()
		if err == nil {
			for _, msg := range message {
				stringMessage := string(msg)
				stringMessage = strings.Trim(stringMessage, " ")
				if strings.HasPrefix(stringMessage, downloadCommandPrefix) {
					parts := strings.Split(stringMessage, downloadCommandPrefix)
					if len(parts) > 1 {
						command := parts[1]
						// Enqueue this, the downloader function will dequeue and resume downloads
						downloaderChannel <- []byte(command)
						fmt.Println("Enqueued Download command")
					} else {
						fmt.Println("invalid Download CMD: ", stringMessage)
					}
				}
				if strings.HasPrefix(stringMessage, deleteCommandPrefix) {
					fmt.Println(strings.Split(stringMessage, ":")[1])
				}
			}
		} else {
			fmt.Println("Error is: ", err.Error())
		}
	}
}

func logsFlusher() {
	fmt.Println("Connecting to socket..")

	for true {
		dealer.SendFrame([]byte("ping"), goczmq.FlagNone)
		time.Sleep(10 * time.Second)
		encoded, err := json.Marshal(Log{Level: -10, Message: "This is a log"})
		if err != nil {
			fmt.Println("error:", err)
		}

		err = dealer.SendFrame(encoded, goczmq.FlagNone)
		if err != nil {
			// log.Fatal(err)
			fmt.Println("SENDING ERROR", err.Error())
		} else {
			fmt.Println("SENT MESSAGE TO ZMQ")
		}
	}
}

func setupIPC() {
	for true {
		test, err := goczmq.NewPair("tcp://127.0.0.1:5555")
		// create a dealer and start one go routine for sending log messages
		// start another go routine for receiving messages
		dealer = test
		// retry every 30 seconds to connect to Python ZMQ, this is very critical for the hub to receive messages from cloud
		if err != nil {
			log.Fatal(err)
			time.Sleep(30 * time.Second)
		}
		defer dealer.Destroy()
		fmt.Println("router created and bound")
		// for i := 0; i < 101; i++ {
		// 	appendLog(Log{Message: "This is log number " + strconv.Itoa(i), Level: i})
		// }
		fmt.Println(reflect.TypeOf(dealer))
		go logsFlusher()
		go messageReceiver()
		go downloaderThread()
		break
	}
	for true {
		time.Sleep(1000 * time.Second)
	}
}
