package wasv

import (
	"bufio"
	"context"
	"fmt"
	"net"

	"github.com/marquesch/wasvc/internal/database"
	"github.com/marquesch/wasvc/internal/socket"
	"github.com/marquesch/wasvc/internal/whatsapp"
)

func HandleConnection(conn net.Conn) error {
	reader := bufio.NewReader(conn)
	defer conn.Close()

	var clientCommand socket.ClientCommand
	err := socket.ReadEvent(reader, &clientCommand)
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

	case "get":
		phoneNumber := clientCommand.Args[0]
		ctx, cancel := context.WithCancel(context.Background())

		response.Success = true
		err = socket.WriteEvent(conn, response)

		persistedMessages, err := database.GetMessages(whatsapp.GetJID(phoneNumber))
		if err != nil {
			return fmt.Errorf("error getting messages from whatspp_user: %w", err)
		}
		for _, persistedMessage := range persistedMessages {
			eventMessage := whatsapp.FormatMessage(persistedMessage)
			event := socket.ServerResponse{Success: true, Message: eventMessage}
			err := socket.WriteEvent(conn, event)
			if err != nil {
				fmt.Println("error writing MessageReceivedEvent: ", err)
			}
			// time.Sleep(time.Millisecond)

		}
		whatsapp.StreamMessages(ctx, conn, phoneNumber)
		for {
			err = socket.ReadEvent(reader, &clientCommand)
			if err != nil {
				cancel()
				return fmt.Errorf("error reading event from client: %w", err)
			}

			if clientCommand.Command == "cancel" {
				cancel()
				return nil
			}
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
	case "media":
		phoneNumber := args[0]
		filePath := args[1]
		caption := args[2]

		err := whatsapp.SendMediaMessage(phoneNumber, filePath, caption)
		if err != nil {
			return fmt.Errorf("error media image message: %w", err)
		}
	}

	return nil
}

func checkContact(phoneNumber string) (bool, error) {
	toJID := whatsapp.GetJID(phoneNumber)

	contactExists, err := whatsapp.WhatsappUserExists(toJID)
	if err != nil {
		return false, fmt.Errorf("error checking if contact exists: %w", err)
	}
	return contactExists, nil
}
