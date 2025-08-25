package main

import (
	"fmt"
	"net"
	"os"

	"github.com/marquesch/wasvc/internal/socket"
	"github.com/marquesch/wasvc/internal/wasv"
	"github.com/marquesch/wasvc/internal/whatsapp"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	listener, err := socket.StartServer()
	if err != nil {
		fmt.Println("error starting server: ", err)
		os.Exit(1)
	}
	defer listener.Close()
	defer os.Remove(socket.SocketPath)

	successChan := make(chan bool)
	connChan := make(chan net.Conn)

	go whatsapp.Connect(successChan)
	go socket.Listen(connChan, listener)

	var retries int
	var done bool

	for !done {
		select {
		case success := <-successChan:
			if !success {
				if retries < 3 {
					fmt.Println("error connecting to whatsapp. retrying")
					whatsapp.Connect(successChan)
					retries++
				} else {
					fmt.Println(("could not connect to whatsapp after 3 tries. shutting down"))
					os.Exit(1)
				}
			} else {
				done = true
			}
		case conn := <-connChan:
			response := socket.ServerEvent{
				Success: false,
				Message: "whatsapp client is still connecting",
			}
			err := socket.WriteEvent(conn, response)
			if err != nil {
				fmt.Println("error writing event: ", err)
			}
		}
	}
	defer whatsapp.WAClient.Disconnect()

	for {
		conn := <-connChan
		go wasv.HandleConnection(conn)
	}

}
