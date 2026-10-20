package worker

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/danila-kuryakin/banking_mini_cores/services/notification-service/internal/domain/interfaces"
	"github.com/danila-kuryakin/banking_mini_cores/services/notification-service/internal/domain/models"
)

// Имена операций в журнале вызовов
const (
	TEST_OP_FETCH      = "fetch"
	TEST_OP_FETCH_WAIT = "fetch_wait"
	TEST_OP_HANDLE     = "handle"
	TEST_OP_SEND       = "send"
	TEST_OP_COMMIT     = "commit"
)

// Поля тестовых сообщений
const (
	TEST_TOPIC     = "ledger.events"
	TEST_PARTITION = 2
)

// call - одна запись журнала: операция, сообщение, результат и момент вызова от старта
type call struct {
	Op  string
	Msg models.Message
	Err error
	At  time.Duration
}

// callLog - общий журнал вызовов всех фейков одного кейса
type callLog struct {
	mu     sync.Mutex
	start  time.Time
	calls  []call
	stopAt int
	stop   chan struct{}
}

// newCallLog создаёт журнал; stop закрывается, когда в журнале stopAt записей (0 - никогда)
func newCallLog(stopAt int) *callLog {
	return &callLog{start: time.Now(), stopAt: stopAt, stop: make(chan struct{})}
}

func (l *callLog) add(op string, msg models.Message, err error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.calls = append(l.calls, call{Op: op, Msg: msg, Err: err, At: time.Since(l.start)})
	if len(l.calls) == l.stopAt {
		close(l.stop)
	}
}

func (l *callLog) snapshot() []call {
	l.mu.Lock()
	defer l.mu.Unlock()
	out := make([]call, len(l.calls))
	copy(out, l.calls)
	return out
}

func (l *callLog) elapsed() time.Duration {
	return time.Since(l.start)
}

// fetchResult - очередной ответ Fetch по сценарию
type fetchResult struct {
	msg models.Message
	err error
}

// fakeEventReader отдаёт сообщения по сценарию, а когда он кончился - блокируется до отмены ctx
type fakeEventReader struct {
	mu              sync.Mutex
	log             *callLog
	fetches         []fetchResult
	fetchAlwaysErr  error
	commitErrs      []error
	commitAlwaysErr error
	idle            chan struct{}
	idleOnce        sync.Once
}

var _ interfaces.EventReader = (*fakeEventReader)(nil)

func newFakeEventReader(log *callLog, fetches []fetchResult, fetchAlwaysErr error, commitErrs []error, commitAlwaysErr error) *fakeEventReader {
	return &fakeEventReader{
		log:             log,
		fetches:         append([]fetchResult(nil), fetches...),
		fetchAlwaysErr:  fetchAlwaysErr,
		commitErrs:      append([]error(nil), commitErrs...),
		commitAlwaysErr: commitAlwaysErr,
		idle:            make(chan struct{}),
	}
}

func (r *fakeEventReader) Fetch(ctx context.Context) (models.Message, error) {
	if err := ctx.Err(); err != nil {
		r.log.add(TEST_OP_FETCH, models.Message{}, err)
		return models.Message{}, err
	}
	r.mu.Lock()
	if r.fetchAlwaysErr != nil {
		err := r.fetchAlwaysErr
		r.mu.Unlock()
		r.log.add(TEST_OP_FETCH, models.Message{}, err)
		return models.Message{}, err
	}
	if len(r.fetches) > 0 {
		res := r.fetches[0]
		r.fetches = r.fetches[1:]
		r.mu.Unlock()
		r.log.add(TEST_OP_FETCH, res.msg, res.err)
		return res.msg, res.err
	}
	r.mu.Unlock()
	// сценарий кончился: сообщений нет, ждём отмены
	r.log.add(TEST_OP_FETCH_WAIT, models.Message{}, nil)
	r.idleOnce.Do(func() { close(r.idle) })
	<-ctx.Done()
	return models.Message{}, ctx.Err()
}

func (r *fakeEventReader) Commit(_ context.Context, msg models.Message) error {
	r.mu.Lock()
	err := r.commitAlwaysErr
	if err == nil && len(r.commitErrs) > 0 {
		err = r.commitErrs[0]
		r.commitErrs = r.commitErrs[1:]
	}
	r.mu.Unlock()
	r.log.add(TEST_OP_COMMIT, msg, err)
	return err
}

// fakeDeadLetterWriter записывает отложенные сообщения и причины, ошибки отдаёт по очереди
type fakeDeadLetterWriter struct {
	mu            sync.Mutex
	log           *callLog
	sendErrs      []error
	sendAlwaysErr error
	reasons       []error
}

var _ interfaces.DeadLetterWriter = (*fakeDeadLetterWriter)(nil)

func newFakeDeadLetterWriter(log *callLog, sendErrs []error, sendAlwaysErr error) *fakeDeadLetterWriter {
	return &fakeDeadLetterWriter{
		log:           log,
		sendErrs:      append([]error(nil), sendErrs...),
		sendAlwaysErr: sendAlwaysErr,
	}
}

func (d *fakeDeadLetterWriter) Send(_ context.Context, msg models.Message, reason error) error {
	d.mu.Lock()
	d.reasons = append(d.reasons, reason)
	err := d.sendAlwaysErr
	if err == nil && len(d.sendErrs) > 0 {
		err = d.sendErrs[0]
		d.sendErrs = d.sendErrs[1:]
	}
	d.mu.Unlock()
	d.log.add(TEST_OP_SEND, msg, err)
	return err
}

func (d *fakeDeadLetterWriter) reasonsSnapshot() []error {
	d.mu.Lock()
	defer d.mu.Unlock()
	return append([]error(nil), d.reasons...)
}

// fakeHandler отдаёт результаты обработки по Offset сообщения; по исчерпании - nil
type fakeHandler struct {
	mu      sync.Mutex
	log     *callLog
	results map[int64][]error
	onCall  func()
}

func newFakeHandler(log *callLog, results map[int64][]error) *fakeHandler {
	copied := make(map[int64][]error, len(results))
	for offset, errs := range results {
		copied[offset] = append([]error(nil), errs...)
	}
	return &fakeHandler{log: log, results: copied}
}

func (h *fakeHandler) handle(_ context.Context, msg models.Message) error {
	h.mu.Lock()
	var err error
	if errs := h.results[msg.Offset]; len(errs) > 0 {
		err = errs[0]
		h.results[msg.Offset] = errs[1:]
	}
	hook := h.onCall
	h.mu.Unlock()
	if hook != nil {
		hook()
	}
	h.log.add(TEST_OP_HANDLE, msg, err)
	return err
}

// testMessage собирает сообщение со всеми заполненными полями
func testMessage(offset int64) models.Message {
	return models.Message{
		Topic:     TEST_TOPIC,
		Partition: TEST_PARTITION,
		Offset:    offset,
		Key:       []byte(fmt.Sprintf("account-%d", offset)),
		Value:     []byte(fmt.Sprintf(`{"event_id":"%d"}`, offset)),
		Headers: []models.Header{
			{Key: "event_type", Value: []byte("transaction_completed")},
			{Key: "request_id", Value: []byte(fmt.Sprintf("req-%d", offset))},
		},
	}
}
