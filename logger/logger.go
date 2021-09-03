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
	DeviceIdInData                       string                 `json:"DeviceIdInData"`
	ApplicationName                      string                 `json:"ApplicationName"`
	ApplicationVersion                   string                 `json:"ApplicationVersion"`
	Timestamp                            int64                  `json:"TimeStamp"`
	AssetDeleteOnDeviceByScheduler       *AssetInfo             `json:"AssetDeleteOnDeviceByScheduler,omitempty"`
	AssetDeleteOnDeviceByCommand         *AssetInfo             `json:"AssetDeleteOnDeviceByCommand,omitempty"`
	FailedAssetDeleteOnDeviceByScheduler *AssetInfo             `json:"FailedAssetDeleteOnDeviceByScheduler,omitempty"`
	FailedAssetDeleteOnDeviceByCommand   *AssetInfo             `json:"FailedAssetDeleteOnDeviceByCommand,omitempty"`
	AssetDownloadOnDeviceFromSES         *AssetInfo             `json:"AssetDownloadOnDeviceFromSES,omitempty"`
	AssetDownloadOnDeviceByCommand       *AssetInfo             `json:"AssetDownloadOnDeviceByCommand,omitempty"`
	FailedAssetDownloadOnMobile          *AssetInfo             `json:"FailedAssetDownloadOnMobile,omitempty"`
	SucessfulAssetDownloadOnMobile       *AssetInfo             `json:"SucessfulAssetDownloadOnMobile,omitempty"`
	CorruptedFileStatsFromScheduler      *IntegrityStatsMessage `json:"CorruptedFileStatsFromScheduler,omitempty"`

	HubStorage             float64 `json:"HubStorage,omitempty"`
	Memory                 float64 `json:"Memory,omitempty"`
	Liveness               string  `json:"Liveness,omitempty"`
	InvalidCommandOnDevice string  `json:"InvalidCommandOnDevice,omitempty"`
	Error                  string  `json:"Error,omitempty"`
	Critical               string  `json:"Critical,omitempty"`
	Warning                string  `json:"Warning,omitempty"`
	Info                   string  `json:"Info,omitempty"`
	Debug                  string  `json:"Debug,omitempty"`
}
type AssetInfo struct {
	Size             float64 `json:"Size,omitempty"`
	AssetId          string  `json:"AssetId,omitempty"`
	RelativeLocation string  `json:"RelativeLocation,omitempty"`
	IsUpdate         bool    `json:"IsUpdate,omitempty"`
	StartTime        int64   `json:"StartTime,omitempty"`
	EndTime          int64   `json:"EndTime,omitempty"`
	Duration         int     `json:"Duration,omitempty"`
}
type IntegrityStatsMessage struct {
	AssetId          string `json:"AssetId,omitempty"`
	RelativeLocation string `json:"RelativeLocation,omitempty"`
	Filename         string `json:"Filename,omitempty"`
	ActualSHA        string `json:"ActualSHA,omitempty"`
	ExpectedSHA      string `json:"ExpectedSHA,omitempty"`
}
type MessageSubType struct {
	StringMessage  string
	FloatValue     float64
	AssetInfo      AssetInfo
	IntegrityStats IntegrityStatsMessage
}

const (
	Liveness                             EventType = "Liveness"
	Debug                                          = "Debug"
	Info                                           = "Info"
	Warning                                        = "Warning"
	Error                                          = "Error"
	Critical                                       = "Critical"
	AssetDeleteOnDeviceByScheduler                 = "AssetDeleteOnDeviceByScheduler"
	AssetDeleteOnDeviceByCommand                   = "AssetDeleteOnDeviceByCommand"
	FailedAssetDeleteOnDeviceByScheduler           = "FailedAssetDeleteOnDeviceByScheduler"
	FailedAssetDeleteOnDeviceByCommand             = "FailedAssetDeleteOnDeviceByCommand"
	AssetDownloadOnDeviceFromSES                   = "AssetDownloadOnDeviceFromSES"
	AssetDownloadOnDeviceByCommand                 = "AssetDownloadOnDeviceByCommand"
	FailedAssetDownloadOnMobile                    = "FailedAssetDownloadOnMobile"
	SucessfulAssetDownloadOnMobile                 = "SucessfulAssetDownloadOnMobile"
	CorruptedFileStatsFromScheduler                = "CorruptedFileStatsFromScheduler"
	InvalidCommandOnDevice                         = "InvalidCommandOnDevice"
	HubStorage                                     = "HubStorage"
	Memory                                         = "Memory"
)

// const (
// 	upstream_address   = "HubEdgeProxyModule:5001"
// 	applicationName    = "Hub Module"
// 	applicationVersion = "v1.0"
// )

func (lt EventType) isValid() error {
	switch lt {
	case Liveness, Debug, Info, Warning, Error, Critical, AssetDeleteOnDeviceByScheduler, AssetDeleteOnDeviceByCommand, FailedAssetDeleteOnDeviceByScheduler, FailedAssetDeleteOnDeviceByCommand, AssetDownloadOnDeviceFromSES, AssetDownloadOnDeviceByCommand, FailedAssetDownloadOnMobile, SucessfulAssetDownloadOnMobile, CorruptedFileStatsFromScheduler, Memory, HubStorage:
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
	r, err := client.SendTelemetry(ctx, &pbTelemetry.TelemetryRequest{ApplicationName: l.applicationName, TelemetryData: message})
	if err != nil {
		log.Println(message)
		log.Println("could not send telemetry to proxy: %v", err)
		return err
	}
	log.Printf("Response from proxy server: %s", r.GetMessage())
	return nil
}

func (l *Logger) Log(eventType string, subType *MessageSubType) {
	if err := EventType(eventType).isValid(); err != nil {
		fmt.Errorf("[Logger]invalid message type %s", eventType)
	}
	tm := new(TelemetryMessage)
	tm.ApplicationName = l.applicationName
	tm.ApplicationVersion = l.applicationVersion
	tm.DeviceIdInData = l.deviceId
	tm.Timestamp = time.Now().Unix()
	switch EventType(eventType) {
	case Liveness:
		tm.Liveness = subType.StringMessage
	case InvalidCommandOnDevice:
		tm.InvalidCommandOnDevice = subType.StringMessage
	case HubStorage:
		tm.HubStorage = subType.FloatValue
	case Memory:
		tm.Memory = subType.FloatValue
	case AssetDeleteOnDeviceByScheduler:
		tm.AssetDeleteOnDeviceByScheduler = &subType.AssetInfo
	case AssetDeleteOnDeviceByCommand:
		tm.AssetDeleteOnDeviceByCommand = &subType.AssetInfo
	case FailedAssetDeleteOnDeviceByScheduler:
		tm.FailedAssetDeleteOnDeviceByScheduler = &subType.AssetInfo
	case FailedAssetDeleteOnDeviceByCommand:
		tm.FailedAssetDeleteOnDeviceByCommand = &subType.AssetInfo
	case AssetDownloadOnDeviceFromSES:
		tm.AssetDownloadOnDeviceFromSES = &subType.AssetInfo
	case AssetDownloadOnDeviceByCommand:
		tm.AssetDownloadOnDeviceByCommand = &subType.AssetInfo
	case FailedAssetDownloadOnMobile:
		tm.FailedAssetDownloadOnMobile = &subType.AssetInfo
	case SucessfulAssetDownloadOnMobile:

		tm.SucessfulAssetDownloadOnMobile = &subType.AssetInfo
	case CorruptedFileStatsFromScheduler:
		tm.CorruptedFileStatsFromScheduler = &subType.IntegrityStats
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

		// subMessage := new(ContentSyncInfoMessage)
		// subMessage.DownloadStatus = messageBody["DownloadStatus"]
		// subMessage.FolderPath = messageBody["FolderPath"]
		// subMessage.Channel = messageBody["Channel"]
		// temp, _ := strconv.ParseFloat(messageBody["AssetSize"], 64)
		// subMessage.AssetSize = temp
		// subMessage.AssetUpdate = messageBody["AssetUpdate"]
		// tm.ContentSyncInfo = subMessage
		// &tm.ContentSyncInfo.DownloadStatus = messageBody["DownloadStatus"]
		// &tm.ContentSyncInfo.FolderPath = messageBody["FolderPath"]
		// &tm.ContentSyncInfo.Channel = messageBody["Channel"]
		// temp, _ := strconv.ParseFloat(messageBody["AssetSize"], 64)
		// &tm.ContentSyncInfo.AssetSize = temp
		// &tm.ContentSyncInfo.AssetUpdate = messageBody["AssetUpdate"]
	}
	b, err := json.Marshal(tm)
	if err != nil {
		log.Println(fmt.Sprintf("[LoggerGRPC] error in creating grpc message: %v", err))
	}
	msgString := string(b)
	fmt.Println("New Message UPSTREAM:")
	fmt.Println(msgString)
	err = l.sendGrpcUpstream(msgString)
	if err != nil {
		log.Println("[LoggerGRPC] error in sending upstream: %v", err)
	}
}
