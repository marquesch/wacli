package main

import (
	"log"
	"os"
	"time"
)

func main() {
	file, err := os.OpenFile("/var/log/wasvc/wasvc.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	log.SetOutput(file)
	for {
		log.Println("Service is still up and running")
		time.Sleep(time.Second)
	}
}
