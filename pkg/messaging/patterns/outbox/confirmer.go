package outbox

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"
)

// confirmResult carries the result of a Kafka produce callback.
type confirmResult struct {
	ID  string
	Err error
}

type confirmer struct {
	outboxRepository repository
	confirmChan      <-chan confirmResult
	logger           *zap.Logger
	wg               sync.WaitGroup
}

func newConfirmer(
	outboxRepository repository,
	confirmChan chan confirmResult,
	logger *zap.Logger,
) *confirmer {
	return &confirmer{
		outboxRepository: outboxRepository,
		confirmChan:      confirmChan,
		logger:           logger,
	}
}

func (c *confirmer) Run(ctx context.Context) error {
	defer c.wg.Wait()

	results := make([]confirmResult, 0, 100)

	flush := func() {
		if len(results) == 0 {
			return
		}
		copySlice := make([]confirmResult, len(results))
		copy(copySlice, results)
		c.wg.Add(1)
		go c.handleConfirmation(ctx, copySlice)
		results = results[:0]
	}

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			flush()
			return nil
		default:
		}

		select {
		case <-ctx.Done():
			flush()
			return nil
		case result := <-c.confirmChan:
			results = append(results, result)
			if len(results) == 100 {
				flush()
			}
		case <-ticker.C:
			flush()
		}
	}
}

func (c *confirmer) handleConfirmation(ctx context.Context, results []confirmResult) {
	defer c.wg.Done()

	ids := make([]string, 0, len(results))
	for _, r := range results {
		if r.Err != nil {
			c.logger.Error("kafka delivery failed - message will be retried",
				zap.String("message_id", r.ID),
				zap.Error(r.Err))
			continue
		}
		ids = append(ids, r.ID)
	}

	if len(ids) == 0 {
		return
	}

	err := c.outboxRepository.UpdateAsSentByIDs(ctx, ids)
	if err != nil {
		c.logger.Error("failed to update confirmation", zap.Error(err))
		return
	}

	c.logger.Debug("outbox sending confirmed", zap.Int("count", len(ids)))
}
