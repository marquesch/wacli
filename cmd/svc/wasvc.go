package main

import (
	"fmt"
	"net"
	"os"
	"time"

	"context"

	"github.com/marquesch/wasvc/internal/socket"
	"github.com/marquesch/wasvc/internal/whatsapp"

	_ "github.com/mattn/go-sqlite3"
	"github.com/mdp/qrterminal/v3"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"

	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"
)

func whatsappConnect() (*whatsmeow.Client, error) {
	dbLog := waLog.Stdout("Database", "ERROR", true)

	ctx := context.Background()
	container, err := sqlstore.New(ctx, "sqlite3", "file:examplestore.db?_foreign_keys=on", dbLog)
	if err != nil {
		return nil, fmt.Errorf("error setting db: %w", err)
	}

	deviceStore, err := container.GetFirstDevice(ctx)
	if err != nil {
		return nil, fmt.Errorf("error setting device: %w", err)
	}

	clientLog := waLog.Stdout("Client", "ERROR", true)
	client := whatsmeow.NewClient(deviceStore, clientLog)

	if client.Store.ID == nil {
		qrChan, _ := client.GetQRChannel(context.Background())

		err = client.Connect()
		if err != nil {
			return nil, fmt.Errorf("error connecting to client: %w", err)
		}

		for evt := range qrChan {
			if evt.Event == "code" {
				qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)
			}
		}

	} else {
		err = client.Connect()
		if err != nil {
			return nil, fmt.Errorf("error connecting to client: %w", err)
		}
	}

	return client, nil
}

func sendTextMessage(client *whatsmeow.Client, recipient string, text string) error {
	toJID, _ := types.ParseJID(fmt.Sprintf("%s@s.whatsapp.net", recipient))

	message := &waE2E.Message{
		Conversation: proto.String(text),
	}

	_, err := client.SendMessage(context.Background(), toJID, message)
	if err != nil {
		return fmt.Errorf("error sending message: %w", err)
	}

	return nil
}

func main() {
	waClient, err := whatsapp.Connect()
	if err != nil {
		fmt.Println("Error connecting to whatsapp: ", err)
		os.Exit(1)
	}
	// defer waClient.SendPresence(types.PresenceUnavailable)
	defer waClient.Disconnect()

	listener, err := socket.StartServer()
	if err != nil {
		fmt.Println("error starting server: ", err)
		os.Exit(1)
	}
	defer listener.Close()
	defer os.Remove(socket.SocketPath)

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("error accepting connection: ", err)
			continue
		}
		go handleClientEvent(conn, waClient)
	}
}

func handleClientEvent(conn net.Conn, waClient *whatsmeow.Client) error {
	defer conn.Close()

	conn.SetReadDeadline(time.Now().Add(5 * time.Second))

	var clientEvent socket.ClientEvent
	err := socket.ReadEvent(conn, &clientEvent)
	if err != nil {
		return fmt.Errorf("error reading message: %w", err)
	}

	var response socket.ServerEvent

	switch clientEvent.Command {
	case "send":
		phoneNumber := clientEvent.Args[0]
		body := clientEvent.Args[1]

		response.Success = true
		err = whatsapp.SendTextMessage(waClient, phoneNumber, body)
		if err != nil {
			response.Success = false
			response.Error = fmt.Sprintf("error sending text message: %s", err)
		}
	case "check":
		phoneNumber := clientEvent.Args[0]

		toJID := whatsapp.GetJID(phoneNumber)
		contactExists, err := whatsapp.ContactExists(waClient, toJID)
		if err != nil {
			response.Error = fmt.Sprintf("error checking if contact exists: %s", err)
		}
		response.Success = contactExists
	}

	err = socket.WriteEvent(conn, response)
	if err != nil {
		return fmt.Errorf("error writing message: %w", err)
	}

	return nil
}
