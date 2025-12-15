package email

import "time"

type Email struct {
	ID        string
	GmailID   string
	From      string
	Subject   string
	Body      string
	Category  Category
	Label     Label
	CreatedAt time.Time
}

func NewEmail(gmailID, from, subject, body string) *Email {
	return &Email{
		GmailID:   gmailID,
		From:      from,
		Subject:   subject,
		Body:      body,
		CreatedAt: time.Now(),
	}
}

func (e *Email) Classify(category Category, label Label) {
	e.Category = category
	e.Label = label
}

func (e *Email) NeedsReply() bool {
	return e.Category == CategoryActionNeeded
}
