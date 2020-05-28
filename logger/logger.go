// custom logger implementaion to handle telemetry as well
package logger

import (
    "fmt"
    "log"
    "time"
    "context"
    "errors"

    "google.golang.org/grpc"

    pb "./pblogger"
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
    return errors.New(fmt.Sprintf("Inalid log type %v", string(lt)))
}


type Logger struct{
    client pb.LogClient
}

func MakeLogger(port int) Logger {
    connection_string := fmt.Sprintf("localhost:%v", port)
    conn, err := grpc.Dial(connection_string, grpc.WithInsecure())
    if err != nil {
        log.Fatalf("Logger Error: failed to dial: %v", err)
    }

    client := pb.NewLogClient(conn)

    return Logger{client}
}

func (l Logger) Log(log_type LogType, log_string string) {
    if err := log_type.isValid(); err != nil{
        log.Fatalf("Logger Error: %v", err)
    }

    ctx := context.Background()
    request := pb.SingleLog{Logtype: string(log_type), Logstring: time.Now().String() + " " + log_string}

    // ignore the reponse
    _, err2 := l.client.SendSingleLog(ctx, &request)
    if err2 != nil {
        log.Fatalf("Logger Error: %v", err2)
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