package models

import (
	"time"
)

// ObjectInfo - то, что сервису нужно знать об объекте в хранилище.
type ObjectInfo struct {
	Key          string
	Size         int64
	ContentType  string // заявлен клиентом при заливке, доверять нельзя - на Confirm тип определяется по magic bytes
	LastModified time.Time
}
