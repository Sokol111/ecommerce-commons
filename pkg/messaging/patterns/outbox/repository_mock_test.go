package outbox

import (
	"context"
	"sync"
	"time"
)

// mockRepository is a mock implementation of repository interface for testing
type mockRepository struct {
	mu                 sync.Mutex
	created            []*outboxEntity
	createErr          error
	fetchAndLockEntity *outboxEntity
	fetchAndLockErr    error
	fetchAndLockCalls  int
	fetchAndLockFunc   func(ctx context.Context) (*outboxEntity, error)
	updateAsSentIDs    []string
	updateAsSentErr    error
	updateAsSentCalls  int
}

func newMockRepository() *mockRepository {
	return &mockRepository{
		created: make([]*outboxEntity, 0),
	}
}

func newMockRepositoryWithFetchFunc(fetchFunc func(ctx context.Context) (*outboxEntity, error)) *mockRepository {
	return &mockRepository{
		created:          make([]*outboxEntity, 0),
		fetchAndLockFunc: fetchFunc,
	}
}

func (m *mockRepository) Create(ctx context.Context, payload []byte, id string, key string, topic string, headers map[string]string) (*outboxEntity, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.createErr != nil {
		return nil, m.createErr
	}

	entity := &outboxEntity{
		ID:             id,
		Payload:        payload,
		Key:            key,
		Topic:          topic,
		Headers:        headers,
		CreatedAt:      time.Now().UTC(),
		Status:         StatusProcessing,
		LockExpiresAt:  time.Now().Add(10 * time.Second),
		AttemptsToSend: 0,
	}
	m.created = append(m.created, entity)
	return entity, nil
}

func (m *mockRepository) FetchAndLock(ctx context.Context) (*outboxEntity, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.fetchAndLockCalls++

	if m.fetchAndLockFunc != nil {
		m.mu.Unlock()
		result, err := m.fetchAndLockFunc(ctx)
		m.mu.Lock()
		return result, err
	}

	if m.fetchAndLockErr != nil {
		return nil, m.fetchAndLockErr
	}

	if m.fetchAndLockEntity != nil {
		return m.fetchAndLockEntity, nil
	}

	return nil, errEntityNotFound
}

func (m *mockRepository) UpdateAsSentByIds(ctx context.Context, ids []string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.updateAsSentCalls++
	m.updateAsSentIDs = append(m.updateAsSentIDs, ids...)

	if m.updateAsSentErr != nil {
		return m.updateAsSentErr
	}

	return nil
}

func (m *mockRepository) SetFetchAndLockEntity(entity *outboxEntity) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.fetchAndLockEntity = entity
}

func (m *mockRepository) SetFetchAndLockError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.fetchAndLockErr = err
}

func (m *mockRepository) GetFetchAndLockCalls() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.fetchAndLockCalls
}

func (m *mockRepository) GetUpdateAsSentCalls() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.updateAsSentCalls
}

func (m *mockRepository) GetUpdateAsSentIDs() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]string, len(m.updateAsSentIDs))
	copy(result, m.updateAsSentIDs)
	return result
}
