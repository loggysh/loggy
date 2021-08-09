package service

import (
	"fmt"
	uuid "github.com/satori/go.uuid"
	"time"
)


// Base contains common columns for all tables.
type Base struct {
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time `sql:"index"`
}

type Application struct {
	Base
	ID     string
	UserID string `gorm:"type:uuid;column:user_foreign_key;not null;"`
	Name   string
	Icon   string
}

type Device struct {
	Base
	ID      uuid.UUID `gorm:"type:uuid;primary_key;"`
	Details string
}

type Session struct {
	Base
	ID       int32
	DeviceID uuid.UUID `gorm:"type:uuid;column:device_foreign_key;not null;"`
	AppID    string    `gorm:"type:string;column:application_foreign_key;not null;"`
}

type WaitlistUser struct {
	Email string `gorm:"primary_key;"`
}

type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
	CRASH
)

type Message struct {
	ID int
	Base
	SessionID int32
	Session   Session
	Msg       string
	Timestamp time.Time
	Level     LogLevel
}

func (m *Message) String() string {
	var level string
	switch m.Level {
	case DEBUG:
		level = "DEBUG"
	case INFO:
		level = "INFO"
	case WARN:
		level = "WARN"
	case ERROR:
		level = "ERROR"
	case CRASH:
		level = "CRASH"
	default:
		level = "undefined"
	}
	return fmt.Sprintf("%v :: %d :: <%s> :: %s", m.Timestamp, m.SessionID, level, m.Msg)
}

