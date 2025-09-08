package model

import (
	"time"

	"go.mau.fi/whatsmeow/types"
)

type WhatsappUser struct {
	JID       types.JID `json:"jid"`
	Pushname  string    `json:"pushname"`
	ChannelID string    `json:"channel_id"`
}

type Message struct {
	From            WhatsappUser `json:"from"`
	To              WhatsappUser `json:"to"`
	Type            string       `json:"type"`
	ID              string       `json:"id"`
	MediaType       *string      `json:"media_type"`
	MediaURL        *string      `json:"media_url"`
	Body            *string      `json:"body"`
	Timestamp       time.Time    `json:"timestamp"`
	QuotedMessageID *string      `json:"quoted_message_id"`
}
