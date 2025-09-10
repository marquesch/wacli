package whatsapp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"time"

	"github.com/marquesch/wasvc/internal/database"
	"github.com/marquesch/wasvc/internal/model"
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

func init() {
}

type SendTextMessageRequest struct {
	socket.Request
	PhoneNumber     string  `json:"phone_number"`
	Body            string  `json:"body"`
	QuotedMessageID *string `json:"quoted_message_id"`
}

type SendTextMessageHandler struct{}

func (handler *SendTextMessageHandler) Handle(request socket.Request, rawBody json.RawMessage) (socket.Response, chan socket.Event) {
	var sendTextMessageRequest SendTextMessageRequest
	err := json.Unmarshal([]byte(rawBody), &sendTextMessageRequest)
	if err != nil {
		return socket.Response{
			TransactionID: request.TransactionID,
			Status:        "error",
			Error:         err.Error(),
		}, nil
	}

	message, err := sendTextMessage(sendTextMessageRequest.PhoneNumber, sendTextMessageRequest.Body, sendTextMessageRequest.QuotedMessageID)
	if err != nil {
		return socket.Response{
			TransactionID: request.TransactionID,
			Status:        "error",
			Error:         err.Error(),
		}, nil
	}

	return socket.Response{
		TransactionID: request.TransactionID,
		Status:        "success",
		Data:          message,
	}, nil
}

func sendTextMessage(recipient string, body string, quotedMessageID *string) (*model.Message, error) {
	toJID := GetJID(recipient)

	contactExists, err := WhatsappUserExists(toJID)
	if err != nil {
		return nil, fmt.Errorf("error checking contact existence: %w", err)
	}

	if !contactExists {
		return nil, errors.New("recipient is not a valid whatsapp account")
	}

	message := &waE2E.Message{
		Conversation: proto.String(body),
	}

	result, err := WAClient.SendMessage(context.Background(), toJID, message)
	if err != nil {
		return nil, fmt.Errorf("error sending message: %w", err)
	}

	selfID := WAClient.Store.ID

	whatsappUserID, err := database.UpsertWhatsappUser(*selfID, "")
	if err != nil {
		return nil, fmt.Errorf("error upserting self user: %w", err)
	}

	chatID, err := database.UpsertChat(toJID, "", false)
	if err != nil {
		return nil, fmt.Errorf("error upserting chat: %w", err)
	}

	_, err = database.InsertMessage(chatID, whatsappUserID, result.ID, "text", "", body, "", nil, result.Timestamp)
	if err != nil {
		return nil, fmt.Errorf("error inserting message: %w", err)
	}

	return &model.Message{
		ID:              result.ID,
		From:            selfID.String(),
		To:              recipient,
		Type:            "text",
		Body:            &body,
		Timestamp:       result.Timestamp,
		QuotedMessageID: quotedMessageID,
	}, nil
}

type SendMediaMessageRequest struct {
	socket.Request
	PhoneNumber string  `json:"phone_number"`
	FilePath    string  `json:"file_path"`
	Caption     *string `json:"caption"`
}

type SendMediaMessageHandler struct{}

func (handler *SendMediaMessageHandler) Handle(request socket.Request, rawBody json.RawMessage) (socket.Response, chan socket.Event) {
	var sendMediaMessageRequest SendMediaMessageRequest
	err := json.Unmarshal([]byte(rawBody), &sendMediaMessageRequest)
	if err != nil {
		return socket.Response{
			TransactionID: request.TransactionID,
			Status:        "error",
			Error:         err.Error(),
		}, nil
	}

	message, err := SendMediaMessage(sendMediaMessageRequest.PhoneNumber, sendMediaMessageRequest.FilePath, sendMediaMessageRequest.Caption)
	if err != nil {
		return socket.Response{
			TransactionID: request.TransactionID,
			Status:        "error",
			Error:         err.Error(),
		}, nil
	}

	return socket.Response{
		TransactionID: request.TransactionID,
		Status:        "success",
		Data:          message,
	}, nil
}

type CheckWhatsappUserEvent struct {
	PhoneNumber string `json:"phone_number"`
}

func (event *CheckWhatsappUserEvent) Handle() {
	toJID := GetJID(event.PhoneNumber)
	exists, err := WhatsappUserExists(toJID)
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println(exists)
}

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

func SendMediaMessage(phoneNumber string, filePath string, caption *string) (*model.Message, error) {
	toJID := GetJID(phoneNumber)

	contactExists, err := WhatsappUserExists(toJID)
	if err != nil {
		return nil, fmt.Errorf("error checking contact existence: %w", err)
	}

	if !contactExists {
		return nil, errors.New("contact does not exist")
	}

	fileBytes, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed opening file: %w", err)
	}

	var mediaType string
	var uploadResponse whatsmeow.UploadResponse

	mimeType := http.DetectContentType(fileBytes)

	var whatsappMessage waE2E.Message
	var uploadURL string

	switch {
	case imageMimeTypeRegex.MatchString(mimeType):
		mediaType = "image"
		uploadResponse, err = WAClient.Upload(context.Background(), fileBytes, whatsmeow.MediaImage)
		if err != nil {
			return nil, fmt.Errorf("error uploading image to whatsapp servers %w", err)
		}

		// #TODO: set thumbnail to prevent no image showing before downloading on whatsapp
		whatsappMessage.ImageMessage = &waE2E.ImageMessage{
			URL:           &uploadResponse.URL,
			DirectPath:    &uploadResponse.DirectPath,
			MediaKey:      uploadResponse.MediaKey,
			Mimetype:      &mimeType,
			FileSHA256:    uploadResponse.FileSHA256,
			FileEncSHA256: uploadResponse.FileEncSHA256,
			FileLength:    proto.Uint64(uint64(len(fileBytes))),
			Caption:       caption,
		}
		uploadURL = uploadResponse.URL

	case videoMimeTypeRegex.MatchString(mimeType):
		mediaType = "video"
		uploadResponse, err = WAClient.Upload(context.Background(), fileBytes, whatsmeow.MediaVideo)
		if err != nil {
			return nil, fmt.Errorf("error uploading image to whatsapp servers %w", err)
		}

		// #TODO: set thumbnail to prevent no image showing before downloading on whatsapp
		whatsappMessage.VideoMessage = &waE2E.VideoMessage{
			URL:           &uploadResponse.URL,
			DirectPath:    &uploadResponse.DirectPath,
			MediaKey:      uploadResponse.MediaKey,
			Mimetype:      &mimeType,
			FileSHA256:    uploadResponse.FileSHA256,
			FileEncSHA256: uploadResponse.FileEncSHA256,
			FileLength:    proto.Uint64(uint64(len(fileBytes))),
			Caption:       caption,
		}
		uploadURL = uploadResponse.URL

	case audioMimeTypeRegex.MatchString(mimeType):
		mediaType = "audio"
		uploadResponse, err = WAClient.Upload(context.Background(), fileBytes, whatsmeow.MediaAudio)
		if err != nil {
			return nil, fmt.Errorf("error uploading image to whatsapp servers %w", err)
		}

		whatsappMessage.AudioMessage = &waE2E.AudioMessage{
			URL:           &uploadResponse.URL,
			DirectPath:    &uploadResponse.DirectPath,
			MediaKey:      uploadResponse.MediaKey,
			Mimetype:      &mimeType,
			FileSHA256:    uploadResponse.FileSHA256,
			FileEncSHA256: uploadResponse.FileEncSHA256,
			FileLength:    proto.Uint64(uint64(len(fileBytes))),
		}
		uploadURL = uploadResponse.URL

	default:
		mediaType = "document"
		uploadResponse, err = WAClient.Upload(context.Background(), fileBytes, whatsmeow.MediaDocument)
		if err != nil {
			return nil, fmt.Errorf("error uploading image to whatsapp servers %w", err)
		}

		whatsappMessage.DocumentMessage = &waE2E.DocumentMessage{
			URL:           &uploadResponse.URL,
			DirectPath:    &uploadResponse.DirectPath,
			MediaKey:      uploadResponse.MediaKey,
			Mimetype:      &mimeType,
			FileSHA256:    uploadResponse.FileSHA256,
			FileEncSHA256: uploadResponse.FileEncSHA256,
			FileLength:    proto.Uint64(uint64(len(fileBytes))),
		}
		uploadURL = uploadResponse.URL
	}

	result, err := WAClient.SendMessage(context.Background(), toJID, &whatsappMessage)
	if err != nil {
		return nil, fmt.Errorf("error sending message: %w", err)
	}

	selfID := WAClient.Store.ID

	whatsappUserID, err := database.UpsertWhatsappUser(*selfID, "")
	if err != nil {
		return nil, fmt.Errorf("error upserting self user: %w", err)
	}

	chatID, err := database.UpsertChat(toJID, "", false)
	if err != nil {
		return nil, fmt.Errorf("error upserting chat: %w", err)
	}

	_, err = database.InsertMessage(chatID, whatsappUserID, result.ID, "media", mediaType, "", uploadResponse.URL, nil, result.Timestamp)
	if err != nil {
		return nil, fmt.Errorf("error inserting message: %w", err)
	}

	message := model.Message{
		From:      selfID.String(),
		To:        phoneNumber,
		Type:      "media",
		MediaType: &mediaType,
		MediaURL:  &uploadURL,
		ID:        result.ID,
		Timestamp: result.Timestamp,
	}

	return &message, nil
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

func FormatMessage(msg events.Message) string {
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

	return eventMessage
}
