package email

import (
	"context"
	"mailassist/internal/domain/email"
)

type LLMClassifier interface {
	Classify(ctx context.Context, subject, body string) (*email.Classification, error)
}

type EmailRepository interface {
	GetById(ctx context.Context, gmailID string) (*email.Email, error)
	Save(ctx context.Context, e *email.Email) error
	EmailAlreadyProcessed(ctx context.Context, gmailID string) (bool, error)
}

type GmailService interface {
	FetchEmail(ctx context.Context, messageID string) (*email.Email, error)
	ApplyLabel(ctx context.Context, messageID string, label email.Label) error
	CreateDraft(ctx context.Context, emailID, recipient, subject, body string) error
}
