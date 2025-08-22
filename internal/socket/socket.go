package socket

import (
	"encoding/json"
	"fmt"
	"net"
)

type SocketMessage struct {
	Message string `json:"message"`
	ErrStr  string `json:"err_str"`
}

func ReadMessage(conn net.Conn) (SocketMessage, error) {
	var socketMessage SocketMessage

	buf := make([]byte, 1024)
	n, err := conn.Read(buf)
	if err != nil {
		return socketMessage, fmt.Errorf("error reading from connection: %w", err)
	}

	err = json.Unmarshal(buf[:n], &socketMessage)
	if err != nil {
		return socketMessage, fmt.Errorf("error unmarshaling json: %w", err)
	}

	return socketMessage, nil
}

func WriteMessage(conn net.Conn, socketMessage SocketMessage) error {
	message, err := json.Marshal(socketMessage)
	if err != nil {
		return fmt.Errorf("error marshaling json: %w", err)
	}

	_, err = conn.Write([]byte(message))
	if err != nil {
		return fmt.Errorf("error writing to channel: %w", err)
	}

	return nil
}
