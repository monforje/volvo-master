package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// User представляет пользователя бота
type User struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID    int64              `bson:"user_id" json:"user_id"`
	ChatID    int64              `bson:"chat_id" json:"chat_id"`
	Username  string             `bson:"username,omitempty" json:"username,omitempty"`
	FirstName string             `bson:"first_name,omitempty" json:"first_name,omitempty"`
	LastName  string             `bson:"last_name,omitempty" json:"last_name,omitempty"`
	Phone     string             `bson:"phone,omitempty" json:"phone,omitempty"`
	Telegram  string             `bson:"telegram,omitempty" json:"telegram,omitempty"`
	CreatedAt time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time          `bson:"updated_at" json:"updated_at"`
}

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

// AvailableDate представляет доступную дату для записи
type AvailableDate struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Date      time.Time          `bson:"date" json:"date"`
	TimeSlots []TimeSlot         `bson:"time_slots" json:"time_slots"`
	IsActive  bool               `bson:"is_active" json:"is_active"`
	CreatedAt time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time          `bson:"updated_at" json:"updated_at"`
}

// TimeSlot представляет временной слот
type TimeSlot struct {
	Time     string `bson:"time" json:"time"`
	IsBooked bool   `bson:"is_booked" json:"is_booked"`
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

// DefaultTimeSlots стандартные временные слоты
var DefaultTimeSlots = []TimeSlot{
	{Time: "09:00", IsBooked: false},
	{Time: "10:00", IsBooked: false},
	{Time: "11:00", IsBooked: false},
	{Time: "12:00", IsBooked: false},
	{Time: "13:00", IsBooked: false},
	{Time: "14:00", IsBooked: false},
	{Time: "15:00", IsBooked: false},
	{Time: "16:00", IsBooked: false},
	{Time: "17:00", IsBooked: false},
}

// Маппинги для callback data (английские ключи для русских значений)

// ProblemAppearedKeys маппинг русских значений на английские ключи
var ProblemAppearedKeys = map[string]string{
	"Недавно":        "recently",
	"Давно":          "long_ago",
	"Только сегодня": "today",
	"Не помню":       "dont_remember",
}

// ProblemFrequencyKeys маппинг русских значений на английские ключи
var ProblemFrequencyKeys = map[string]string{
	"Постоянно":    "constantly",
	"Периодически": "periodically",
	"Только при определенных условиях": "under_conditions",
}

// EngineTypeKeys маппинг русских значений на английские ключи
var EngineTypeKeys = map[string]string{
	"Бензин":  "petrol",
	"Дизель":  "diesel",
	"Гибрид":  "hybrid",
	"Электро": "electric",
}

// Обратные маппинги (английские ключи на русские значения)

// ProblemAppearedValues маппинг английских ключей на русские значения
var ProblemAppearedValues = map[string]string{
	"recently":      "Недавно",
	"long_ago":      "Давно",
	"today":         "Только сегодня",
	"dont_remember": "Не помню",
}

// ProblemFrequencyValues маппинг английских ключей на русские значения
var ProblemFrequencyValues = map[string]string{
	"constantly":       "Постоянно",
	"periodically":     "Периодически",
	"under_conditions": "Только при определенных условиях",
}

// EngineTypeValues маппинг английских ключей на русские значения
var EngineTypeValues = map[string]string{
	"petrol":   "Бензин",
	"diesel":   "Дизель",
	"hybrid":   "Гибрид",
	"electric": "Электро",
}
