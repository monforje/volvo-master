package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"volvomaster/internal/database"
	"volvomaster/internal/logger"
	"volvomaster/internal/models"
	"volvomaster/internal/services"

	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func getMongoURI() string {
	if uri := os.Getenv("MONGO_URI"); uri != "" {
		return uri
	}
	return "mongodb://localhost:27017"
}

type AdminServer struct {
	dbService *services.DatabaseService
	logger    *logger.Logger
}

func main() {
	logger := logger.New()

	if err := godotenv.Load(); err != nil {
		logger.Info("Файл .env не найден, используем системные переменные")
	}

	mongoURI := getMongoURI()
	db, err := database.Connect(mongoURI)
	if err != nil {
		logger.Fatal("Ошибка подключения к MongoDB: %v", err)
	}
	defer db.Disconnect(context.Background())

	dbService := services.NewDatabaseService(db)
	server := &AdminServer{
		dbService: dbService,
		logger:    logger,
	}

	// Статические файлы
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("cmd/admin_interface/static"))))

	// Маршруты
	http.HandleFunc("/", server.handleIndex)
	http.HandleFunc("/api/dates", server.handleDates)
	http.HandleFunc("/api/add-date", server.handleAddDate)
	http.HandleFunc("/api/delete-date", server.handleDeleteDate)
	http.HandleFunc("/api/requests", server.handleRequests)
	http.HandleFunc("/api/update-slots", server.handleUpdateSlots)

	port := ":8080"
	logger.Info("Админ-панель запущена на http://localhost%s", port)
	logger.Fatal("Ошибка запуска сервера: %v", http.ListenAndServe(port, nil))
}

func (s *AdminServer) handleIndex(w http.ResponseWriter, r *http.Request) {
	// Читаем HTML файл
	htmlBytes, err := os.ReadFile("cmd/admin_interface/static/index.html")
	if err != nil {
		http.Error(w, "Ошибка чтения HTML файла", http.StatusInternalServerError)
		return
	}

	html := string(htmlBytes)

	// Заменяем плейсхолдер на текущую дату
	html = strings.Replace(html, "{{.Today}}", time.Now().Format("2006-01-02"), 1)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(html))
}

func (s *AdminServer) handleDates(w http.ResponseWriter, r *http.Request) {
	dates, err := s.dbService.GetAvailableDates(context.Background())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(dates)
}

func (s *AdminServer) handleAddDate(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Type      string `json:"type"`
		Date      string `json:"date"`
		StartTime string `json:"startTime"`
		EndTime   string `json:"endTime"`
		Interval  int    `json:"interval"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ctx := context.Background()
	var dates []time.Time

	switch req.Type {
	case "week":
		// Добавляем следующие 7 дней
		for i := 1; i <= 7; i++ {
			dates = append(dates, time.Now().AddDate(0, 0, i))
		}
	case "month":
		// Добавляем следующие 30 дней
		for i := 1; i <= 30; i++ {
			dates = append(dates, time.Now().AddDate(0, 0, i))
		}
	case "custom":
		if req.Date == "" {
			http.Error(w, "Date is required", http.StatusBadRequest)
			return
		}
		date, err := time.Parse("2006-01-02", req.Date)
		if err != nil {
			http.Error(w, "Invalid date format", http.StatusBadRequest)
			return
		}
		dates = append(dates, date)
	}

	// Создаем временные слоты
	var timeSlots []models.TimeSlot

	if req.Type == "custom" && req.StartTime != "" && req.EndTime != "" {
		// Парсим время начала и окончания
		startHour, startMin, _ := parseTime(req.StartTime)
		endHour, endMin, _ := parseTime(req.EndTime)

		// Конвертируем в минуты для удобства
		startMinutes := startHour*60 + startMin
		endMinutes := endHour*60 + endMin
		interval := req.Interval
		if interval == 0 {
			interval = 60 // по умолчанию 1 час
		}

		// Создаем слоты с заданным интервалом
		for minutes := startMinutes; minutes < endMinutes; minutes += interval {
			hour := minutes / 60
			min := minutes % 60
			timeSlots = append(timeSlots, models.TimeSlot{
				Time:     fmt.Sprintf("%02d:%02d", hour, min),
				IsBooked: false,
			})
		}
	} else {
		// По умолчанию с 9:00 до 17:00 каждый час
		for hour := 9; hour <= 17; hour++ {
			timeSlots = append(timeSlots, models.TimeSlot{
				Time:     fmt.Sprintf("%02d:00", hour),
				IsBooked: false,
			})
		}
	}

	// Сохраняем даты
	for _, date := range dates {
		availableDate := &models.AvailableDate{
			Date:      date,
			TimeSlots: timeSlots,
			IsActive:  true,
		}

		if err := s.dbService.SaveAvailableDate(ctx, availableDate); err != nil {
			s.logger.Error("Ошибка сохранения даты %s: %v", date.Format("02.01.2006"), err)
		} else {
			s.logger.Info("Добавлена дата: %s", date.Format("02.01.2006"))
		}
	}

	w.WriteHeader(http.StatusOK)
}

func (s *AdminServer) handleDeleteDate(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		ID string `json:"id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	id, err := primitive.ObjectIDFromHex(req.ID)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	// Получаем дату и деактивируем её
	date, err := s.dbService.GetAvailableDateByID(context.Background(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	date.IsActive = false
	if err := s.dbService.SaveAvailableDate(context.Background(), date); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *AdminServer) handleRequests(w http.ResponseWriter, r *http.Request) {
	requests, err := s.dbService.GetServiceRequests(context.Background(), nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(requests)
}

func (s *AdminServer) handleUpdateSlots(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		DateID string `json:"dateId"`
		Slots  []struct {
			Index    int  `json:"index"`
			IsBooked bool `json:"is_booked"`
		} `json:"slots"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	id, err := primitive.ObjectIDFromHex(req.DateID)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	// Получаем дату
	date, err := s.dbService.GetAvailableDateByID(context.Background(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Обновляем слоты
	for _, slotUpdate := range req.Slots {
		if slotUpdate.Index < len(date.TimeSlots) {
			date.TimeSlots[slotUpdate.Index].IsBooked = slotUpdate.IsBooked
		}
	}

	// Сохраняем обновленную дату
	if err := s.dbService.SaveAvailableDate(context.Background(), date); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// parseTime парсит строку времени в формате "HH:MM"
func parseTime(timeStr string) (hour, minute int, err error) {
	_, err = fmt.Sscanf(timeStr, "%d:%d", &hour, &minute)
	return
}
