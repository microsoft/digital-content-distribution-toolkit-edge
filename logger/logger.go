// custom logger implementaion to handle telemetry as well
package logger

import (
	// "io"
	"fmt"
    "log"
	"os"
	"bufio"
	"encoding/json"
    "syscall"
    "time"
)
type LogType string;

type Message struct {
	DeviceId string
	// MessageType string
	MessageSubType string
	TimeStamp string
	MessageBody map[string]string
}

const (
    Telemetry LogType = "Telemetry"
    Debug = "Debug"
    Info = "Info"
    Warning = "Warning"
    Error = "Error"
    Critical = "Critical"
)

func (lt LogType) isValid() error {
    switch lt {
    case Telemetry, Debug, Info, Warning, Error, Critical:
        return nil
    }
    return fmt.Errorf("Inalid log type %v", string(lt))
}


type Logger struct{
	deviceId string
	file *os.File
	writer *bufio.Writer
}

func MakeLogger(deviceId string, logFilePath string, bufferSize int) *Logger {
	fmt.Println(">>>>>>", deviceId, logFilePath, bufferSize)
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
    if err := LogType(messageType).isValid(); err != nil{
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
	fmt.Println(msgString)

	err = l.lockFile()
	if err != nil {
		log.Println("[Logger] error in locking file")
	}

	l.writer.WriteString(msgString+"\n")
	if(messageType == "Telemetry" || messageType == "Critical" || messageType == "Error") {
		l.writer.Flush()
		l.file.Sync()
	}

	err = l.unlockFile()
	if err != nil {
		log.Println("[Logger] error in unlocking file")
	}
}

// Standalone testing of logger module:
// func dummy(n int, logger Logger) {
//     for i := 4; i >= 0; i-- {
//         if(i > 0) {
//             logger.Log("Message", "valid division"); 
//         } else {
//             logger.Log("Critical", "division by zero");
//         }
//     }
// }

// func main() {
//     fmt.Println("Making logger...");
//     logger := MakeLogger();

//     logger.Log("Warning", "This is just a dummy testing!")

//     fmt.Println("\n");

//     dummy(4, logger);
// }