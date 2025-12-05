package worker

import (
	"context"
	"log"
	"sync"
	"time"

	"mailassist/internal/gmailc"
	"mailassist/internal/llm"
	"mailassist/internal/store"
)

type Job struct {
	ID      string
	Subject string
	Body    string
}

type Pool struct {
	Workers int
	Jobs    chan Job

	llm     *llm.Client
	gmail   *gmailc.Client
	db      *store.Store
	labelID string
}

func NewPool(workers int, llm *llm.Client, gmail *gmailc.Client, db *store.Store, labelID string) *Pool {
	return &Pool{
		Workers: workers,
		Jobs:    make(chan Job, 100),
		llm:     llm,
		gmail:   gmail,
		db:      db,
		labelID: labelID,
	}
}

func (p *Pool) Start() {
	go p.Run()
}

func (p *Pool) Submit(job Job) {
	p.Jobs <- job
}

func (p *Pool) Close() {
	close(p.Jobs)
}

func (p *Pool) Run() {
	var wg sync.WaitGroup
	wg.Add(p.Workers)

	for i := 0; i < p.Workers; i++ {
		go func(workerID int) {
			defer wg.Done()

			for job := range p.Jobs {

				ctx := context.Background()

				already, err := p.db.AlreadyProcessed(job.ID)
				if err != nil {
					log.Printf("[worker %d] DB check error for %s: %v\n", workerID, job.ID, err)
					continue
				}
				if already {
					log.Printf("[worker %d] Message %s already processed, skip\n", workerID, job.ID)
					continue
				}

				// Skip empty emails
				if job.Body == "" {
					log.Printf("[worker %d] Empty body for %s, skipping\n", workerID, job.ID)
					continue
				}

				result, err := p.llm.AnalyzeEmail(ctx, job.Subject, job.Body)
				if err != nil {
					log.Printf("[worker %d] LLM error for %s: %v\n", workerID, job.ID, err)
					continue
				}

				emailRecord := &store.EmailRecord{
					GmailID:  job.ID,
					Subject:  job.Subject,
					Body:     job.Body,
					Category: result.Category,
					Label:    result.Label,
					Draft:    result.Reply,
				}
				if err := p.db.SaveEmail(emailRecord); err != nil {
					log.Printf("[worker %d] DB save error for %s: %v\n", workerID, job.ID, err)
				}

				if err := p.gmail.AddLabelToMessage(job.ID, p.labelID); err != nil {
					log.Printf("[worker %d] AddLabel error for %s: %v\n", workerID, job.ID, err)
				}

				// Create draft reply for action_needed emails
				// Drafts are filtered out in ExtractMessageIDs to avoid feedback loops
				if result.Category == "action_needed" {
					email := &gmailc.Email{
						ID:      job.ID,
						Subject: job.Subject,
						Body:    job.Body,
					}
					if err := p.gmail.CreateReplyDraft(email, result.Reply); err != nil {
						log.Printf("[worker %d] Draft error for %s: %v\n", workerID, job.ID, err)
					} else {
						log.Printf("[worker %d] Draft created for %s\n", workerID, job.ID)
					}
				}

				log.Printf("[worker %d] OK: %s – category=%s label=%s\n",
					workerID, job.ID, result.Category, result.Label)

				// Throttling to avoid rate limits (200ms is safer than 120ms)
				time.Sleep(200 * time.Millisecond)
			}
		}(i)
	}

	wg.Wait()
}
