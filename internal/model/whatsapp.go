package model

import "time"

type Message struct {
	From            string    `json:"from"`
	To              string    `json:"to"`
	Type            string    `json:"type"`
	ID              string    `json:"id"`
	MediaType       *string   `json:"media_type"`
	MediaURL        *string   `json:"media_url"`
	Body            *string   `json:"body"`
	Timestamp       time.Time `json:"timestamp"`
	QuotedMessageID *string   `json:"quoted_message_id"`
}
