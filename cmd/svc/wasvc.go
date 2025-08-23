package main

import (
	"fmt"
	"net"
	"os"
	"time"

	"github.com/marquesch/wasvc/internal/socket"
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
	go listen(connChan, listener)

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
		go handleClientEvent(conn)
	}

}

func listen(connChan chan net.Conn, listener net.Listener) {
	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("error accepting connection: ", err)
			continue
		}
		connChan <- conn
	}
}

func handleClientEvent(conn net.Conn) error {
	defer conn.Close()

	conn.SetReadDeadline(time.Now().Add(5 * time.Second))

	var clientEvent socket.ClientEvent
	err := socket.ReadEvent(conn, &clientEvent)
	if err != nil {
		return fmt.Errorf("error reading message: %w", err)
	}

	var responseEvent socket.ServerEvent

	switch clientEvent.Command {
	case "send":
		phoneNumber := clientEvent.Args[0]
		body := clientEvent.Args[1]

		responseEvent.Success = true
		err = whatsapp.SendTextMessage(phoneNumber, body)
		if err != nil {
			responseEvent.Success = false
			responseEvent.Message = fmt.Sprintf("error sending text message: %s", err)
		}
	case "check":
		phoneNumber := clientEvent.Args[0]

		toJID := whatsapp.GetJID(phoneNumber)
		contactExists, err := whatsapp.ContactExists(toJID)
		responseEvent.Success = true
		responseEvent.Message = fmt.Sprintf("%t", contactExists)
		if err != nil {
			responseEvent.Success = false
			responseEvent.Message = fmt.Sprintf("error checking if contact exists: %s", err)
		}
	}

	err = socket.WriteEvent(conn, responseEvent)
	if err != nil {
		return fmt.Errorf("error writing message: %w", err)
	}

	return nil
}
