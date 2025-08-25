package cli

import (
	"fmt"
	"net"
	"os"

	"github.com/marquesch/wasvc/internal/socket"
)

func SendCommand(clientEvent socket.ClientEvent) (socket.ServerEvent, error) {
	conn, err := net.Dial("unix", socket.SocketPath)
	if err != nil {
		fmt.Println("Dial error:", err)
		os.Exit(1)
	}
	defer conn.Close()

	err = socket.WriteEvent(conn, clientEvent)
	if err != nil {
		fmt.Println("write error: ", err)
		os.Exit(1)
	}

	var response socket.ServerEvent

	err = socket.ReadEvent(conn, &response)
	if err != nil {
		fmt.Println("read error: ", err)
		os.Exit(1)
	}

	return response, nil
}
