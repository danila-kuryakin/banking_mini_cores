package service

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/danila-kuryakin/banking_mini_cores/services/kyc-service/internal/adapters/repository"
	"github.com/danila-kuryakin/banking_mini_cores/services/kyc-service/internal/domain"
	"github.com/danila-kuryakin/banking_mini_cores/services/kyc-service/internal/domain/models"
)

// newTestSanctionsService собирает сервис на фейке санкционного списка
func newTestSanctionsService(sanctions *fakeSanctionsRepository) *SanctionsService {
	return NewSanctionsService(&repository.Repository{Application: newFakeApplicationRepository(), Sanctions: sanctions})
}

func TestSanctionsService_Add(t *testing.T) {
	errBoom := errors.New(TEST_BOOM_MESSAGE)
	birthDate := *testBirthDate()
	entry := func(firstName, lastName, source string) models.SanctionEntry {
		return models.SanctionEntry{ID: TEST_SANCTION_ID, FirstName: firstName, LastName: lastName, BirthDate: birthDate, Source: source}
	}
	stored := &models.SanctionEntry{ID: TEST_SANCTION_ID, FirstName: "Ivan", LastName: "Petrov", BirthDate: birthDate, Source: "OFAC"}

	tests := []struct {
		name      string
		input     models.SanctionEntry
		addResult *models.SanctionEntry
		addErr    error
		wantAdd   models.SanctionEntry
		want      *models.SanctionEntry
		wantErr   error
	}{
		{
			name:      "edge whitespace trimmed",
			input:     entry("  Ivan\t", "\nPetrov ", " OFAC "),
			addResult: stored,
			wantAdd:   entry("Ivan", "Petrov", "OFAC"),
			want:      stored,
		},
		{
			// фейк без addResult возвращает принятую запись
			name:    "clean fields unchanged",
			input:   entry("Ivan", "Petrov", "OFAC"),
			wantAdd: entry("Ivan", "Petrov", "OFAC"),
			want:    &models.SanctionEntry{ID: TEST_SANCTION_ID, FirstName: "Ivan", LastName: "Petrov", BirthDate: birthDate, Source: "OFAC"},
		},
		{
			name:    "inner whitespace kept",
			input:   entry(" Anna  Maria ", "\tvon  Petrov\n", "EU  list "),
			wantAdd: entry("Anna  Maria", "von  Petrov", "EU  list"),
			want:    &models.SanctionEntry{ID: TEST_SANCTION_ID, FirstName: "Anna  Maria", LastName: "von  Petrov", BirthDate: birthDate, Source: "EU  list"},
		},
		{
			name:    "whitespace only becomes empty",
			input:   entry("   ", "\t\n", " "),
			wantAdd: entry("", "", ""),
			want:    &models.SanctionEntry{ID: TEST_SANCTION_ID, BirthDate: birthDate},
		},
		{
			name:    "duplicate entry",
			input:   entry("Ivan", "Petrov", "OFAC"),
			addErr:  domain.ErrSanctionEntryExists,
			wantAdd: entry("Ivan", "Petrov", "OFAC"),
			wantErr: domain.ErrSanctionEntryExists,
		},
		{
			name:    "repository fails",
			input:   entry("Ivan", "Petrov", "OFAC"),
			addErr:  errBoom,
			wantAdd: entry("Ivan", "Petrov", "OFAC"),
			wantErr: errBoom,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sanctions := &fakeSanctionsRepository{addResult: tt.addResult, addErr: tt.addErr}
			svc := newTestSanctionsService(sanctions)

			got, err := svc.Add(t.Context(), tt.input)

			assert.Equal(t, []models.SanctionEntry{tt.wantAdd}, sanctions.addCalls)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestSanctionsService_List(t *testing.T) {
	errBoom := errors.New(TEST_BOOM_MESSAGE)
	twoEntries := []models.SanctionEntry{
		{ID: TEST_SANCTION_ID, FirstName: "Ivan", LastName: "Petrov", BirthDate: *testBirthDate(), Source: "OFAC"},
		{ID: TEST_APPLICATION_ID, FirstName: "Anna", LastName: "Sidorova", BirthDate: *testBirthDate(), Source: "EU"},
	}

	tests := []struct {
		name       string
		limit      int
		offset     int
		listResult []models.SanctionEntry
		listErr    error
		wantCall   pageCall
		want       []models.SanctionEntry
		wantErr    error
	}{
		{name: "zero limit and negative offset", limit: 0, offset: -5, wantCall: pageCall{Limit: 20, Offset: 0}},
		{name: "negative limit", limit: -1, offset: 0, wantCall: pageCall{Limit: 20, Offset: 0}},
		{name: "minimal limit", limit: 1, offset: 0, wantCall: pageCall{Limit: 1, Offset: 0}},
		{name: "max limit with offset", limit: 100, offset: 3, wantCall: pageCall{Limit: 100, Offset: 3}},
		{name: "over max limit", limit: 101, offset: 0, wantCall: pageCall{Limit: 100, Offset: 0}},
		{name: "offset minus one", limit: 10, offset: -1, wantCall: pageCall{Limit: 10, Offset: 0}},
		{name: "two entries", limit: 10, offset: 0, listResult: twoEntries, wantCall: pageCall{Limit: 10, Offset: 0}, want: twoEntries},
		{name: "empty result", limit: 10, offset: 0, listResult: []models.SanctionEntry{}, wantCall: pageCall{Limit: 10, Offset: 0}, want: []models.SanctionEntry{}},
		{name: "repository fails", limit: 10, offset: 0, listErr: errBoom, wantCall: pageCall{Limit: 10, Offset: 0}, wantErr: errBoom},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sanctions := &fakeSanctionsRepository{listResult: tt.listResult, listErr: tt.listErr}
			svc := newTestSanctionsService(sanctions)

			got, err := svc.List(t.Context(), tt.limit, tt.offset)

			assert.Equal(t, []pageCall{tt.wantCall}, sanctions.listCalls)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
