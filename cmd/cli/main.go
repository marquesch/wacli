package main

import (
	"fmt"
	"net"
	"os"

	"github.com/marquesch/wasvc/internal/socket"
)

const socketPath = "/tmp/app.sock"

func main() {
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		fmt.Println("Dial error:", err)
		os.Exit(1)
	}
	defer conn.Close()

	message := socket.ClientEvent{
		Command: "send",
		Args:    []string{"number", "msg"},
	}

	err = socket.WriteEvent(conn, message)
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

	fmt.Printf("got succesful response: %t", response.Success)
}
