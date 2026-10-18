package checks

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/danila-kuryakin/banking_mini_cores/services/kyc-service/internal/domain/interfaces"
	"github.com/danila-kuryakin/banking_mini_cores/services/kyc-service/internal/domain/models"
)

var _ interfaces.SanctionsRepository = (*fakeSanctionsRepository)(nil)

// findMatchCall - аргументы одного вызова FindMatch
type findMatchCall struct {
	firstName string
	lastName  string
	birthDate time.Time
}

// fakeSanctionsRepository - in-memory санкционный список с одной заданной записью-совпадением
type fakeSanctionsRepository struct {
	mu             sync.Mutex
	match          *models.SanctionEntry
	findMatchErr   error
	findMatchCalls []findMatchCall
}

func (f *fakeSanctionsRepository) FindMatch(_ context.Context, firstName, lastName string, birthDate time.Time) (*models.SanctionEntry, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.findMatchCalls = append(f.findMatchCalls, findMatchCall{firstName: firstName, lastName: lastName, birthDate: birthDate})
	if f.findMatchErr != nil {
		return nil, f.findMatchErr
	}
	if f.match == nil {
		return nil, nil
	}
	entry := *f.match
	return &entry, nil
}

func (f *fakeSanctionsRepository) Add(_ context.Context, _ models.SanctionEntry) (*models.SanctionEntry, error) {
	return nil, errors.New("not implemented in fake")
}

func (f *fakeSanctionsRepository) List(_ context.Context, _, _ int) ([]models.SanctionEntry, error) {
	return nil, errors.New("not implemented in fake")
}

// calls возвращает копию записанных вызовов FindMatch
func (f *fakeSanctionsRepository) calls() []findMatchCall {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]findMatchCall(nil), f.findMatchCalls...)
}
