package main

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"os"
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
	tmpl := `<!DOCTYPE html>
<html>
<head>
    <title>Volvo Service - Админ панель</title>
    <meta charset="utf-8">
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .container { max-width: 1200px; margin: 0 auto; }
        .section { margin-bottom: 30px; padding: 20px; border: 1px solid #ddd; border-radius: 5px; }
        .date-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(200px, 1fr)); gap: 10px; margin: 20px 0; }
        .date-card { padding: 15px; border: 1px solid #ccc; border-radius: 5px; background: #f9f9f9; }
        .date-card.active { background: #e8f5e8; border-color: #4caf50; }
        .time-slots { margin-top: 10px; font-size: 12px; }
        .btn { padding: 8px 16px; margin: 5px; border: none; border-radius: 3px; cursor: pointer; }
        .btn-primary { background: #007bff; color: white; }
        .btn-danger { background: #dc3545; color: white; }
        .btn-success { background: #28a745; color: white; }
        .form-group { margin: 10px 0; }
        .form-group label { display: block; margin-bottom: 5px; }
        .form-group input, .form-group select { width: 100%; padding: 8px; border: 1px solid #ddd; border-radius: 3px; }
        .requests-table { width: 100%; border-collapse: collapse; margin-top: 10px; }
        .requests-table th, .requests-table td { padding: 8px; border: 1px solid #ddd; text-align: left; }
        .requests-table th { background: #f5f5f5; }
        .time-slots-input { display: flex; gap: 10px; align-items: center; margin: 10px 0; }
        .time-slots-input input { width: 80px; }
        .time-slots-input select { width: 120px; }
        .date-card.selected { background: #ffeb3b !important; border-color: #f57f17; }
        .bulk-actions { margin: 20px 0; padding: 15px; background: #f8f9fa; border-radius: 5px; }
        .checkbox { margin-right: 10px; }
        .time-slot { display: inline-block; margin: 2px; padding: 4px 8px; border: 1px solid #ddd; border-radius: 3px; font-size: 12px; }
        .time-slot.booked { background: #ffcdd2; color: #c62828; }
        .time-slot.available { background: #c8e6c9; color: #2e7d32; }
        .edit-slots { margin-top: 10px; }
        .edit-slots input { width: 60px; margin: 2px; }
        .modal { display: none; position: fixed; z-index: 1000; left: 0; top: 0; width: 100%; height: 100%; background-color: rgba(0,0,0,0.4); }
        .modal-content { background-color: #fefefe; margin: 15% auto; padding: 20px; border: 1px solid #888; width: 80%; max-width: 500px; border-radius: 5px; }
        .close { color: #aaa; float: right; font-size: 28px; font-weight: bold; cursor: pointer; }
        .close:hover { color: black; }
    </style>
</head>
<body>
    <div class="container">
        <h1>🚗 Volvo Service - Админ панель</h1>
        
        <div class="section">
            <h2>📅 Управление датами</h2>
            
            <div class="form-group">
                <label>Быстрое добавление дат:</label>
                <button class="btn btn-success" onclick="addNextWeek()">Добавить неделю (следующие 7 дней)</button>
                <button class="btn btn-success" onclick="addNextMonth()">Добавить месяц (следующие 30 дней)</button>
            </div>
            
            <div class="form-group">
                <label>Добавить конкретную дату:</label>
                <input type="date" id="customDate" min="{{.Today}}">
                <div class="time-slots-input">
                    <label>Время начала:</label>
                    <input type="time" id="startTime" value="09:00">
                    <label>Время окончания:</label>
                    <input type="time" id="endTime" value="17:00">
                    <label>Интервал (минуты):</label>
                    <select id="interval">
                        <option value="60">1 час</option>
                        <option value="30">30 минут</option>
                        <option value="45">45 минут</option>
                        <option value="90">1.5 часа</option>
                    </select>
                </div>
                <button class="btn btn-primary" onclick="addCustomDate()">Добавить дату</button>
            </div>
            
            <div id="datesList">
                <h3>Доступные даты:</h3>
                <div class="bulk-actions">
                    <label><input type="checkbox" id="selectAll" onchange="toggleSelectAll()"> Выбрать все</label>
                    <button class="btn btn-danger" onclick="deleteSelected()">Удалить выбранные</button>
                    <button class="btn btn-primary" onclick="loadDates()">Обновить список</button>
                </div>
                <div id="datesGrid" class="date-grid"></div>
            </div>
        </div>
        
        <div class="section">
            <h2>📋 Заявки</h2>
            <button class="btn btn-primary" onclick="loadRequests()">Обновить список заявок</button>
            <div id="requestsList"></div>
        </div>
    </div>

    <!-- Модальное окно для редактирования слотов -->
    <div id="editModal" class="modal">
        <div class="modal-content">
            <span class="close" onclick="closeModal()">&times;</span>
            <h3>Редактирование временных слотов</h3>
            <div id="modalContent"></div>
        </div>
    </div>

    <script>
        window.onload = function() {
            loadDates();
            loadRequests();
        };

        function loadDates() {
            fetch('/api/dates')
                .then(response => response.json())
                .then(data => {
                    const grid = document.getElementById('datesGrid');
                    grid.innerHTML = '';
                    
                    data.forEach(date => {
                        const card = document.createElement('div');
                        card.className = 'date-card ' + (date.is_active ? 'active' : '');
                        card.dataset.id = date.id;
                        
                        const dateStr = new Date(date.date).toLocaleDateString('ru-RU');
                        const weekday = new Date(date.date).toLocaleDateString('ru-RU', {weekday: 'long'});
                        
                        const timeSlots = date.time_slots.filter(slot => !slot.is_booked).length;
                        const totalSlots = date.time_slots.length;
                        
                        // Создаем HTML для временных слотов
                        let slotsHtml = '';
                        date.time_slots.forEach(slot => {
                            const slotClass = slot.is_booked ? 'time-slot booked' : 'time-slot available';
                            slotsHtml += '<span class="' + slotClass + '">' + slot.time + '</span>';
                        });
                        
                        card.innerHTML = '<input type="checkbox" class="checkbox" onchange="toggleDateSelection(this)">' +
                                       '<div><strong>' + dateStr + ' (' + weekday + ')</strong></div>' +
                                       '<div class="time-slots">Свободных слотов: ' + timeSlots + ' из ' + totalSlots + '</div>' +
                                       '<div class="edit-slots">' + slotsHtml + '</div>' +
                                       '<button class="btn btn-primary" onclick="editSlots(\'' + date.id + '\')">Редактировать слоты</button>' +
                                       '<button class="btn btn-danger" onclick="deleteDate(\'' + date.id + '\')">Удалить</button>';
                        
                        grid.appendChild(card);
                    });
                });
        }

        function addNextWeek() {
            fetch('/api/add-date', {
                method: 'POST',
                headers: {'Content-Type': 'application/json'},
                body: JSON.stringify({type: 'week'})
            }).then(() => loadDates());
        }

        function addNextMonth() {
            fetch('/api/add-date', {
                method: 'POST',
                headers: {'Content-Type': 'application/json'},
                body: JSON.stringify({type: 'month'})
            }).then(() => loadDates());
        }

        function addCustomDate() {
            const date = document.getElementById('customDate').value;
            const startTime = document.getElementById('startTime').value;
            const endTime = document.getElementById('endTime').value;
            const interval = parseInt(document.getElementById('interval').value);
            
            if (!date) {
                alert('Выберите дату!');
                return;
            }
            
            if (!startTime || !endTime) {
                alert('Укажите время начала и окончания!');
                return;
            }
            
            fetch('/api/add-date', {
                method: 'POST',
                headers: {'Content-Type': 'application/json'},
                body: JSON.stringify({
                    type: 'custom', 
                    date: date,
                    startTime: startTime,
                    endTime: endTime,
                    interval: interval
                })
            }).then(() => {
                loadDates();
                document.getElementById('customDate').value = '';
            });
        }

        function deleteDate(id) {
            if (confirm('Удалить эту дату?')) {
                fetch('/api/delete-date', {
                    method: 'POST',
                    headers: {'Content-Type': 'application/json'},
                    body: JSON.stringify({id: id})
                }).then(() => loadDates());
            }
        }

        function toggleDateSelection(checkbox) {
            const card = checkbox.closest('.date-card');
            if (checkbox.checked) {
                card.classList.add('selected');
            } else {
                card.classList.remove('selected');
            }
        }

        function toggleSelectAll() {
            const selectAll = document.getElementById('selectAll');
            const checkboxes = document.querySelectorAll('.date-card .checkbox');
            
            checkboxes.forEach(checkbox => {
                checkbox.checked = selectAll.checked;
                toggleDateSelection(checkbox);
            });
        }

        function deleteSelected() {
            const selectedCheckboxes = document.querySelectorAll('.date-card .checkbox:checked');
            
            if (selectedCheckboxes.length === 0) {
                alert('Выберите даты для удаления!');
                return;
            }
            
            if (!confirm('Удалить ' + selectedCheckboxes.length + ' выбранных дат?')) {
                return;
            }
            
            const deletePromises = [];
            
            selectedCheckboxes.forEach(checkbox => {
                const card = checkbox.closest('.date-card');
                const id = card.dataset.id;
                
                deletePromises.push(
                    fetch('/api/delete-date', {
                        method: 'POST',
                        headers: {'Content-Type': 'application/json'},
                        body: JSON.stringify({id: id})
                    })
                );
            });
            
            Promise.all(deletePromises).then(() => {
                loadDates();
                document.getElementById('selectAll').checked = false;
            });
        }

        function editSlots(dateId) {
            fetch('/api/dates')
                .then(response => response.json())
                .then(dates => {
                    const date = dates.find(d => d.id === dateId);
                    if (!date) {
                        alert('Дата не найдена!');
                        return;
                    }
                    
                    const modal = document.getElementById('editModal');
                    const modalContent = document.getElementById('modalContent');
                    
                    let slotsHtml = '<div><strong>' + new Date(date.date).toLocaleDateString('ru-RU') + '</strong></div>';
                    slotsHtml += '<div style="margin: 15px 0;">';
                    
                    date.time_slots.forEach((slot, index) => {
                        const checked = slot.is_booked ? 'checked' : '';
                        slotsHtml += '<div style="margin: 5px 0;">' +
                                   '<input type="checkbox" id="slot_' + index + '" ' + checked + '>' +
                                   '<label for="slot_' + index + '">' + slot.time + '</label>' +
                                   '</div>';
                    });
                    
                    slotsHtml += '</div>';
                    slotsHtml += '<button class="btn btn-primary" onclick="saveSlots(\'' + dateId + '\')">Сохранить</button>';
                    slotsHtml += '<button class="btn btn-danger" onclick="closeModal()">Отмена</button>';
                    
                    modalContent.innerHTML = slotsHtml;
                    modal.style.display = 'block';
                });
        }

        function saveSlots(dateId) {
            const checkboxes = document.querySelectorAll('#modalContent input[type="checkbox"]');
            const slots = [];
            
            checkboxes.forEach((checkbox, index) => {
                slots.push({
                    index: index,
                    is_booked: checkbox.checked
                });
            });
            
            fetch('/api/update-slots', {
                method: 'POST',
                headers: {'Content-Type': 'application/json'},
                body: JSON.stringify({
                    dateId: dateId,
                    slots: slots
                })
            }).then(() => {
                closeModal();
                loadDates();
            });
        }

        function closeModal() {
            document.getElementById('editModal').style.display = 'none';
        }

        function loadRequests() {
            fetch('/api/requests')
                .then(response => response.json())
                .then(data => {
                    const container = document.getElementById('requestsList');
                    
                    if (data.length === 0) {
                        container.innerHTML = '<p>Заявок пока нет</p>';
                        return;
                    }
                    
                    let html = '<table class="requests-table">';
                    html += '<tr><th>Дата создания</th><th>Имя</th><th>Контакт</th><th>Модель</th><th>Проблема</th><th>Время записи</th><th>Статус</th></tr>';
                    
                    data.forEach(request => {
                        const date = new Date(request.created_at).toLocaleDateString('ru-RU');
                        const appointmentDate = request.appointment_date ? 
                            new Date(request.appointment_date).toLocaleDateString('ru-RU') + ' ' + 
                            new Date(request.appointment_date).toLocaleTimeString('ru-RU', {hour: '2-digit', minute: '2-digit'}) : 
                            'Не указано';
                        
                        html += '<tr>' +
                               '<td>' + date + '</td>' +
                               '<td>' + request.name + '</td>' +
                               '<td>' + request.contact + '</td>' +
                               '<td>' + request.volvo_model + ' ' + request.year + '</td>' +
                               '<td>' + request.problem + '</td>' +
                               '<td>' + appointmentDate + '</td>' +
                               '<td>' + request.status + '</td>' +
                               '</tr>';
                    });
                    
                    html += '</table>';
                    container.innerHTML = html;
                });
        }
    </script>
</body>
</html>`

	tmplData := struct {
		Today string
	}{
		Today: time.Now().Format("2006-01-02"),
	}

	t, err := template.New("admin").Parse(tmpl)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	t.Execute(w, tmplData)
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
