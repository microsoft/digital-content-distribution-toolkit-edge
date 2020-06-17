package main

import (
	"fmt"
	"time"
)

const _interval time.Duration = 120

func checkIntegrity() {
	for true {
		time.Sleep(_interval * time.Second)
		fmt.Println("Info", "Checking files integrity from background thread")
		navigate("root")
	}
}
