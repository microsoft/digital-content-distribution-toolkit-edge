package main

import (
	pb "binehub/DownstreamCommands"
	l "binehub/logger"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"google.golang.org/grpc"
)

type server struct {
	pb.UnimplementedCommandServer
}
type UpdateFilterPayload struct {
	DeviceID  string   `json:"deviceId"`
	CommandID string   `json:"commandId"`
	Filters   []string `json:"filters"`
}

func newCommandServer() *server {
	s := &server{}
	return s
}

func (s *server) ReceiveCommand(ctx context.Context, commandParams *pb.CommandServiceRequest) (*pb.CommandServiceResponse, error) {
	command := commandParams.GetCommandName()
	payload := commandParams.GetPayload()
	fmt.Println("Received from client- Command: ", command)
	fmt.Println("Received from client- Payload:", payload)
	switch command {
	case "Download":
		go handleDownload(payload)
	case "Delete":
		go handleDelete(payload)
	case "FilterUpdate":
		go handleFilterUpdate(payload)
	default:
		log.Printf("Command received: %s. Not supported by the hub device\n", command)
		//send telemetry
		sm := l.MessageSubType{StringMessage: "Invalid command name received on the device"}
		logger.Log("InvalidCommandOnDevice", &sm)
	}
	fmt.Println("Returning back the response to the proxy.....")
	return &pb.CommandServiceResponse{Code: 1, Message: "Recieved payload for " + command}, nil
}

func handleFilterUpdate(payload string) {
	//check if valid payload
	//setfilters
	//if success- write in the ini file(persistence)
	//put the entry in the retry bucket in both the cases(success/fail)
	var filterPayload UpdateFilterPayload
	jsonerr := json.Unmarshal([]byte(payload), &filterPayload)
	if jsonerr != nil {
		log.Println("[FilterUpdate] Error: ", fmt.Sprintf("%s", jsonerr))
		sm := l.MessageSubType{StringMessage: "FilterUpdate: " + jsonerr.Error()}
		logger.Log("Error", &sm)
		return
	}
	deviceId := device_cfg.Section("DEVICE_DETAIL").Key("deviceId").String()
	deviceIdInPayload := filterPayload.DeviceID
	commandIdInPayload := filterPayload.CommandID
	filtersInPayload := filterPayload.Filters
	if deviceIdInPayload != deviceId || commandIdInPayload == "" || filtersInPayload == nil {
		log.Printf("Payload received: %s. Invalid params received in the command payload\n", payload)
		// send invalid payload telemetry
		sm := l.MessageSubType{StringMessage: "FilterUpdate: Invalid payload received on the device"}
		logger.Log("InvalidCommandOnDevice", &sm)
		return
	}
	serviceId := cfg.Section("MSTORE_SERVICE").Key("serviceId").String()
	//var setkeywords = false
	var failedReason string
	var keywords string
	for _, filter := range filtersInPayload {
		keywords += "/" + filter
	}
	err := callSetkeywords(serviceId, keywords)
	if err == nil {
		//persistence of the filters
		cfg.Section("DEVICE_INFO").Key("FILTERS").SetValue(keywords)
		writeErr := cfg.SaveTo("hub_config.ini")
		if writeErr == nil {
			//setkeywords = true
			fmt.Println("Filters set successfully::", cfg.Section("DEVICE_INFO").Key("FILTERS").String())
		} else {
			failedReason = err.Error()
			fmt.Println(failedReason)
		}
	} else {
		failedReason = err.Error()
		fmt.Println(failedReason)
	}
	// TODO: insert entry into the bucket

}
func callSetkeywords(serviceId, keywords string) error {
	setFilterCall := "http://host.docker.internal:8134/setkeywords/" + serviceId + keywords
	//setFilterCall := "http://localhost:8134/setkeywords/" + serviceId + keywords
	fmt.Println(setFilterCall)
	res, err := http.Get(setFilterCall)
	if err != nil {
		log.Println("[FilterUpdate] Error: ", fmt.Sprintf("%s", err))
		sm := l.MessageSubType{StringMessage: "FilterUpdate: " + err.Error()}
		logger.Log("Error", &sm)
		return err
	}
	defer res.Body.Close()
	response, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Println("[FilterUpdate] Error: ", fmt.Sprintf("%s", err))
		sm := l.MessageSubType{StringMessage: "FilterUpdate: " + err.Error()}
		logger.Log("Error", &sm)
		return err
	}
	str := string(response)
	r := regexp.MustCompile(`(?s)<body>(.*)</body>`)
	result := r.FindString(str)
	status := strings.Fields(strings.Trim(result, "\n"))
	if status[2] == "FAILED" {
		err := errors.New(fmt.Sprintf("setkeywords API of mstore service failed. Response: %s", status))
		log.Println("[FilterUpdate] Error: ", fmt.Sprintf("%s", err))
		sm := l.MessageSubType{StringMessage: "FilterUpdate: " + err.Error()}
		logger.Log("Error", &sm)
		return err
	}
	return nil
}
func handleDelete(payload string) {
	time.Sleep(60 * time.Second)
	fmt.Println("in another thread after sleep")
}

func handleDownload(payload string) {

}
func handleCommands(port int, wg sync.WaitGroup) {
	lis, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	fmt.Printf("[Command Server] Listening for commands on port %d .......\n", port)
	defer wg.Done()
	grpcServer := grpc.NewServer()
	pb.RegisterCommandServer(grpcServer, newCommandServer())
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
