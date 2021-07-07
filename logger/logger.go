// custom logger implementaion to handle telemetry as well
package logger

import (
	// "io"
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"syscall"
	"time"

	pbTelemetry "../goUpstreamTelemetry"
	"google.golang.org/grpc"
)

type EventType string

type Message struct {
	DeviceId string
	// MessageType string
	MessageSubType string
	TimeStamp      string
	MessageBody    map[string]string
}
type TelemetryMessage struct {
	DeviceIdInData                       string                 `json:"DeviceIdInData"`
	TimeStamp                            int64                  `json:"TimeStamp"`
	ContentSyncInfo   ContentSyncInfoMessage 	`json:"ContentSyncInfo,omitempty"`
	ContentDeleteInfo ContentDeleteInfoMessage  `json:"ContentDeleteInfo,omitempty"`
	IntegrityStats    IntegrityStatsMessage     `json:"IntegrityStats,omitempty"`
	HubStorage float64 `json:"HubStorage,omitempty"`
	Memory     float64 `json:"Memory,omitempty"`
	Liveness   string  `json:"Liveness,omitempty"`
	Error      string  `json:"Error,omitempty"`
	Critical   string  `json:"Critical,omitempty"`
	Warning    string  `json:"Warning,omitempty"`
	Info       string  `json:"Info,omitempty"`
	Debug      string  `json:"Debug,omitempty"`
}
type ContentSyncInfoMessage struct {
	DownloadStatus string
	AssetSize      float64
	Channel        string
	FolderPath     string
	AssetUpdate    string // for addtoexisting files
}
type ContentDeleteInfoMessage struct {
	DeleteStatus string
	AssetSize    float64
	Mode         string
	FolderPath   string
}
type IntegrityStatsMessage struct {
	IntegrityStatus string
	Filename        string
}

const (
	Liveness          EventType = "Liveness"
	Debug                       = "Debug"
	Info                        = "Info"
	Warning                     = "Warning"
	Error                       = "Error"
	Critical                    = "Critical"
	ContentSyncInfo             = "ContentSyncInfo"
	ContentDeleteInfo           = "ContentDeleteInfo"
	IntegrityStats              = "IntegrityStats"
	HubStorage                  = "HubStorage"
)
const (
	upstream_address = "HubEdgeProxyModule:5001"
	applicationName  = "Hub Module"
)

func (lt EventType) isValid() error {
	switch lt {
	case Liveness, Debug, Info, Warning, Error, Critical, ContentSyncInfo, ContentDeleteInfo, IntegrityStats, HubStorage:
		return nil
	}
	return fmt.Errorf("Invalid log type %v", string(lt))
}

type Logger struct {
	deviceId string
	file     *os.File
	writer   *bufio.Writer
}

func MakeLogger(deviceId string, logFilePath string, bufferSize int) *Logger {
	file, err := os.OpenFile(logFilePath, syscall.O_CREAT|syscall.O_WRONLY|syscall.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("[Logger]error in opening file: %s", logFilePath)
	}

	writer := bufio.NewWriterSize(file, bufferSize)
	l := Logger{deviceId, file, writer}

	return &l
}

func (l *Logger) Close() {
	l.writer.Flush()
	l.file.Sync()
	l.file.Close()
}

func (l *Logger) lockFile() error {
	err := syscall.Flock(int(l.file.Fd()), syscall.LOCK_EX)
	if err != nil {
		return err
	}

	return nil
}

func (l *Logger) unlockFile() error {
	err := syscall.Flock(int(l.file.Fd()), syscall.LOCK_UN)
	if err != nil {
		return err
	}

	return nil
}

func (l *Logger) Log(messageType string, messageSubType string, messageBody map[string]string) {
	if err := EventType(messageType).isValid(); err != nil {
		fmt.Errorf("[Logger]invalid message type %s", messageType)
	}

	msg := Message{l.deviceId, messageSubType, fmt.Sprintf("%d", time.Now().Unix()), messageBody}
	messsageDict := make(map[string]Message)
	messsageDict[messageType] = msg
	b, err := json.Marshal(messsageDict)
	if err != nil {
		log.Println(fmt.Sprintf("[Logger] error in creating message: %v", err))
	}
	msgString := string(b)
	//fmt.Println(msgString)
	// call grpc method to send the telemetry event
	if messageType == "Liveness" || messageType == "ContentSyncInfo" {
		l.constructTelemetryMessageAndSend(messageType, messageBody)
	}

	//
	err = l.lockFile()
	if err != nil {
		log.Println("[Logger] error in locking file")
	}

	l.writer.WriteString(msgString + "\n")
	if messageType == "Liveness" || messageType == "Telemetry" || messageType == "Critical" || messageType == "Error" {
		l.writer.Flush()
		l.file.Sync()
	}

	err = l.unlockFile()
	if err != nil {
		log.Println("[Logger] error in unlocking file")
	}
}

func grpcUpstream(message string) error {
	fmt.Println("Initiating connection")
	// Set up a connection to the server.
	conn, err := grpc.Dial(upstream_address, grpc.WithInsecure())
	if err != nil {
		log.Println("did not connect: %v", err)
		return err
	}
	defer conn.Close()
	log.Println("creating client....")
	client := pbTelemetry.NewTelemetryClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	//ctx := context.Background()
	fmt.Println("going to send telemetry")
	r, err := client.SendTelemetry(ctx, &pbTelemetry.TelemetryRequest{ApplicationName: applicationName, TelemetryData: message})
	if err != nil {
		log.Println("could not send telemetry to proxy: %v", err)
		return err
	}
	log.Printf("Response from proxy server: %s", r.GetMessage())
	return nil
}

func (l *Logger) constructTelemetryMessageAndSend(mtype string, messageBody map[string]string) {
	tm := new(TelemetryMessage)
	switch EventType(mtype) {
	case Liveness:

		tm.DeviceIdInData = l.deviceId
		tm.TimeStamp = fmt.Sprintf("%d", time.Now().Unix())
		tm.Liveness = messageBody["STATUS"]

	case ContentSyncInfo:
		tm.DeviceIdInData = l.deviceId
		tm.TimeStamp = fmt.Sprintf("%d", time.Now().Unix())
		tm.ContentSyncInfo.DownloadStatus = messageBody["DownloadStatus"]
		tm.ContentSyncInfo.FolderPath = messageBody["FolderPath"]
		tm.ContentSyncInfo.Channel = messageBody["Channel"]
		temp, _ := strconv.ParseFloat(messageBody["AssetSize"], 64)
		tm.ContentSyncInfo.AssetSize = temp
		tm.ContentSyncInfo.AssetUpdate = messageBody["AssetUpdate"]
	}
	b, err := json.Marshal(tm)
	if err != nil {
		log.Println(fmt.Sprintf("[LoggerGRPC] error in creating grpc message: %v", err))
	}
	msgString := string(b)
	fmt.Println("New Message UPSTREAM:")
	fmt.Println(msgString)
	err = grpcUpstream(msgString)
	if err != nil {
		log.Println("[LoggerGRPC] error in sending upstream: %v", err)
	}
}
