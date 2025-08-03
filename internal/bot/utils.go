package bot

import "time"

// getWeekdayRussian возвращает сокращенное название дня недели на русском языке
func getWeekdayRussian(weekday time.Weekday) string {
	weekdays := map[time.Weekday]string{
		time.Monday:    "Пн",
		time.Tuesday:   "Вт",
		time.Wednesday: "Ср",
		time.Thursday:  "Чт",
		time.Friday:    "Пт",
		time.Saturday:  "Сб",
		time.Sunday:    "Вс",
	}
	return weekdays[weekday]
}
