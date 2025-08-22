package main

import (
	"fmt"
	"math/rand"
	"net"
	"os"
	"time"

	"github.com/marquesch/wasvc/internal/socket"
)

const socketPath = "/tmp/app.sock"

func main() {
	if err := os.RemoveAll(socketPath); err != nil {
		fmt.Println("Error removing old socket:", err)
		os.Exit(1)
	}

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		fmt.Println("Listen error:", err)
		os.Exit(1)
	}

	defer func() {
		listener.Close()
		os.Remove(socketPath)
	}()

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Accept error:", err)
			continue
		}
		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) error {
	defer conn.Close()

	conn.SetReadDeadline(time.Now().Add(5 * time.Second))

	message, err := socket.ReadMessage(conn)
	if err != nil {
		return fmt.Errorf("error reading message: %w", err)
	}

	var response socket.SocketMessage

	if rand.Intn(100) > 50 {
		response.Message = fmt.Sprintf("success! your message was: %s", message.Message)
		response.ErrStr = ""
	} else {
		response.Message = fmt.Sprintf("failed! your message was: %s", message.Message)
		response.ErrStr = "faileddddddd. good luck next time"
	}

	err = socket.WriteMessage(conn, response)
	if err != nil {
		return fmt.Errorf("error writing message: %w", err)
	}

	return nil
}
