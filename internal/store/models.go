package store

type EmailRecord struct {
	ID        int64
	GmailID   string
	From      string
	Subject   string
	Body      string
	Category  string
	Label     string
	Draft     string
	CreatedAt int64
}
