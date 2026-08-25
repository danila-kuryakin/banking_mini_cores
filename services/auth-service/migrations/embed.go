// Package migrations хранит SQL-миграции auth-service, вшитые в бинарь,
// чтобы накат не зависел от рабочего каталога и раскладки файлов.
package migrations

import "embed"

// FS — все *.sql этого каталога. Каталог для source/iofs — ".".
//
//go:embed *.sql
var FS embed.FS
