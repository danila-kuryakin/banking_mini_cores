package filerepo

import (
	"mime"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/danila-kuryakin/banking_mini_cores/services/filestore-service/internal/domain"
	"github.com/danila-kuryakin/banking_mini_cores/services/filestore-service/internal/domain/models"
)

const (
	TEST_FILENAME_PLAIN     = "passport.pdf"
	TEST_FILENAME_SPACE     = "my file.pdf"
	TEST_FILENAME_QUOTE     = `a"b.pdf`
	TEST_FILENAME_SEMICOLON = "a;b.pdf"
	TEST_FILENAME_CYRILLIC  = "паспорт.pdf"
	TEST_FILENAME_NEWLINE   = "a\nb.pdf"
	TEST_FILENAME_CRLF      = "a\r\nb.pdf"
	TEST_FILENAME_EMOJI     = "😀.png"

	TEST_CONTENT_TYPE_PDF          = "application/pdf"
	TEST_CONTENT_TYPE_PNG          = "image/png"
	TEST_CONTENT_TYPE_WITH_CHARSET = "text/plain; charset=utf-8"
)

func TestResponseParams(t *testing.T) {
	tests := []struct {
		name string
		opts models.DownloadOptions
		want url.Values
	}{
		{
			name: "only filename plain",
			opts: models.DownloadOptions{Filename: TEST_FILENAME_PLAIN},
			want: url.Values{domain.RESPONSE_CONTENT_DISPOSITION: {`attachment; filename=passport.pdf`}},
		},
		{
			name: "only content type",
			opts: models.DownloadOptions{ContentType: TEST_CONTENT_TYPE_PDF},
			want: url.Values{domain.RESPONSE_CONTENT_TYPE: {TEST_CONTENT_TYPE_PDF}},
		},
		{
			name: "cyrillic filename and content type",
			opts: models.DownloadOptions{Filename: TEST_FILENAME_CYRILLIC, ContentType: TEST_CONTENT_TYPE_PDF},
			want: url.Values{
				domain.RESPONSE_CONTENT_DISPOSITION: {`attachment; filename*=utf-8''%D0%BF%D0%B0%D1%81%D0%BF%D0%BE%D1%80%D1%82.pdf`},
				domain.RESPONSE_CONTENT_TYPE:        {TEST_CONTENT_TYPE_PDF},
			},
		},
		{
			name: "filename with space and content type",
			opts: models.DownloadOptions{Filename: TEST_FILENAME_SPACE, ContentType: TEST_CONTENT_TYPE_PNG},
			want: url.Values{
				domain.RESPONSE_CONTENT_DISPOSITION: {`attachment; filename="my file.pdf"`},
				domain.RESPONSE_CONTENT_TYPE:        {TEST_CONTENT_TYPE_PNG},
			},
		},
		{
			name: "filename with quote",
			opts: models.DownloadOptions{Filename: TEST_FILENAME_QUOTE},
			want: url.Values{domain.RESPONSE_CONTENT_DISPOSITION: {`attachment; filename="a\"b.pdf"`}},
		},
		{
			name: "filename with semicolon",
			opts: models.DownloadOptions{Filename: TEST_FILENAME_SEMICOLON},
			want: url.Values{domain.RESPONSE_CONTENT_DISPOSITION: {`attachment; filename="a;b.pdf"`}},
		},
		{
			name: "filename with newline",
			opts: models.DownloadOptions{Filename: TEST_FILENAME_NEWLINE},
			want: url.Values{domain.RESPONSE_CONTENT_DISPOSITION: {`attachment; filename*=utf-8''a%0Ab.pdf`}},
		},
		{
			name: "filename with emoji",
			opts: models.DownloadOptions{Filename: TEST_FILENAME_EMOJI},
			want: url.Values{domain.RESPONSE_CONTENT_DISPOSITION: {`attachment; filename*=utf-8''%F0%9F%98%80.png`}},
		},
		{
			name: "content type with parameter passed unchanged",
			opts: models.DownloadOptions{ContentType: TEST_CONTENT_TYPE_WITH_CHARSET},
			want: url.Values{domain.RESPONSE_CONTENT_TYPE: {TEST_CONTENT_TYPE_WITH_CHARSET}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := responseParams(tt.opts)

			assert.Equal(t, tt.want, got)
		})
	}
}

// R1: пустые опции дают пустой результат, nil или пустая мапа - не важно
func TestResponseParams_Empty(t *testing.T) {
	got := responseParams(models.DownloadOptions{})

	assert.Len(t, got, 0)
}

// R6: значение disposition разбирается обратно в attachment с исходным именем и без сырых CR/LF
func TestResponseParams_DispositionRoundTrip(t *testing.T) {
	tests := []struct {
		name     string
		filename string
	}{
		{name: "plain", filename: TEST_FILENAME_PLAIN},
		{name: "space", filename: TEST_FILENAME_SPACE},
		{name: "quote", filename: TEST_FILENAME_QUOTE},
		{name: "semicolon", filename: TEST_FILENAME_SEMICOLON},
		{name: "cyrillic", filename: TEST_FILENAME_CYRILLIC},
		{name: "newline", filename: TEST_FILENAME_NEWLINE},
		{name: "crlf", filename: TEST_FILENAME_CRLF},
		{name: "emoji", filename: TEST_FILENAME_EMOJI},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := responseParams(models.DownloadOptions{Filename: tt.filename})

			values := got[domain.RESPONSE_CONTENT_DISPOSITION]
			require.Len(t, values, 1)
			disposition := values[0]

			mediaType, params, err := mime.ParseMediaType(disposition)
			require.NoError(t, err)
			assert.Equal(t, domain.CONTENT_DISPOSITION_ATTACHMENT, mediaType)
			assert.Equal(t, map[string]string{"filename": tt.filename}, params)
			assert.NotContains(t, disposition, "\r")
			assert.NotContains(t, disposition, "\n")
		})
	}
}

// R4, R5: множество ключей определяется только тем, какие поля заполнены
func TestResponseParams_KeySet(t *testing.T) {
	tests := []struct {
		name     string
		opts     models.DownloadOptions
		wantKeys []string
	}{
		{
			name:     "both empty",
			opts:     models.DownloadOptions{},
			wantKeys: []string{},
		},
		{
			name:     "only filename",
			opts:     models.DownloadOptions{Filename: TEST_FILENAME_CYRILLIC},
			wantKeys: []string{domain.RESPONSE_CONTENT_DISPOSITION},
		},
		{
			name:     "only content type",
			opts:     models.DownloadOptions{ContentType: TEST_CONTENT_TYPE_PNG},
			wantKeys: []string{domain.RESPONSE_CONTENT_TYPE},
		},
		{
			name:     "both set",
			opts:     models.DownloadOptions{Filename: TEST_FILENAME_CYRILLIC, ContentType: TEST_CONTENT_TYPE_PNG},
			wantKeys: []string{domain.RESPONSE_CONTENT_DISPOSITION, domain.RESPONSE_CONTENT_TYPE},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := responseParams(tt.opts)

			gotKeys := make([]string, 0, len(got))
			for key := range got {
				gotKeys = append(gotKeys, key)
			}
			assert.ElementsMatch(t, tt.wantKeys, gotKeys)
		})
	}
}

// п.7: повторный вызов с теми же опциями даёт равный результат
func TestResponseParams_Repeatable(t *testing.T) {
	opts := models.DownloadOptions{Filename: TEST_FILENAME_CYRILLIC, ContentType: TEST_CONTENT_TYPE_PDF}

	first := responseParams(opts)
	second := responseParams(opts)

	assert.Equal(t, first, second)
}
