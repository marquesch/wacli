package cli

import (
	"fmt"
	"net"
	"os"

	"github.com/marquesch/wasvc/internal/socket"
)

func SendCommand(command socket.ClientCommand) (socket.ServerResponse, error) {
	conn, err := net.Dial("unix", socket.SocketPath)
	if err != nil {
		fmt.Println("Dial error:", err)
		os.Exit(1)
	}
	defer conn.Close()

	err = socket.WriteEvent(conn, command)
	if err != nil {
		fmt.Println("write error: ", err)
		os.Exit(1)
	}

	var response socket.ServerResponse

	err = socket.ReadEvent(conn, &response)
	if err != nil {
		fmt.Println("read error: ", err)
		os.Exit(1)
	}

	return response, nil
}

func SendCommandNoWait(command socket.ClientCommand) error {
	conn, err := net.Dial("unix", socket.SocketPath)
	if err != nil {
		return fmt.Errorf("dial error: %w", err)
	}
	defer conn.Close()

	err = socket.WriteEvent(conn, command)
	if err != nil {
		return fmt.Errorf("write error: %w", err)
	}

	return nil
}
