package email

import (
	"context"
	"fmt"
	"log"
)

type ClassifyEmailUseCase struct {
	repo         EmailRepository
	llm          LLMClassifier
	gmailService GmailService
}

func NewClassifyEmailUseCase(
	repo EmailRepository,
	llm LLMClassifier,
	gmailService GmailService,
) *ClassifyEmailUseCase {
	return &ClassifyEmailUseCase{
		repo:         repo,
		llm:          llm,
		gmailService: gmailService,
	}
}

func (uc *ClassifyEmailUseCase) Execute(ctx context.Context, gmailID string) error {
	processed, err := uc.repo.EmailAlreadyProcessed(ctx, gmailID)
	if err != nil {
		return fmt.Errorf("check processed: %w", err)
	}
	if processed {
		log.Printf("Email %s already processed, skipping", gmailID)
		return nil
	}

	emailEntity, err := uc.gmailService.FetchEmail(ctx, gmailID)
	if err != nil {
		return fmt.Errorf("fetch email: %w", err)
	}

	// Skip if empty body
	if emailEntity.Body == "" {
		log.Printf("Empty body for %s, skipping", gmailID)
		return nil
	}

	// Classify using LLM
	classification, err := uc.llm.Classify(ctx, emailEntity.Subject, emailEntity.Body)
	if err != nil {
		return fmt.Errorf("classify email: %w", err)
	}

	// Update domain entity
	emailEntity.Classify(classification.Category, classification.Label)

	// Apply label in Gmail
	if err := uc.gmailService.ApplyLabel(ctx, gmailID, emailEntity.Label); err != nil {
		log.Printf("Failed to apply label for %s: %v", gmailID, err)
	}

	// Create draft reply if needed
	if emailEntity.NeedsReply() && classification.Reply != "" {
		if err := uc.gmailService.CreateDraft(
			ctx,
			emailEntity.From,
			"Re: "+emailEntity.Subject,
			classification.Reply,
		); err != nil {
			log.Printf("Failed to create draft for %s: %v", gmailID, err)
		}
	}

	if err := uc.repo.Save(ctx, emailEntity); err != nil {
		return fmt.Errorf("save email: %w", err)
	}

	log.Printf("OK: %s – category=%s label=%s", gmailID, emailEntity.Category, emailEntity.Label)

	return nil
}
