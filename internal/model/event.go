package model

import (
	"encoding/json"
	"fmt"

	"github.com/marquesch/wasvc/internal/whatsapp"
)

type Payload struct {
	Type          string  `json:"type"`
	TransactionID *string `json:"transaction_id"`
	Event         Event   `json:"event"`
}

type Event interface {
	Handle()
}

type Response interface{}

type MessageSentResponse struct {
	TransactionID *string `json:"transaction_id"`
	Status        string  `json:"status"`
	Message       Message `json:"message"`
}

type SendTextMessage struct {
	PhoneNumber string `json:"phone_number"`
	Body        string `json:"body"`
}

func (event *SendTextMessage) Handle() {
	whatsapp.SendTextMessage(event.PhoneNumber, event.Body)
}

type SendMediaMessage struct {
	PhoneNumber string  `json:"phone_number"`
	FilePath    string  `json:"file_path"`
	Caption     *string `json:"caption"`
}

func (event *SendMediaMessage) Handle() {
	whatsapp.SendMediaMessage(event.PhoneNumber, event.FilePath, event.Caption)
}

type CheckWhatsappUser struct {
	PhoneNumber string `json:"phone_number"`
}

func (event *CheckWhatsappUser) Handle() {
	toJID := whatsapp.GetJID(event.PhoneNumber)
	exists, err := whatsapp.WhatsappUserExists(toJID)
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println(exists)
}

func EventFactory(rawPayload string) (*Payload, error) {
	var payload struct {
		Type          string          `json:"type"`
		TransactionID *string         `json:"transaction_id"`
		Event         json.RawMessage `json:"event"`
	}

	err := json.Unmarshal([]byte(rawPayload), &payload)
	if err != nil {
		return nil, err
	}

	var event Event
	switch payload.Type {
	case "send_text_message":
		event = &SendTextMessage{}
	case "send_media_message":
		event = &SendMediaMessage{}
	case "check_whatsapp_user":
		event = &CheckWhatsappUser{}
	}

	err = json.Unmarshal([]byte(payload.Event), &event)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling payload event: %w", err)
	}

	return &Payload{
		Type:          payload.Type,
		TransactionID: payload.TransactionID,
		Event:         event,
	}, nil
}
