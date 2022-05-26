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
	"syscall"
	"time"

	"github.com/google/uuid"

	pbTelemetry "binehub/goUpstreamTelemetry"

	"google.golang.org/grpc"
)

type EventType string

type Message struct {
	DeviceId string
	// MessageType string
	MessagesubType string
	TimeStamp      string
	MessageBody    map[string]string
}
type TelemetryMessage struct {
	DeviceIdInData                 string                `json:"deviceIdInData"`
	ApplicationName                string                `json:"applicationName"`
	ApplicationVersion             string                `json:"applicationVersion"`
	Timestamp                      int64                 `json:"timestamp"`
	AssetDeleteOnDeviceByScheduler *ContentsInfo         `json:"assetDeleteOnDeviceByScheduler,omitempty"`
	AssetDeleteOnDeviceByCommand   *ContentsInfo         `json:"assetDeleteOnDeviceByCommand,omitempty"`
	AssetDownloadOnDeviceFromSES   *ContentsInfo         `json:"assetDownloadOnDeviceFromSES,omitempty"`
	AssetDownloadOnDeviceByCommand *ContentsInfo         `json:"assetDownloadOnDeviceByCommand,omitempty"`
	TelemetryCommandMessage        *TelemetryCommandData `json:"telemetryCommandData,omitempty"`
	TotalNumberOfAssetsOnTheDevice int                   `json:"totalNumberOfAssetsOnTheDevice,omitempty"`
	HubStorageAvailable            float64               `json:"hubStorageAvailable,omitempty"`
	TotalStorage                   float64               `json:"totalStorage,omitempty"`
	Memory                         float64               `json:"memory,omitempty"`
	Liveness                       string                `json:"liveness,omitempty"`
	LivenessNumeric                int                   `json:"livenessNumeric,omitempty"`
	InvalidCommandOnDevice         string                `json:"invalidCommandOnDevice,omitempty"`
	Error                          string                `json:"error,omitempty"`
	Critical                       string                `json:"critical,omitempty"`
	Warning                        string                `json:"warning,omitempty"`
	Info                           string                `json:"info,omitempty"`
	Debug                          string                `json:"debug,omitempty"`
	EventType                      string                `json:"eventType"`
	EventId                        string                `json:"eventId"`
}
type ContentsInfo struct {
	NumberOfContents  int    `json:"numberOfContents,omitempty"`
	ContentProperties string `json:"contentProperties,omitempty"`
}

type MessageSubType struct {
	StringMessage        string
	FloatValue           float64
	Integer              int
	ContentsInfo         ContentsInfo
	TelemetryCommandData TelemetryCommandData
}

type TelemetryCommandName int

const (
	ContentDownloaded TelemetryCommandName = iota
	ContentDeleted
	CompleteCommand
	ProvisionDevice
)

type TelemetryCommandData struct {
	CommandName TelemetryCommandName `json:"commandName"`
	CommandData string               `json:"commandData,omitempty"`
	IsAssetMap  bool                 `json:"isAssetMap,omitempty"`
}

const (
	Liveness                       = "Liveness"
	LivenessNumeric                = "LivenessNumeric"
	Debug                          = "Debug"
	Info                           = "Info"
	Warning                        = "Warning"
	Error                          = "Error"
	Critical                       = "Critical"
	AssetDeleteOnDeviceByScheduler = "AssetDeleteOnDeviceByScheduler"
	AssetDeleteOnDeviceByCommand   = "AssetDeleteOnDeviceByCommand"
	AssetDownloadOnDeviceFromSES   = "AssetDownloadOnDeviceFromSES"
	AssetDownloadOnDeviceByCommand = "AssetDownloadOnDeviceByCommand"
	TotalNumberOfAssetsOnTheDevice = "TotalNumberOfAssetsOnTheDevice"
	InvalidCommandOnDevice         = "InvalidCommandOnDevice"
	HubStorageAvailable            = "HubStorageAvailable"
	TotalStorage                   = "TotalStorage"
	Memory                         = "Memory"
	TelemetryCommandMessage        = "TelemetryCommand"
)

func (lt EventType) isValid() error {
	switch lt {
	case Liveness, LivenessNumeric, Debug, Info, Warning, Error, Critical, AssetDeleteOnDeviceByScheduler, AssetDownloadOnDeviceFromSES, TotalNumberOfAssetsOnTheDevice, Memory, HubStorageAvailable, TotalStorage, TelemetryCommandMessage:
		return nil
	}
	return fmt.Errorf("Invalid log type %v", string(lt))
}

type Logger struct {
	deviceId           string
	file               *os.File
	writer             *bufio.Writer
	applicationName    string
	applicationVersion string
	upstreamAddress    string
}

func MakeLogger(deviceId string, logFilePath string, bufferSize int, applicationName string, applicationVersion string, upstreamAddress string) *Logger {
	file, err := os.OpenFile(logFilePath, syscall.O_CREAT|syscall.O_WRONLY|syscall.O_APPEND, 0666)
	if err != nil {
		//log.Printf("ERROR:", err.Error())
		log.Fatalf("[Logger]error in opening file:", logFilePath)
	}

	writer := bufio.NewWriterSize(file, bufferSize)
	l := Logger{deviceId, file, writer, applicationName, applicationVersion, upstreamAddress}

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

/********************* To be removed **************************/
func (l *Logger) Log_old(messageType string, messagesubType string, messageBody map[string]string) {
	if err := EventType(messageType).isValid(); err != nil {
		fmt.Errorf("[Logger]invalid message type %s", messageType)
	}

	msg := Message{l.deviceId, messagesubType, fmt.Sprintf("%d", time.Now().Unix()), messageBody}
	messsageDict := make(map[string]Message)
	messsageDict[messageType] = msg
	b, err := json.Marshal(messsageDict)
	if err != nil {
		log.Println(fmt.Sprintf("[Logger] error in creating message: %v", err))
	}
	msgString := string(b)
	//fmt.Println("msg in old logs::", msgString)
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

func (l *Logger) sendGrpcUpstream(message string) error {
	fmt.Println("Initiating connection")
	// Set up a connection to the server.
	conn, err := grpc.Dial(l.upstreamAddress, grpc.WithInsecure())
	if err != nil {
		log.Printf("did not connect: %v", err)
		return err
	}
	defer conn.Close()
	fmt.Println("creating client....")
	client := pbTelemetry.NewTelemetryClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	//ctx := context.Background()
	fmt.Println("going to send telemetry")
	r, err := client.SendTelemetry(ctx, &pbTelemetry.TelemetryRequest{ApplicationName: l.applicationName, TelemetryData: message})
	if err != nil {
		log.Println(message)
		log.Printf("could not send telemetry to proxy: %v", err)
		return err
	}
	log.Printf("Response from proxy server: %s", r.GetMessage())
	return nil
}

func (l *Logger) Log(eventType string, subType *MessageSubType) error {
	if err := EventType(eventType).isValid(); err != nil {
		fmt.Errorf("[Logger]invalid message type %s", eventType)
		return fmt.Errorf(fmt.Sprintf("[Logger]invalid message type %s", eventType))
	}

	tm := new(TelemetryMessage)
	tm.ApplicationName = l.applicationName
	tm.ApplicationVersion = l.applicationVersion
	tm.DeviceIdInData = l.deviceId
	tm.Timestamp = time.Now().Unix()
	tm.EventType = eventType
	tm.EventId = uuid.NewString()

	switch EventType(eventType) {
	case Liveness:
		tm.Liveness = subType.StringMessage
	case LivenessNumeric:
		tm.LivenessNumeric = subType.Integer
	case InvalidCommandOnDevice:
		tm.InvalidCommandOnDevice = subType.StringMessage
	case HubStorageAvailable:
		tm.HubStorageAvailable = subType.FloatValue
	case TotalStorage:
		tm.TotalStorage = subType.FloatValue
	case Memory:
		tm.Memory = subType.FloatValue
	case AssetDeleteOnDeviceByScheduler:
		tm.AssetDeleteOnDeviceByScheduler = &subType.ContentsInfo
	case AssetDeleteOnDeviceByCommand:
		tm.AssetDeleteOnDeviceByCommand = &subType.ContentsInfo
	case AssetDownloadOnDeviceFromSES:
		tm.AssetDownloadOnDeviceFromSES = &subType.ContentsInfo
	case AssetDownloadOnDeviceByCommand:
		tm.AssetDownloadOnDeviceByCommand = &subType.ContentsInfo
	case TotalNumberOfAssetsOnTheDevice:
		tm.TotalNumberOfAssetsOnTheDevice = subType.Integer
	case TelemetryCommandMessage:
		tm.TelemetryCommandMessage = &subType.TelemetryCommandData
	case Error:
		tm.Error = subType.StringMessage
	case Critical:
		tm.Critical = subType.StringMessage
	case Info:
		tm.Info = subType.StringMessage
	case Debug:
		tm.Debug = subType.StringMessage
	case Warning:
		tm.Warning = subType.StringMessage
	}
	byteMessage, err := json.Marshal(tm)
	fmt.Println(err)
	if err != nil {
		log.Println(fmt.Sprintf("[LoggerGRPC] error in creating grpc message: %v", err))
		return fmt.Errorf(fmt.Sprintf("[LoggerGRPC] error in creating grpc message: %v", err))
	}
	msgString := string(byteMessage)
	fmt.Println(fmt.Sprintf("New Message UPSTREAM: %s", msgString))
	log.Println(fmt.Sprintf("New Message UPSTREAM: %s", msgString))
	err = l.sendGrpcUpstream(msgString)
	if err != nil {
		log.Printf("[LoggerGRPC] error in sending upstream: %v", err)
		return fmt.Errorf(fmt.Sprintf("[LoggerGRPC] error in sending upstream: %v", err))
	}

	return nil
}
