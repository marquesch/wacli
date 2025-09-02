package whatsapp

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"regexp"
	"time"

	"github.com/marquesch/wasvc/internal/database"
	"github.com/marquesch/wasvc/internal/socket"
	"github.com/mdp/qrterminal/v3"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"
)

const (
	colorGreen = "\033[0;32m"
	colorBlue  = "\033[0;34m"
	noColor    = "\033[0m"
)

var (
	WAClient *whatsmeow.Client

	imageMimeTypeRegex = regexp.MustCompile("image/.*")
	videoMimeTypeRegex = regexp.MustCompile("video/.*")
	audioMimeTypeRegex = regexp.MustCompile("audio/.*")
)

func updateDBHandler(evt any) {
	if msg, ok := evt.(*events.Message); ok {
		var err error
		var chatName string
		var mediaURL string
		authorJID := msg.Info.Sender.ToNonAD()
		authorName := msg.Info.PushName
		chatJID := msg.Info.Chat.ToNonAD()
		isGroup := msg.Info.IsGroup
		if !msg.Info.IsFromMe {
			chatName = msg.Info.PushName
		}
		if isGroup {
			groupInfo, err := WAClient.GetGroupInfo(chatJID)
			if err != nil {
				fmt.Println("error trying to get group info: ", err)
				return
			}
			chatName = groupInfo.Name
		}
		whatsappMsgID := msg.Info.ID
		msgType := msg.Info.Type
		mediaType := msg.Info.MediaType
		body := msg.Message.GetConversation()
		msgTimestamp := msg.Info.Timestamp

		switch mediaType {
		case "video":
			mediaURL = *msg.Message.VideoMessage.URL
		case "image":
			mediaURL = *msg.Message.ImageMessage.URL
		case "document":
			mediaURL = *msg.Message.DocumentMessage.URL
		}

		var authorID uint32
		authorID, err = database.UpsertWhatsappUser(authorJID, authorName)
		if err != nil {
			fmt.Println("error upserting whatsapp user: ", err)
			return
		}

		var chatID uint32
		chatID, err = database.UpsertChat(chatJID, chatName, isGroup)
		if err != nil {
			fmt.Println("error upserting chat: ", err)
			return
		}

		_, err = database.InsertMessage(chatID, authorID, whatsappMsgID, msgType, mediaType, body, mediaURL, nil, msgTimestamp)
		if err != nil {
			fmt.Println("error inserting message: ", err)
			return
		}
	}
}

func Connect(successChan chan bool) {
	dbLog := waLog.Stdout("Database", "ERROR", true)

	ctx := context.Background()
	container, err := sqlstore.New(ctx, "sqlite3", fmt.Sprintf("file:%s?_foreign_keys=on", database.WhatsmeowDatabasePath), dbLog)
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
	WAClient.AddEventHandler(updateDBHandler)

	successChan <- true
}

func WhatsappUserExists(jid types.JID) (bool, error) {
	userExists, err := database.CheckUserInDatabase(jid)
	if err != nil {
		return false, fmt.Errorf("error checking user existence in database: %w", err)
	}

	if userExists {
		return true, nil
	}

	userInfo, err := WAClient.GetUserInfo([]types.JID{jid})
	if err != nil {
		return false, fmt.Errorf("error getting user info from client: %w", err)
	}

	userDevices := userInfo[jid].Devices
	return len(userDevices) > 0, nil
}

func GetJID(phoneNumber string) types.JID {
	toJID, _ := types.ParseJID(fmt.Sprintf("%s@s.whatsapp.net", phoneNumber))

	return toJID
}

func SendTextMessage(phoneNumber string, text string) error {
	toJID := GetJID(phoneNumber)

	contactExists, err := WhatsappUserExists(toJID)
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

func SendMediaMessage(phoneNumber string, filePath string, caption string) error {
	toJID := GetJID(phoneNumber)

	contactExists, err := WhatsappUserExists(toJID)
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

	mimeType := http.DetectContentType(fileBytes)

	var message waE2E.Message

	switch {
	case imageMimeTypeRegex.MatchString(mimeType):
		uploadResponse, err := WAClient.Upload(context.Background(), fileBytes, whatsmeow.MediaImage)
		if err != nil {
			return fmt.Errorf("error uploading image to whatsapp servers %w", err)
		}

		// #TODO: set thumbnail to prevent no image showing before downloading on whatsapp
		message.ImageMessage = &waE2E.ImageMessage{
			URL:           &uploadResponse.URL,
			DirectPath:    &uploadResponse.DirectPath,
			MediaKey:      uploadResponse.MediaKey,
			Mimetype:      &mimeType,
			FileSHA256:    uploadResponse.FileSHA256,
			FileEncSHA256: uploadResponse.FileEncSHA256,
			FileLength:    proto.Uint64(uint64(len(fileBytes))),
			Caption:       &caption,
		}

	case videoMimeTypeRegex.MatchString(mimeType):
		uploadResponse, err := WAClient.Upload(context.Background(), fileBytes, whatsmeow.MediaVideo)
		if err != nil {
			return fmt.Errorf("error uploading image to whatsapp servers %w", err)
		}

		// #TODO: set thumbnail to prevent no image showing before downloading on whatsapp
		message.VideoMessage = &waE2E.VideoMessage{
			URL:           &uploadResponse.URL,
			DirectPath:    &uploadResponse.DirectPath,
			MediaKey:      uploadResponse.MediaKey,
			Mimetype:      &mimeType,
			FileSHA256:    uploadResponse.FileSHA256,
			FileEncSHA256: uploadResponse.FileEncSHA256,
			FileLength:    proto.Uint64(uint64(len(fileBytes))),
			Caption:       &caption,
		}

	case audioMimeTypeRegex.MatchString(mimeType):
		uploadResponse, err := WAClient.Upload(context.Background(), fileBytes, whatsmeow.MediaAudio)
		if err != nil {
			return fmt.Errorf("error uploading image to whatsapp servers %w", err)
		}

		message.AudioMessage = &waE2E.AudioMessage{
			URL:           &uploadResponse.URL,
			DirectPath:    &uploadResponse.DirectPath,
			MediaKey:      uploadResponse.MediaKey,
			Mimetype:      &mimeType,
			FileSHA256:    uploadResponse.FileSHA256,
			FileEncSHA256: uploadResponse.FileEncSHA256,
			FileLength:    proto.Uint64(uint64(len(fileBytes))),
		}

	default:
		uploadResponse, err := WAClient.Upload(context.Background(), fileBytes, whatsmeow.MediaDocument)
		if err != nil {
			return fmt.Errorf("error uploading image to whatsapp servers %w", err)
		}

		message.DocumentMessage = &waE2E.DocumentMessage{
			URL:           &uploadResponse.URL,
			DirectPath:    &uploadResponse.DirectPath,
			MediaKey:      uploadResponse.MediaKey,
			Mimetype:      &mimeType,
			FileSHA256:    uploadResponse.FileSHA256,
			FileEncSHA256: uploadResponse.FileEncSHA256,
			FileLength:    proto.Uint64(uint64(len(fileBytes))),
		}
	}

	_, err = WAClient.SendMessage(context.Background(), toJID, &message)
	if err != nil {
		return fmt.Errorf("error sending message: %w", err)
	}

	return nil
}

func GetMessageEvents(msgChan chan events.Message, toJID types.JID) uint32 {
	eventHandlerId := WAClient.AddEventHandler(func(evt any) {
		if msg, ok := evt.(*events.Message); ok {
			if msg.Message == nil || msg.Info.Chat != toJID {
				return
			}

			msgChan <- *msg
		}
	})

	return eventHandlerId
}

func StreamMessages(ctx context.Context, conn net.Conn, phoneNumber string) {
	toJID := GetJID(phoneNumber)

	msgChan := make(chan events.Message)
	eventHandlerId := GetMessageEvents(msgChan, toJID)

	go func() {
		defer close(msgChan)
		defer WAClient.RemoveEventHandler(eventHandlerId)
		for {
			select {
			case msg := <-msgChan:
				var color string

				if msg.Info.IsFromMe {
					color = colorGreen
				} else {
					color = colorBlue
				}

				var body string
				switch msg.Info.Type {
				case "text":
					body = msg.Message.GetConversation()

				case "media":
					body = msg.Info.MediaType
				}

				eventMessage := fmt.Sprintf("\n%s%s %s%s", color, msg.Info.Timestamp.Format(time.TimeOnly), body, noColor)

				event := socket.ServerResponse{Success: true, Message: eventMessage}
				err := socket.WriteEvent(conn, event)
				if err != nil {
					fmt.Println("error writing MessageReceivedEvent: ", err)
				}

			case <-ctx.Done():
				return
			}
		}
	}()
}
