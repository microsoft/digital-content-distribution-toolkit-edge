// custom logger implementaion to handle telemetry as well
package logger

import (
	// "io"
	"fmt"
    "log"
	"os"
	"bufio"
    "syscall"
    "time"
)
type LogType string;

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
	file *os.File
	writer *bufio.Writer
}

func MakeLogger(logFilePath string, bufferSize int) *Logger {
	file, err := os.OpenFile(logFilePath, syscall.O_CREAT|syscall.O_WRONLY|syscall.O_APPEND, 0666)
    if err != nil {
        log.Fatalf("[Logger]error in opening file: %s", logFilePath)
	}
	
	writer := bufio.NewWriterSize(file, bufferSize)
	l := Logger{file, writer}

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

func (l *Logger) Log(logType LogType, logString string) {
    if err := logType.isValid(); err != nil{
        fmt.Errorf("[Logger]invalid log type %s", logType)
	}
	
	err := l.lockFile()
	if err != nil {
		log.Printf("[Logger] error in locking file")
	}

	l.writer.WriteString(fmt.Sprintf("[%s]%s %s\n", logType, time.Now().String(), logString))
	if(logType == "Telemetry" || logType == "Critical" || logType == "Error") {
		l.writer.Flush()
		l.file.Sync()
	}

	err = l.unlockFile()
	if err != nil {
		log.Printf("[Logger] error in unlocking file")
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