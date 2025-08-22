package socket

import (
	"encoding/json"
	"fmt"
	"net"
)

type ClientEvent struct {
	Command string   `json:"command"`
	Args    []string `json:"args"`
}

type ServerEvent struct {
	Success bool `json:"success"`
}

type Event interface {
	ClientEvent | ServerEvent
}

func ReadEvent[T Event](conn net.Conn, event *T) error {
	buf := make([]byte, 1024)
	n, err := conn.Read(buf)
	if err != nil {
		return fmt.Errorf("error reading from connection: %w", err)
	}

	err = json.Unmarshal(buf[:n], &event)
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

	_, err = conn.Write([]byte(message))
	if err != nil {
		return fmt.Errorf("error writing to channel: %w", err)
	}

	return nil
}
