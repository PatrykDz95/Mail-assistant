package store

import (
	"database/sql"
	_ "modernc.org/sqlite"
	"time"
)

type Store struct {
	DB *sql.DB
}

func NewStore(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}

	schema := `
CREATE TABLE IF NOT EXISTS emails (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    gmail_id TEXT UNIQUE,
    from_addr TEXT,
    subject TEXT,
    body TEXT,
    category TEXT,
    label TEXT,
    draft TEXT,
    created_at INTEGER
);
`
	if _, err := db.Exec(schema); err != nil {
		return nil, err
	}

	return &Store{DB: db}, nil
}

func (s *Store) SaveEmail(rec *EmailRecord) error {
	rec.CreatedAt = time.Now().Unix()
	_, err := s.DB.Exec(
		`INSERT OR IGNORE INTO emails 
         (gmail_id, from_addr, subject, body, category, label, draft, created_at)
         VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		rec.GmailID, rec.From, rec.Subject, rec.Body,
		rec.Category, rec.Label, rec.Draft, rec.CreatedAt,
	)
	return err
}

func (s *Store) AlreadyProcessed(gmailID string) (bool, error) {
	row := s.DB.QueryRow(`SELECT 1 FROM emails WHERE gmail_id = ? LIMIT 1`, gmailID)
	var dummy int
	err := row.Scan(&dummy)
	if err == sql.ErrNoRows {
		return false, nil
	}
	return err == nil, err
}
