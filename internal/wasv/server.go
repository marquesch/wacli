package wasv

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"time"

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
		phoneNumber, _ := clientCommand.Args[0].(string)
		contactExists, err := checkContact(phoneNumber)
		response.Success = true
		response.Message = fmt.Sprintf("%t", contactExists)
		if err != nil {
			response.Success = false
			response.Message = fmt.Sprintf("error checking if contact exists: %s", err)
		}

	case "get":
		fmt.Println(clientCommand.Args)
		phoneNumber, _ := clientCommand.Args[0].(string)
		var tail int
		if value, ok := clientCommand.Args[1].(float64); ok {
			tail = int(value)
		}
		follow, _ := clientCommand.Args[2].(bool)

		ctx, cancel := context.WithCancel(context.Background())

		response.Success = true
		err = socket.WriteEvent(conn, response)

		persistedMessages, err := database.GetMessages(whatsapp.GetJID(phoneNumber), tail)
		if err != nil {
			cancel()
			return fmt.Errorf("error getting messages from whatspp_user: %w", err)
		}

		var lastDate time.Time
		for _, persistedMessage := range persistedMessages {
			messageDay := persistedMessage.Info.Timestamp.Truncate(24 * time.Hour)
			if lastDate.Before(messageDay) {
				lastDate = messageDay
				event := socket.ServerResponse{Success: true, Message: fmt.Sprintf("\n\n%s\n", lastDate.Format("Mon Jan _2"))}
				err := socket.WriteEvent(conn, event)
				if err != nil {
					cancel()
					fmt.Println("error writing MessageReceivedEvent: ", err)
				}
			}
			eventMessage := whatsapp.FormatMessage(persistedMessage)
			event := socket.ServerResponse{Success: true, Message: eventMessage}
			err := socket.WriteEvent(conn, event)
			if err != nil {
				cancel()
				fmt.Println("error writing MessageReceivedEvent: ", err)
			}

		}
		if !follow {
			cancel()
			return nil
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

func sendMessage(messageType string, args []any) error {
	switch messageType {
	case "text":
		phoneNumber, _ := args[0].(string)
		body, _ := args[1].(string)

		err := whatsapp.SendTextMessage(phoneNumber, body)
		if err != nil {
			return fmt.Errorf("error sending text message: %w", err)
		}
	case "media":
		phoneNumber, _ := args[0].(string)
		filePath, _ := args[1].(string)
		caption, _ := args[2].(string)

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
