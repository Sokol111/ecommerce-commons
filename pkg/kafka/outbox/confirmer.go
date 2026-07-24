package outbox

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"
)

// ConfirmResult carries the result of a Kafka produce callback.
type ConfirmResult struct {
	ID  string
	Err error
}

type Confirmer struct {
	outboxRepository Repository
	confirmChan      <-chan ConfirmResult
	logger           *zap.Logger
	wg               sync.WaitGroup
}

func NewConfirmer(
	outboxRepository Repository,
	confirmChan chan ConfirmResult,
	logger *zap.Logger,
) *Confirmer {
	return &Confirmer{
		outboxRepository: outboxRepository,
		confirmChan:      confirmChan,
		logger:           logger,
	}
}

func (c *Confirmer) Run(ctx context.Context) error {
	defer c.wg.Wait()

	results := make([]ConfirmResult, 0, 100)

	flush := func() {
		if len(results) == 0 {
			return
		}
		copySlice := make([]ConfirmResult, len(results))
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

func (c *Confirmer) handleConfirmation(ctx context.Context, results []ConfirmResult) {
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
