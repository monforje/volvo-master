// Загружаем даты при загрузке страницы
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