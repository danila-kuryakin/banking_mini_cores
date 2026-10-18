package worker

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/danila-kuryakin/banking_mini_cores/services/antifraud-service/internal/domain/interfaces"
	"github.com/danila-kuryakin/banking_mini_cores/services/antifraud-service/internal/domain/models"
)

// errDB - ошибка зависимости для сценариев повторов
var errDB = errors.New("db failure")

// fetchResult - заданный ответ Fetch
type fetchResult struct {
	msg models.Message
	err error
}

// addResult - заданный ответ AddTransfer
type addResult struct {
	added bool
	err   error
}

// fakeReader - очередь ответов Fetch и запись коммитов
type fakeReader struct {
	mu              sync.Mutex
	fetchQueue      []fetchResult
	fetchTimes      []time.Time
	commitErrs      []error
	commitErrAlways error
	commits         []models.Message
	commitTimes     []time.Time
	commitAddCounts []int
	totals          *fakeTotals
}

var _ interfaces.EventReader = (*fakeReader)(nil)

func (r *fakeReader) Fetch(ctx context.Context) (models.Message, error) {
	r.mu.Lock()
	r.fetchTimes = append(r.fetchTimes, time.Now())
	if err := ctx.Err(); err != nil {
		r.mu.Unlock()
		return models.Message{}, err
	}
	if len(r.fetchQueue) > 0 {
		res := r.fetchQueue[0]
		r.fetchQueue = r.fetchQueue[1:]
		r.mu.Unlock()
		return res.msg, res.err
	}
	r.mu.Unlock()
	<-ctx.Done()
	return models.Message{}, ctx.Err()
}

func (r *fakeReader) Commit(_ context.Context, msg models.Message) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.commits = append(r.commits, msg)
	r.commitTimes = append(r.commitTimes, time.Now())
	r.commitAddCounts = append(r.commitAddCounts, r.totals.callCount())
	if r.commitErrAlways != nil {
		return r.commitErrAlways
	}
	if len(r.commitErrs) > 0 {
		err := r.commitErrs[0]
		r.commitErrs = r.commitErrs[1:]
		return err
	}
	return nil
}

// Commits - все попытки коммита, включая неудачные
func (r *fakeReader) Commits() []models.Message {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]models.Message(nil), r.commits...)
}

// CommitAddCounts - сколько было вызовов AddTransfer к моменту каждого коммита
func (r *fakeReader) CommitAddCounts() []int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]int(nil), r.commitAddCounts...)
}

// CommitTimes - виртуальное время каждого коммита
func (r *fakeReader) CommitTimes() []time.Time {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]time.Time(nil), r.commitTimes...)
}

// FetchTimes - виртуальное время каждого вызова Fetch
func (r *fakeReader) FetchTimes() []time.Time {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]time.Time(nil), r.fetchTimes...)
}

// fakeTotals - очередь ответов AddTransfer и запись аргументов
type fakeTotals struct {
	mu        sync.Mutex
	results   []addResult
	errAlways error
	calls     []models.CompletedTransfer
	callTimes []time.Time
}

var _ interfaces.TotalsRepository = (*fakeTotals)(nil)

func (f *fakeTotals) AddTransfer(_ context.Context, transfer models.CompletedTransfer) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, transfer)
	f.callTimes = append(f.callTimes, time.Now())
	if f.errAlways != nil {
		return false, f.errAlways
	}
	if len(f.results) > 0 {
		res := f.results[0]
		f.results = f.results[1:]
		return res.added, res.err
	}
	return true, nil
}

func (f *fakeTotals) callCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.calls)
}

// Calls - аргументы всех вызовов AddTransfer
func (f *fakeTotals) Calls() []models.CompletedTransfer {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]models.CompletedTransfer(nil), f.calls...)
}

// CallTimes - виртуальное время каждого вызова AddTransfer
func (f *fakeTotals) CallTimes() []time.Time {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]time.Time(nil), f.callTimes...)
}

// newTestLogger - логгер без вывода
func newTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// pauses - промежутки между соседними отметками времени, nil для меньше двух отметок
func pauses(times []time.Time) []time.Duration {
	var out []time.Duration
	for i := 1; i < len(times); i++ {
		out = append(out, times[i].Sub(times[i-1]))
	}
	return out
}

// runWorker - запускает Run внутри пузыря synctest, ждёт advance виртуального времени и полной блокировки, затем отменяет context
func runWorker(t *testing.T, reader *fakeReader, totals *fakeTotals, minBackoff, maxBackoff, advance time.Duration) {
	t.Helper()
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	w := NewDailyTotals(reader, totals, newTestLogger(), minBackoff, maxBackoff)
	done := make(chan struct{})
	go func() {
		defer close(done)
		w.Run(ctx)
	}()
	time.Sleep(advance)
	synctest.Wait()
	cancel()
	<-done
}
