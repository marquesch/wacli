package wasv

import (
	"fmt"
	"net"
	"time"

	"github.com/marquesch/wasvc/internal/socket"
	"github.com/marquesch/wasvc/internal/whatsapp"
)

func HandleConnection(conn net.Conn) error {
	defer conn.Close()

	conn.SetReadDeadline(time.Now().Add(5 * time.Second))

	var clientCommand socket.ClientCommand
	err := socket.ReadEvent(conn, &clientCommand)
	if err != nil {
		return fmt.Errorf("error reading message: %w", err)
	}

	var response socket.ServerResponse

	switch clientCommand.Command {
	case "send":
		err = sendMessage(clientCommand.Subcommand, clientCommand.Args)
		if err != nil {
			response.Message = fmt.Sprintf("%s", err)
		} else {
			response.Success = true
		}

	case "check":
		contactExists, err := checkContact(clientCommand.Args[0])
		response.Success = true
		response.Message = fmt.Sprintf("%t", contactExists)
		if err != nil {
			response.Success = false
			response.Message = fmt.Sprintf("error checking if contact exists: %s", err)
		}
	}

	err = socket.WriteEvent(conn, response)
	if err != nil {
		return fmt.Errorf("error writing message: %w", err)
	}

	return nil
}

func sendMessage(messageType string, args []string) error {
	switch messageType {
	case "text":
		phoneNumber := args[0]
		body := args[1]

		err := whatsapp.SendTextMessage(phoneNumber, body)
		if err != nil {
			return fmt.Errorf("error sending text message: %w", err)
		}
	}
	return nil
}

func checkContact(phoneNumber string) (bool, error) {
	toJID := whatsapp.GetJID(phoneNumber)

	contactExists, err := whatsapp.ContactExists(toJID)
	if err != nil {
		return false, fmt.Errorf("error checking if contact exists: %w", err)
	}
	return contactExists, nil
}
