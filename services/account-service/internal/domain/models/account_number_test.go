package models

import (
	"fmt"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	TEST_BALANCE_PREFIX   = "40817"
	TEST_ACCOUNT_NUM_LEN  = 20
	TEST_CURRENCY_RUB     = Currency("RUB")
	TEST_CURRENCY_USD     = Currency("USD")
	TEST_CURRENCY_EUR     = Currency("EUR")
	TEST_DETERMINE_SERIAL = int64(42)
)

// testWeightedSum считает сумму (цифра*вес mod 10) с весами 7,1,3 по кругу.
func testWeightedSum(t *testing.T, number string) int {
	t.Helper()
	weights := []int{7, 1, 3}
	sum := 0
	for i, r := range number {
		require.True(t, r >= '0' && r <= '9', "non digit at position %d", i)
		sum += (int(r-'0') * weights[i%3]) % 10
	}
	return sum
}

func TestAccountNumberConstants(t *testing.T) {
	assert.Equal(t, 8, CHECK_DIGIT_POSITION)
	assert.Equal(t, 11, SERIAL_LENGTH)
	assert.Equal(t, int64(99_999_999_999), int64(MAX_SERIAL))
}

func TestNewNumberGenerator(t *testing.T) {
	tests := []struct {
		name    string
		prefix  string
		wantErr bool
	}{
		{name: "typical prefix", prefix: "40817"},
		{name: "all zeros", prefix: "00000"},
		{name: "all nines", prefix: "99999"},
		{name: "empty", prefix: "", wantErr: true},
		{name: "four digits", prefix: "4081", wantErr: true},
		{name: "six digits", prefix: "408170", wantErr: true},
		{name: "letter inside", prefix: "4081a", wantErr: true},
		{name: "leading space", prefix: " 40817", wantErr: true},
		{name: "trailing space", prefix: "40817 ", wantErr: true},
		{name: "full width digits", prefix: "４０８１７", wantErr: true},
		{name: "minus sign", prefix: "-4081", wantErr: true},
		{name: "arabic indic digits", prefix: "٤٠٨١٧", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gen, err := NewNumberGenerator(tt.prefix)
			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, gen)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, gen)
		})
	}
}

func TestNumberGenerator_Build(t *testing.T) {
	tests := []struct {
		name     string
		prefix   string
		currency Currency
		serial   int64
		want     string
		wantErr  bool
	}{
		// S = 8+0+4+7+7+4+7+0+1 = 38; 8*3=24 -> 4
		{name: "RUB serial 1", prefix: TEST_BALANCE_PREFIX, currency: TEST_CURRENCY_RUB, serial: 1, want: "40817810400000000001"},
		// S = 37+2 = 39; 9*3=27 -> 7
		{name: "RUB serial 2", prefix: TEST_BALANCE_PREFIX, currency: TEST_CURRENCY_RUB, serial: 2, want: "40817810700000000002"},
		// S = 39 -> 7
		{name: "USD serial 1", prefix: TEST_BALANCE_PREFIX, currency: TEST_CURRENCY_USD, serial: 1, want: "40817840700000000001"},
		// S = 51 -> 3
		{name: "EUR serial 1", prefix: TEST_BALANCE_PREFIX, currency: TEST_CURRENCY_EUR, serial: 1, want: "40817978300000000001"},
		// S = 37+69 = 106 -> 18 -> 8
		{name: "RUB max serial", prefix: TEST_BALANCE_PREFIX, currency: TEST_CURRENCY_RUB, serial: MAX_SERIAL, want: "40817810899999999999"},
		// S = 53 -> 27 -> 9
		{name: "other prefix RUB 12345", prefix: "40702", currency: TEST_CURRENCY_RUB, serial: 12345, want: "40702810900000012345"},
		{name: "serial zero", prefix: TEST_BALANCE_PREFIX, currency: TEST_CURRENCY_RUB, serial: 0, wantErr: true},
		{name: "serial negative", prefix: TEST_BALANCE_PREFIX, currency: TEST_CURRENCY_RUB, serial: -1, wantErr: true},
		{name: "serial above max", prefix: TEST_BALANCE_PREFIX, currency: TEST_CURRENCY_RUB, serial: MAX_SERIAL + 1, wantErr: true},
		{name: "serial min int64", prefix: TEST_BALANCE_PREFIX, currency: TEST_CURRENCY_RUB, serial: math.MinInt64, wantErr: true},
		{name: "serial max int64", prefix: TEST_BALANCE_PREFIX, currency: TEST_CURRENCY_RUB, serial: math.MaxInt64, wantErr: true},
		{name: "currency without numeric code", prefix: TEST_BALANCE_PREFIX, currency: Currency("GEL"), serial: 1, wantErr: true},
		{name: "empty currency", prefix: TEST_BALANCE_PREFIX, currency: Currency(""), serial: 1, wantErr: true},
		{name: "lower case currency", prefix: TEST_BALANCE_PREFIX, currency: Currency("rub"), serial: 1, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gen, err := NewNumberGenerator(tt.prefix)
			require.NoError(t, err)
			require.NotNil(t, gen)

			got, err := gen.Build(tt.currency, tt.serial)
			if tt.wantErr {
				require.Error(t, err)
				assert.Equal(t, "", got)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestNumberGenerator_BuildInvariants(t *testing.T) {
	tests := []struct {
		name     string
		prefix   string
		currency Currency
		serial   int64
	}{
		{name: "40817 RUB 1", prefix: "40817", currency: TEST_CURRENCY_RUB, serial: 1},
		{name: "40817 USD max", prefix: "40817", currency: TEST_CURRENCY_USD, serial: MAX_SERIAL},
		{name: "40702 EUR 12345", prefix: "40702", currency: TEST_CURRENCY_EUR, serial: 12345},
		{name: "99999 RUB 50000000000", prefix: "99999", currency: TEST_CURRENCY_RUB, serial: 50_000_000_000},
		{name: "00000 USD 7", prefix: "00000", currency: TEST_CURRENCY_USD, serial: 7},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gen, err := NewNumberGenerator(tt.prefix)
			require.NoError(t, err)
			require.NotNil(t, gen)

			number, err := gen.Build(tt.currency, tt.serial)
			require.NoError(t, err)
			require.Len(t, number, TEST_ACCOUNT_NUM_LEN)

			numericCode, ok := tt.currency.NumericCode()
			require.True(t, ok)

			assert.Equal(t, 0, testWeightedSum(t, number)%10)
			assert.Equal(t, tt.prefix, number[:5])
			assert.Equal(t, numericCode, number[5:CHECK_DIGIT_POSITION])
			assert.Equal(t, fmt.Sprintf("%0*d", SERIAL_LENGTH, tt.serial), number[CHECK_DIGIT_POSITION+1:])
		})
	}
}

func TestNumberGenerator_BuildDeterministic(t *testing.T) {
	genFirst, err := NewNumberGenerator(TEST_BALANCE_PREFIX)
	require.NoError(t, err)
	genSecond, err := NewNumberGenerator(TEST_BALANCE_PREFIX)
	require.NoError(t, err)

	first, err := genFirst.Build(TEST_CURRENCY_RUB, TEST_DETERMINE_SERIAL)
	require.NoError(t, err)
	again, err := genFirst.Build(TEST_CURRENCY_RUB, TEST_DETERMINE_SERIAL)
	require.NoError(t, err)
	other, err := genSecond.Build(TEST_CURRENCY_RUB, TEST_DETERMINE_SERIAL)
	require.NoError(t, err)

	// серийник 42: S = 37+(4*7 mod 10)+2*1 = 37+8+2 = 47; 7*3=21 -> 1
	assert.Equal(t, "40817810100000000042", first)
	assert.Equal(t, first, again)
	assert.Equal(t, first, other)
}

func TestNumberGenerator_BuildDistinctSerials(t *testing.T) {
	tests := []struct {
		name         string
		serialFirst  int64
		serialSecond int64
	}{
		{name: "1 and 2", serialFirst: 1, serialSecond: 2},
		{name: "1 and max", serialFirst: 1, serialSecond: MAX_SERIAL},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gen, err := NewNumberGenerator(TEST_BALANCE_PREFIX)
			require.NoError(t, err)

			first, err := gen.Build(TEST_CURRENCY_RUB, tt.serialFirst)
			require.NoError(t, err)
			second, err := gen.Build(TEST_CURRENCY_RUB, tt.serialSecond)
			require.NoError(t, err)

			assert.NotEqual(t, first, second)
		})
	}
}
