package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ServiceRequest представляет заявку на обслуживание
type ServiceRequest struct {
	ID     primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID int64              `bson:"user_id" json:"user_id"`
	ChatID int64              `bson:"chat_id" json:"chat_id"`

	// Первый этап - контактная информация
	Name    string `bson:"name" json:"name"`
	Contact string `bson:"contact" json:"contact"`

	// Второй этап - информация об автомобиле
	VolvoModel   string `bson:"volvo_model" json:"volvo_model"`
	Year         string `bson:"year" json:"year"`
	EngineType   string `bson:"engine_type" json:"engine_type"`
	EngineVolume string `bson:"engine_volume" json:"engine_volume"`
	Mileage      string `bson:"mileage" json:"mileage"`

	// Третий этап - информация о проблеме
	Problem              string `bson:"problem" json:"problem"`
	ProblemFirstAppeared string `bson:"problem_first_appeared" json:"problem_first_appeared"`
	ProblemFrequency     string `bson:"problem_frequency" json:"problem_frequency"`
	SafetyImpact         string `bson:"safety_impact" json:"safety_impact"`
	PreviousRepairs      string `bson:"previous_repairs" json:"previous_repairs"`
	RecentChanges        string `bson:"recent_changes" json:"recent_changes"`

	// Четвертый этап - дата записи
	AppointmentDate time.Time `bson:"appointment_date" json:"appointment_date"`

	// Служебная информация
	Stage     int       `bson:"stage" json:"stage"`
	CreatedAt time.Time `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time `bson:"updated_at" json:"updated_at"`
	Status    string    `bson:"status" json:"status"` // "in_progress", "completed", "cancelled"
}

// UserSession представляет сессию пользователя
type UserSession struct {
	UserID    int64                  `bson:"user_id" json:"user_id"`
	ChatID    int64                  `bson:"chat_id" json:"chat_id"`
	Stage     int                    `bson:"stage" json:"stage"`
	RequestID primitive.ObjectID     `bson:"request_id,omitempty" json:"request_id"`
	Data      map[string]interface{} `bson:"data" json:"data"`
	UpdatedAt time.Time              `bson:"updated_at" json:"updated_at"`
}

// BotStages константы для этапов
const (
	StageStart = iota
	StagePersonalInfo
	StageCarInfo
	StageProblemInfo
	StageDateSelection
	StageCompleted
)

// EngineTypes доступные типы двигателей
var EngineTypes = []string{
	"Бензин",
	"Дизель",
	"Гибрид",
	"Электро",
}

// ProblemFrequencies варианты частоты проблемы
var ProblemFrequencies = []string{
	"Постоянно",
	"Периодически",
	"Только при определенных условиях",
}

// ProblemAppeared варианты когда впервые появилась проблема
var ProblemAppeared = []string{
	"Недавно",
	"Давно",
	"Только сегодня",
	"Не помню",
}
