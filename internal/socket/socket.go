package socket

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
)

const SocketPath = "/tmp/app.sock"

type ClientCommand struct {
	Command    string `json:"command"`
	Subcommand string `json:"subcommand"`
	Args       []any  `json:"args"`
}

type ServerResponse struct {
	Success bool   `json:"success"`
	Message string `json:"response"`
}

type Event interface {
	ClientCommand | ServerResponse
}

func Accept(connChan chan net.Conn, listener net.Listener) {
	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("error accepting connection: ", err)
			continue
		}
		connChan <- conn
	}
}

func StartServer() (net.Listener, error) {
	err := os.RemoveAll(SocketPath)
	if err != nil {
		fmt.Println("Error removing old socket:", err)
		os.Exit(1)
	}

	listener, err := net.Listen("unix", SocketPath)
	if err != nil {
		fmt.Println("Listen error:", err)
		os.Exit(1)
	}

	return listener, nil
}

func ReadEvent[T Event](reader *bufio.Reader, event *T) error {
	buffer, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("error reading from connection: %w", err)
	}

	err = json.Unmarshal([]byte(buffer), &event)
	if err != nil {
		return fmt.Errorf("error unmarshaling json: %w", err)
	}

	return nil
}

func WriteEvent[T Event](conn net.Conn, event T) error {
	message, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("error marshaling json: %w", err)
	}

	_, err = conn.Write((append(message, '\n')))
	if err != nil {
		return fmt.Errorf("error writing to channel: %w", err)
	}

	return nil
}
