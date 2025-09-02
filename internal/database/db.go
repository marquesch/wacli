package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"go.mau.fi/whatsmeow/types"
)

var (
	ServerDatabasePath    string
	WhatsmeowDatabasePath string
	db                    *sql.DB
)

const busyTimeout = 2000

func createTables() error {
	transaction, err := db.Begin()
	if err != nil {
		return fmt.Errorf("error creating transaction: %w", err)
	}

	createWhatsappTableSQL := `
	CREATE TABLE IF NOT EXISTS whatsapp_user (
		id INTEGER NOT NULL PRIMARY KEY,
		name TEXT NOT NULL,
		jid TEXT UNIQUE
	);
	`
	_, err = transaction.Exec(createWhatsappTableSQL)
	if err != nil {
		return fmt.Errorf("error creating whatsapp_user table: %w", err)
	}

	createChatTableSQL := `
	CREATE TABLE IF NOT EXISTS chat (
		id INTEGER NOT NULL PRIMARY KEY,
		name TEXT NOT NULL,
		jid TEXT UNIQUE,
		is_group BOOLEAN
	)
	`
	_, err = transaction.Exec(createChatTableSQL)
	if err != nil {
		return fmt.Errorf("error creating chat table: %w", err)
	}

	createMessageTableSQL := `
	CREATE TABLE IF NOT EXISTS message (
		id INTEGER NOT NULL PRIMARY KEY,
		chat_id INTEGER NOT NULL,
		author_id INTEGER NOT NULL,
		whatsapp_id TEXT UNIQUE NOT NULL,
	    type TEXT NOT NULL,
		media_type TEXT,
	    body TEXT,
		media_url TEXT,
		quoted_message_id INTEGER,
		timestamp TIMESTAMP,
		FOREIGN KEY (author_id) REFERENCES whatsapp_user(id),
		FOREIGN KEY (quoted_message_id) REFERENCES message(id)
	);
	`
	_, err = transaction.Exec(createMessageTableSQL)
	if err != nil {
		return fmt.Errorf("error creating message table: %w", err)
	}

	err = transaction.Commit()
	if err != nil {
		return fmt.Errorf("error committing database initial state: %w", err)
	}

	return nil
}

func init() {
	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}

	wacliLocalFolder := filepath.Join(userHomeDir, ".local", "lib", "wacli")

	err = os.MkdirAll(wacliLocalFolder, 0755)
	if err != nil {
		panic(err)
	}

	ServerDatabasePath = filepath.Join(wacliLocalFolder, "wacli.db")
	WhatsmeowDatabasePath = filepath.Join(wacliLocalFolder, "whatsmeow.db")

	databaseDSN := fmt.Sprintf("file:%s?_journal_mode=WAL&_busy_timeout=%d", ServerDatabasePath, busyTimeout)

	db, err = sql.Open("sqlite3", databaseDSN)
	if err != nil {
		panic(err)
	}

	err = createTables()
	if err != nil {
		panic(err)
	}
}

func UpsertChat(jid types.JID, name string, isGroup bool) (uint32, error) {
	var chatID uint32

	tx, err := db.Begin()
	if err != nil {
		return chatID, fmt.Errorf("error beginning transaction: %w", err)
	}

	defer tx.Rollback()

	statement := `
	INSERT INTO chat(jid, name, is_group)
	VALUES (?, ?, ?)
	ON CONFLICT(jid) DO UPDATE SET
		name = CASE
			WHEN excluded.name <> '' THEN excluded.name
			ELSE chat.name
		END
	RETURNING (id);
	`

	err = tx.QueryRow(statement, jid.String(), name, isGroup).Scan(&chatID)
	if err != nil {
		return chatID, fmt.Errorf("error scanning id: %w", err)
	}

	err = tx.Commit()
	if err != nil {
		return chatID, fmt.Errorf("error commiting chat upsert: %w", err)
	}

	return chatID, nil
}

func UpsertWhatsappUser(jid types.JID, name string) (uint32, error) {
	var whatsappUserID uint32

	tx, err := db.Begin()
	if err != nil {
		return whatsappUserID, fmt.Errorf("error beginning transaction: %w", err)
	}

	defer tx.Rollback()

	statement := `
	INSERT INTO whatsapp_user(jid, name)
	VALUES (?, ?)
	ON CONFLICT(jid) DO UPDATE SET
		name = excluded.name
	RETURNING (id);
	`

	result := tx.QueryRow(statement, jid.String(), name)
	result.Scan(&whatsappUserID)

	err = tx.Commit()
	if err != nil {
		return whatsappUserID, fmt.Errorf("error committing contact upsert: %w", err)
	}

	return whatsappUserID, nil
}

func InsertMessage(chatID uint32, authorID uint32, whatsappMsgID string, msgType string, mediaType string, body string, mediaURL string, quotedMsgID *uint32, msgTimestamp time.Time) (uint32, error) {
	var msgID uint32

	tx, err := db.Begin()
	if err != nil {
		return msgID, fmt.Errorf("error beginning transaction: %w", err)
	}

	defer tx.Rollback()

	statement := `
	INSERT INTO message(chat_id, author_id, whatsapp_id, type, media_type, body, media_url, quoted_message_id, timestamp)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	RETURNING (id);
	`

	err = tx.QueryRow(statement, chatID, authorID, whatsappMsgID, msgType, mediaType, body, mediaURL, quotedMsgID, msgTimestamp).Scan(&msgID)
	if err != nil {
		return msgID, fmt.Errorf("error scanning message_id: %w", err)
	}

	err = tx.Commit()
	if err != nil {
		return msgID, fmt.Errorf("error committing message upsert: %w", err)
	}

	return msgID, nil
}

func CheckUserInDatabase(userJID types.JID) (bool, error) {
	var userExists bool

	statement := `
	SELECT EXISTS(SELECT 1 FROM whatsapp_user WHERE jid = '?');
	`

	err := db.QueryRow(statement, userJID.String()).Scan(userExists)

	return userExists, err
}
