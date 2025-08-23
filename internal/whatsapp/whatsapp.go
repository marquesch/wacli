package whatsapp

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/mdp/qrterminal/v3"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"
)

func Connect() (*whatsmeow.Client, error) {
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
	client.SendPresence(types.PresenceAvailable)

	return client, nil
}

func ContactExists(client *whatsmeow.Client, jid types.JID) (bool, error) {
	usersInfo, err := client.GetUserInfo([]types.JID{jid})
	if err != nil {
		return false, err
	}

	userDevices := usersInfo[jid].Devices
	return len(userDevices) > 0, nil
}

func GetJID(phoneNumber string) types.JID {
	toJID, _ := types.ParseJID(fmt.Sprintf("%s@s.whatsapp.net", phoneNumber))

	return toJID
}

func SendTextMessage(client *whatsmeow.Client, phoneNumber string, text string) error {
	toJID := GetJID(phoneNumber)

	contactExists, err := ContactExists(client, toJID)
	if err != nil {
		return fmt.Errorf("error checking contact existence: %w", err)
	}

	if !contactExists {
		return errors.New("contact does not exist")
	}

	message := &waE2E.Message{
		Conversation: proto.String(text),
	}

	_, err = client.SendMessage(context.Background(), toJID, message)
	if err != nil {
		return fmt.Errorf("error sending message: %w", err)
	}

	return nil
}
