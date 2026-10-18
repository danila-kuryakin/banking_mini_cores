package worker

import (
	"bytes"
	"context"
	"sync"

	"github.com/danila-kuryakin/banking_mini_cores/services/ledger-service/internal/domain/interfaces"
	"github.com/danila-kuryakin/banking_mini_cores/services/ledger-service/internal/domain/models"
)

// relayResponse - заранее заданный ответ фейка на один вызов Relay
type relayResponse struct {
	sent int
	err  error
}

// fakeOutboxRepository отдаёт ответы из очереди и записывает вызовы
type fakeOutboxRepository struct {
	mu          sync.Mutex
	responses   []relayResponse
	publishMsgs []models.OutboxMessage
	limits      []int
	publishErrs []error
	// propagatePublishErr - вернуть ошибку publish как ошибку Relay
	propagatePublishErr bool
}

var _ interfaces.OutboxRepository = (*fakeOutboxRepository)(nil)

func (f *fakeOutboxRepository) Relay(ctx context.Context, limit int, publish func(ctx context.Context, msgs []models.OutboxMessage) error) (int, error) {
	f.mu.Lock()
	f.limits = append(f.limits, limit)
	msgs := f.publishMsgs
	f.mu.Unlock()

	// publish вызывается без мьютекса, чтобы не держать его во время чужого кода
	if msgs != nil {
		publishErr := publish(ctx, msgs)
		f.mu.Lock()
		f.publishErrs = append(f.publishErrs, publishErr)
		f.mu.Unlock()
		if f.propagatePublishErr && publishErr != nil {
			return 0, publishErr
		}
	}

	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.responses) == 0 {
		return 0, nil
	}
	resp := f.responses[0]
	f.responses = f.responses[1:]
	return resp.sent, resp.err
}

// gotLimits возвращает копию limit всех вызовов Relay
func (f *fakeOutboxRepository) gotLimits() []int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]int(nil), f.limits...)
}

// calls возвращает число вызовов Relay
func (f *fakeOutboxRepository) calls() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.limits)
}

// gotPublishErrs возвращает ошибки, которые вернул publish
func (f *fakeOutboxRepository) gotPublishErrs() []error {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]error(nil), f.publishErrs...)
}

// fakeEventPublisher записывает полученные пачки
type fakeEventPublisher struct {
	mu         sync.Mutex
	batches    [][]models.OutboxMessage
	publishErr error
}

var _ interfaces.EventPublisher = (*fakeEventPublisher)(nil)

func (f *fakeEventPublisher) Publish(_ context.Context, msgs []models.OutboxMessage) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.batches = append(f.batches, msgs)
	return f.publishErr
}

// gotBatches возвращает копию полученных пачек
func (f *fakeEventPublisher) gotBatches() [][]models.OutboxMessage {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([][]models.OutboxMessage(nil), f.batches...)
}

// syncBuffer - буфер лога, безопасный для записи из горутины
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}
