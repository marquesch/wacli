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

	var clientEvent socket.ClientEvent
	err := socket.ReadEvent(conn, &clientEvent)
	if err != nil {
		return fmt.Errorf("error reading message: %w", err)
	}

	var responseEvent socket.ServerEvent

	switch clientEvent.Command {
	case "send":
		phoneNumber := clientEvent.Args[0]
		body := clientEvent.Args[1]

		responseEvent.Success = true
		err = whatsapp.SendTextMessage(phoneNumber, body)
		if err != nil {
			responseEvent.Success = false
			responseEvent.Message = fmt.Sprintf("error sending text message: %s", err)
		}
	case "check":
		phoneNumber := clientEvent.Args[0]

		toJID := whatsapp.GetJID(phoneNumber)
		contactExists, err := whatsapp.ContactExists(toJID)
		responseEvent.Success = true
		responseEvent.Message = fmt.Sprintf("%t", contactExists)
		if err != nil {
			responseEvent.Success = false
			responseEvent.Message = fmt.Sprintf("error checking if contact exists: %s", err)
		}
	}

	err = socket.WriteEvent(conn, responseEvent)
	if err != nil {
		return fmt.Errorf("error writing message: %w", err)
	}

	return nil
}
