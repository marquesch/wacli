package cli

import (
	"bufio"
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

	reader := bufio.NewReader(conn)
	err = socket.ReadEvent(reader, &response)
	if err != nil {
		fmt.Println("read error: ", err)
		os.Exit(1)
	}

	return response, nil
}
