package whatsapp

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/mdp/qrterminal/v3"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"
)

var (
	databasePath string
	WAClient     *whatsmeow.Client
)

func init() {
	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}

	databasePath = filepath.Join(userHomeDir, ".local", "lib", "wacli", "sqlite.db")
	dbDir := filepath.Dir(databasePath)

	err = os.MkdirAll(dbDir, 0755)
	if err != nil {
		panic(err)
	}
}

func Connect(successChan chan bool) {
	dbLog := waLog.Stdout("Database", "ERROR", true)

	ctx := context.Background()
	container, err := sqlstore.New(ctx, "sqlite3", fmt.Sprintf("file:%s?_foreign_keys=on", databasePath), dbLog)
	if err != nil {
		successChan <- false
	}

	deviceStore, err := container.GetFirstDevice(ctx)
	if err != nil {
		successChan <- false
	}

	clientLog := waLog.Stdout("Client", "ERROR", true)
	WAClient = whatsmeow.NewClient(deviceStore, clientLog)

	if WAClient.Store.ID == nil {
		qrChan, _ := WAClient.GetQRChannel(context.Background())

		err = WAClient.Connect()
		if err != nil {
			successChan <- false
		}

		for evt := range qrChan {
			if evt.Event == "code" {
				qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)
			}
		}

	} else {
		err = WAClient.Connect()
		if err != nil {
			successChan <- false
		}
	}
	WAClient.SendPresence(types.PresenceAvailable)

	successChan <- true
}

func ContactExists(jid types.JID) (bool, error) {
	usersInfo, err := WAClient.GetUserInfo([]types.JID{jid})
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

func SendTextMessage(phoneNumber string, text string) error {
	toJID := GetJID(phoneNumber)

	contactExists, err := ContactExists(toJID)
	if err != nil {
		return fmt.Errorf("error checking contact existence: %w", err)
	}

	if !contactExists {
		return errors.New("contact does not exist")
	}

	message := &waE2E.Message{
		Conversation: proto.String(text),
	}

	_, err = WAClient.SendMessage(context.Background(), toJID, message)
	if err != nil {
		return fmt.Errorf("error sending message: %w", err)
	}

	return nil
}

func SendImageMessage(phoneNumber string, filePath string, caption string) error {
	toJID := GetJID(phoneNumber)

	contactExists, err := ContactExists(toJID)
	if err != nil {
		return fmt.Errorf("error checking contact existence: %w", err)
	}

	if !contactExists {
		return errors.New("contact does not exist")
	}

	fileBytes, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed opening file: %w", err)
	}

	uploadResponse, err := WAClient.Upload(context.Background(), fileBytes, whatsmeow.MediaImage)
	if err != nil {
		return fmt.Errorf("error uploading image to whatsapp servers %w", err)
	}

	message := waE2E.Message{
		ImageMessage: &waE2E.ImageMessage{
			URL:           &uploadResponse.URL,
			Caption:       &caption,
			MediaKey:      uploadResponse.MediaKey,
			FileSHA256:    uploadResponse.FileSHA256,
			FileEncSHA256: uploadResponse.FileEncSHA256,
		},
	}

	_, err = WAClient.SendMessage(context.Background(), toJID, &message)
	if err != nil {
		return fmt.Errorf("error sending message: %w", err)
	}

	return nil
}

func ListenEvents() (chan any, error) {
	eventsChannel := make(chan any)
	WAClient.AddEventHandler(func(evt any) {
		switch evt.(type) {
		case *events.Message:
			eventsChannel <- evt
		}
	})

	return eventsChannel, nil
}
