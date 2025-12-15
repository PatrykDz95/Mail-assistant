package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	"mailassist/internal/domain/email"
	_ "modernc.org/sqlite"
)

type EmailRepository struct {
	db *sql.DB
}

func NewEmailRepository(dbPath string) (*EmailRepository, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	db.Exec("PRAGMA journal_mode=WAL;")
	db.Exec("PRAGMA busy_timeout = 5000;")

	schema := `
CREATE TABLE IF NOT EXISTS emails (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    gmail_id TEXT UNIQUE NOT NULL,
    from_addr TEXT,
    subject TEXT,
    body TEXT,
    category TEXT,
    label TEXT,
    created_at INTEGER
);
`
	if _, err := db.Exec(schema); err != nil {
		return nil, fmt.Errorf("create schema: %w", err)
	}

	return &EmailRepository{db: db}, nil
}

func (r *EmailRepository) GetById(ctx context.Context, gmailID string) (*email.Email, error) {
	var e email.Email
	var category, label string

	err := r.db.QueryRowContext(ctx,
		`SELECT gmail_id, from_addr, subject, body, category, label 
		 FROM emails WHERE gmail_id = ?`,
		gmailID,
	).Scan(&e.GmailID, &e.From, &e.Subject, &e.Body, &category, &label)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("email not found: %s", gmailID)
	}
	if err != nil {
		return nil, fmt.Errorf("query email: %w", err)
	}

	e.Category = email.Category(category)
	e.Label = email.Label(label)

	return &e, nil
}

func (r *EmailRepository) Save(ctx context.Context, e *email.Email) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT OR REPLACE INTO emails 
         (gmail_id, from_addr, subject, body, category, label, created_at)
         VALUES (?, ?, ?, ?, ?, ?, ?)`,
		e.GmailID, e.From, e.Subject, e.Body,
		string(e.Category), string(e.Label), e.CreatedAt.Unix(),
	)

	if err != nil {
		return fmt.Errorf("save email: %w", err)
	}

	return nil
}

func (r *EmailRepository) EmailAlreadyProcessed(ctx context.Context, gmailID string) (bool, error) {
	var exists int
	err := r.db.QueryRowContext(ctx,
		`SELECT 1 FROM emails WHERE gmail_id = ? LIMIT 1`,
		gmailID,
	).Scan(&exists)

	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("check processed: %w", err)
	}

	return true, nil
}

func (r *EmailRepository) Close() error {
	return r.db.Close()
}
