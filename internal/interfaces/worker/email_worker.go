package worker

import (
	"context"
	"log"
	"sync"
	"time"

	"mailassist/internal/application/email"
)

type EmailJob struct {
	GmailID string
}

type Pool struct {
	workers int
	jobs    chan EmailJob
	useCase *email.ClassifyEmailUseCase
	wg      sync.WaitGroup
}

func NewPool(workers int, useCase *email.ClassifyEmailUseCase) *Pool {
	return &Pool{
		workers: workers,
		jobs:    make(chan EmailJob, 100),
		useCase: useCase,
	}
}

func (p *Pool) Start(ctx context.Context) {
	log.Printf("Worker pool started with %d workers", p.workers)

	for i := 0; i < p.workers; i++ {
		p.wg.Add(1)
		go p.worker(ctx, i)
	}
}

func (p *Pool) Submit(job EmailJob) {
	p.jobs <- job
}

func (p *Pool) Shutdown() {
	close(p.jobs)
	p.wg.Wait()
	log.Println("Worker pool shut down")
}

func (p *Pool) worker(ctx context.Context, workerID int) {
	defer p.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case job, ok := <-p.jobs:
			if !ok {
				return
			}

			if err := p.useCase.Execute(ctx, job.GmailID); err != nil {
				log.Printf("[worker %d] Error processing %s: %v", workerID, job.GmailID, err)
			}

			// Rate limiting
			time.Sleep(200 * time.Millisecond)
		}
	}
}
