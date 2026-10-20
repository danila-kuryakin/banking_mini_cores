package models

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/danila-kuryakin/banking_mini_cores/services/customer-service/internal/domain"
)

// fullProfile - полная валидная анкета с датой рождения 30 лет назад.
func fullProfile() Profile {
	now := time.Now().UTC()
	birthDate := time.Date(now.Year()-30, now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	return Profile{
		FirstName:   "Иван",
		LastName:    "Петров",
		BirthDate:   &birthDate,
		Citizenship: "RU",
		Phone:       "+79001234567",
	}
}

func dateUTC(year int, month time.Month, day int) *time.Time {
	value := time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
	return &value
}

func TestProfile_IsComplete(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(p *Profile)
		want   bool
	}{
		{name: "полная", mutate: func(p *Profile) {}, want: true},
		{name: "нет имени", mutate: func(p *Profile) { p.FirstName = "" }, want: false},
		{name: "нет фамилии", mutate: func(p *Profile) { p.LastName = "" }, want: false},
		{name: "нет даты рождения", mutate: func(p *Profile) { p.BirthDate = nil }, want: false},
		{name: "нет гражданства", mutate: func(p *Profile) { p.Citizenship = "" }, want: false},
		{name: "нет телефона", mutate: func(p *Profile) { p.Phone = "" }, want: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			profile := fullProfile()
			tc.mutate(&profile)
			assert.Equal(t, tc.want, profile.IsComplete())
		})
	}
}

func TestProfile_Normalize(t *testing.T) {
	birthDate := dateUTC(1990, time.May, 1)
	raw := Profile{
		FirstName:   "  Анна Мария\t",
		LastName:    "\nПетрова  ",
		BirthDate:   birthDate,
		Citizenship: " RU ",
		Phone:       "\t+79001234567\n",
	}
	original := raw

	got := raw.Normalize()

	assert.Equal(t, Profile{
		FirstName:   "Анна Мария",
		LastName:    "Петрова",
		BirthDate:   birthDate,
		Citizenship: "RU",
		Phone:       "+79001234567",
	}, got)
	assert.Same(t, birthDate, got.BirthDate, "birth date must stay untouched")
	assert.Equal(t, original, raw, "source profile must not change")
}

func TestProfile_Normalize_BlankNameIsIncomplete(t *testing.T) {
	profile := fullProfile()
	profile.FirstName = "   "
	require.True(t, profile.IsComplete())

	assert.False(t, profile.Normalize().IsComplete())
}

func TestProfile_Validate(t *testing.T) {
	now := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name    string
		profile Profile
		wantErr error
	}{
		{name: "полная валидная", profile: fullProfile()},
		{name: "пустая", profile: Profile{}},
		{name: "имя 100 символов", profile: Profile{FirstName: strings.Repeat("я", 100)}},
		{name: "имя 101 символ", profile: Profile{FirstName: strings.Repeat("я", 101)}, wantErr: domain.ErrFirstNameIsTooLong},
		{name: "фамилия 100 символов", profile: Profile{LastName: strings.Repeat("я", 100)}},
		{name: "фамилия 101 символ", profile: Profile{LastName: strings.Repeat("я", 101)}, wantErr: domain.ErrLastNameIsTooLong},
		{name: "гражданство 64 символа", profile: Profile{Citizenship: strings.Repeat("я", 64)}},
		{name: "гражданство 65 символов", profile: Profile{Citizenship: strings.Repeat("я", 65)}, wantErr: domain.ErrCitizenshipIsTooLong},
		{name: "телефон пустой", profile: Profile{Phone: ""}},
		{name: "телефон 7 цифр", profile: Profile{Phone: "+1234567"}},
		{name: "телефон 15 цифр", profile: Profile{Phone: "123456789012345"}},
		{name: "телефон 6 цифр", profile: Profile{Phone: "123456"}, wantErr: domain.ErrPhoneIsNotValid},
		{name: "телефон 16 цифр", profile: Profile{Phone: "1234567890123456"}, wantErr: domain.ErrPhoneIsNotValid},
		{name: "телефон с буквами", profile: Profile{Phone: "+7abc1234"}, wantErr: domain.ErrPhoneIsNotValid},
		{name: "телефон с двумя плюсами", profile: Profile{Phone: "++71234567"}, wantErr: domain.ErrPhoneIsNotValid},
		{name: "телефон с пробелами", profile: Profile{Phone: "+7 900 123 45 67"}, wantErr: domain.ErrPhoneIsNotValid},
		{name: "рождение сегодня", profile: Profile{BirthDate: dateUTC(2026, time.June, 15)}},
		{name: "рождение завтра", profile: Profile{BirthDate: dateUTC(2026, time.June, 16)}, wantErr: domain.ErrBirthDateInFuture},
		{name: "200 лет без дня", profile: Profile{BirthDate: dateUTC(1826, time.June, 16)}},
		// Дата рождения - полночь, а граница - полдень того же дня 200 лет назад.
		{name: "ровно 200 лет", profile: Profile{BirthDate: dateUTC(1826, time.June, 15)}, wantErr: domain.ErrBirthDateIsTooOld},
		{name: "больше 200 лет", profile: Profile{BirthDate: dateUTC(1826, time.June, 14)}, wantErr: domain.ErrBirthDateIsTooOld},
		{name: "длинное имя и неверный телефон", profile: Profile{FirstName: strings.Repeat("я", 101), Phone: "abc"}, wantErr: domain.ErrFirstNameIsTooLong},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.profile.Validate(now)
			if tc.wantErr == nil {
				require.NoError(t, err)
				return
			}
			require.ErrorIs(t, err, tc.wantErr)
		})
	}
}
