package models

import "time"

// LoginFailure - неудачная попытка входа и правила подсчёта, посчитанные сервисом.
type LoginFailure struct {
	At          time.Time // момент попытки
	WindowStart time.Time // At - window: неудачи раньше этого момента не считаются
	MaxAttempts int       // сколько неудач в окне ведут к блокировке
	LockUntil   time.Time // At + lock_duration: до какого момента блокировать
}
