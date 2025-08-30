package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
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

	createContactTableSQL := `
	CREATE TABLE IF NOT EXISTS contact (
		id INTEGER NOT NULL PRIMARY KEY,
		name TEXT,
		jid TEXT UNIQUE,
		is_group BOOLEAN
	);
	`
	_, err = transaction.Exec(createContactTableSQL)
	if err != nil {
		return fmt.Errorf("error creating contact table: %w", err)
	}

	createMessageTableSQL := `
	CREATE TABLE IF NOT EXISTS message (
		id INTEGER NOT NULL PRIMARY KEY,
		contact_id INTEGER NOT NULL,
		whatsapp_id TEXT UNIQUE NOT NULL,
	    type TEXT NOT NULL,
		media_type TEXT,
	    body TEXT,
		media_url TEXT,
		quoted_message_id INTEGER,
		timestamp TIMESTAMP,
		FOREIGN KEY (contact_id) REFERENCES contact(id),
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

func UpsertContact(jid string, name string, isGroup bool) (uint32, error) {
	tx, err := db.Begin()
	if err != nil {
		return 0, fmt.Errorf("error beginning transaction: %w", err)
	}

	defer tx.Rollback()

	statement := `
	INSERT INTO contact(jid, name, is_group)
	VALUES (?, ?, ?)
	ON CONFLICT(jid) DO UPDATE SET
		name = excluded.name
	RETURNING (id);
	`

	var id uint32
	result := tx.QueryRow(statement, jid, name, isGroup)
	result.Scan(&id)

	err = tx.Commit()
	if err != nil {
		return 0, fmt.Errorf("error committing contact upsert: %w", err)
	}

	return id, nil
}

func InsertMessage(contactID uint32, whatsappMsgID string, msgType string, mediaType, body string, mediaURL string, quotedMsgID *uint32, msgTimestamp time.Time) (uint32, error) {
	tx, err := db.Begin()
	if err != nil {
		return 0, fmt.Errorf("error beginning transaction: %w", err)
	}

	defer tx.Rollback()

	var msgID uint32

	statement := `
	INSERT INTO message(contact_id, whatsapp_id, type, media_type, body, media_url, quoted_message_id, timestamp)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	RETURNING (id);
	`

	tx.QueryRow(statement, contactID, whatsappMsgID, msgType, mediaType, body, mediaURL, quotedMsgID, msgTimestamp).Scan(&msgID)
	err = tx.Commit()
	if err != nil {
		return 0, fmt.Errorf("error committing message upsert: %w", err)
	}

	return msgID, nil
}
