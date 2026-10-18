package service

import (
	"bytes"
	"strings"
	"testing"

	"github.com/danila-kuryakin/banking_mini_cores/services/filestore-service/internal/domain/models"
	"github.com/stretchr/testify/assert"
)

func TestDetectContentType(t *testing.T) {
	// 600 байт хвоста - чтобы вход был длиннее окна сниффинга в 512 байт
	tail := make([]byte, 600)

	tests := []struct {
		name string
		head []byte
		want string
	}{
		{name: "pdf", head: []byte("%PDF-1.4\n%test\n"), want: "application/pdf"},
		{name: "png", head: []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR"), want: "image/png"},
		{name: "jpeg", head: []byte("\xff\xd8\xff\xe0\x00\x10JFIF\x00"), want: "image/jpeg"},
		{name: "pdf signature only", head: []byte("%PDF-"), want: "application/pdf"},
		{name: "png signature only", head: []byte("\x89PNG\r\n\x1a\n"), want: "image/png"},
		{name: "jpeg signature only", head: []byte("\xff\xd8\xff"), want: "image/jpeg"},
		{name: "pdf longer than 512 bytes", head: append([]byte("%PDF-1.4"), tail...), want: "application/pdf"},
		{name: "jpeg longer than 512 bytes", head: append([]byte("\xff\xd8\xff"), tail...), want: "image/jpeg"},
		{name: "gif", head: []byte("GIF89a"), want: "image/gif"},
		{name: "plain text without charset", head: []byte("hello world"), want: "text/plain"},
		{name: "html without charset", head: []byte("<html><body>"), want: "text/html"},
		{name: "unknown binary", head: []byte{0x00, 0x01, 0x02, 0x03}, want: "application/octet-stream"},
		{name: "empty slice", head: []byte{}, want: "text/plain"},
		{name: "nil", head: nil, want: "text/plain"},
		{name: "pdf signature after 512 bytes", head: append(make([]byte, 512), []byte("%PDF-1.4")...), want: "application/octet-stream"},
		{name: "incomplete pdf signature", head: []byte("%PDF"), want: "text/plain"},
		{name: "pdf signature after space", head: []byte(" %PDF-1.4"), want: "text/plain"},
		{name: "pdf signature after newline", head: []byte("\n%PDF-1.4"), want: "text/plain"},
		{name: "lowercase pdf signature", head: []byte("%pdf-1.4"), want: "text/plain"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, detectContentType(tt.head))
		})
	}
}

func TestDetectContentType_CleanMediaTypeInvariant(t *testing.T) {
	tests := []struct {
		name string
		head []byte
	}{
		{name: "pdf", head: []byte("%PDF-1.4\n%test\n")},
		{name: "png", head: []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR")},
		{name: "jpeg", head: []byte("\xff\xd8\xff\xe0\x00\x10JFIF\x00")},
		{name: "gif", head: []byte("GIF89a")},
		{name: "plain text", head: []byte("hello world")},
		{name: "html", head: []byte("<html><body>")},
		{name: "unknown binary", head: []byte{0x00, 0x01, 0x02, 0x03}},
		{name: "empty slice", head: []byte{}},
		{name: "nil", head: nil},
		{name: "pdf signature after 512 bytes", head: append(bytes.Repeat([]byte{0x00}, 512), []byte("%PDF-1.4")...)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detectContentType(tt.head)

			assert.NotContains(t, got, ";")
			assert.NotContains(t, got, "charset")
			assert.Equal(t, strings.TrimSpace(got), got)
			assert.Equal(t, strings.ToLower(got), got)
		})
	}
}

func TestDetectContentType_AllowedIffSignatureInvariant(t *testing.T) {
	tests := []struct {
		name         string
		head         []byte
		hasSignature bool
	}{
		{name: "pdf", head: []byte("%PDF-1.4\n%test\n"), hasSignature: true},
		{name: "png", head: []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR"), hasSignature: true},
		{name: "jpeg", head: []byte("\xff\xd8\xff\xe0\x00\x10JFIF\x00"), hasSignature: true},
		{name: "gif", head: []byte("GIF89a"), hasSignature: false},
		{name: "plain text", head: []byte("hello world"), hasSignature: false},
		{name: "html", head: []byte("<html><body>"), hasSignature: false},
		{name: "unknown binary", head: []byte{0x00, 0x01, 0x02, 0x03}, hasSignature: false},
		{name: "empty slice", head: []byte{}, hasSignature: false},
		{name: "nil", head: nil, hasSignature: false},
		{name: "pdf signature after 512 bytes", head: append(bytes.Repeat([]byte{0x00}, 512), []byte("%PDF-1.4")...), hasSignature: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detectContentType(tt.head)

			_, allowed := models.ALLOWED_CONTENT_TYPES[got]
			assert.Equal(t, tt.hasSignature, allowed, "content type %q", got)
		})
	}
}
