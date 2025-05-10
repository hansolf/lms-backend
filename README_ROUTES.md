# API Роуты LMS (Go backend)

## Auth

### POST /api/send/register
- Ожидает:
```json
{
  "email": "user@example.com",
  "password": "string"
}
```
- Ответ:
```json
{
  "message": "Код отправлен на почту"
}
```

### POST /api/auth/verify
- Ожидает:
```json
{
  "code": "string"
}
```
- Ответ:
```json
{
  "token": "jwt-token-string"
}
```

### POST /api/auth/login
- Ожидает:
```json
{
  "email": "user@example.com",
  "password": "string"
}
```
- Ответ:
```json
{
  "token": "jwt-token-string"
}
```

### DELETE /api/auth/logout
- Ожидает: (cookie с токеном)
- Ответ: 204 No Content

### GET /api/verifyteach/{id}
- Ожидает: (ничего)
- Ответ: HTML-страница подтверждения

---

## Me

### GET /api/me
- Ожидает: (cookie с токеном)
- Ответ:
```json
{
  "id": 1,
  "name": "Имя",
  "secondname": "Фамилия",
  "fakultet": "Кафедра",
  "vuz": "ВУЗ",
  "myCourses": [ ... ],
  "email": "user@example.com",
  "role": "студент|преподаватель|администратор",
  "created_at": "2024-05-01T12:00:00Z"
}
```

### PATCH /api/me/update
- Ожидает:
```json
{
  "name": "string",
  "secondname": "string",
  "vuz": "string",
  "kafedra": "string",
  "fakultet": "string"
}
```
- Ответ: обновлённый объект пользователя (см. выше)

### POST /api/me/teacher
- Ожидает:
```json
{
  "name": "string",
  "secondname": "string",
  "vuz": "string",
  "kafedra": "string"
}
```
- Ответ: { "message": "Заявка отправлена" }

---

## Courses

### GET /api/courses/search
- Ожидает: query-параметры
- Ответ: массив курсов

### GET /api/courses/search/deep
- Ожидает: query-параметры
- Ответ: массив курсов

### GET /api/courses
- Ожидает: (ничего)
- Ответ: массив курсов

### POST /api/courses
- Ожидает:
```json
{
  "name": "string",
  "description": "string",
  ...
}
```
- Ответ: созданный курс

### GET /api/courses/{id}
- Ожидает: (ничего)
- Ответ: объект курса

### PUT /api/courses/{id}
- Ожидает: поля курса для обновления
- Ответ: обновлённый курс

### DELETE /api/courses/{id}
- Ожидает: (ничего)
- Ответ: 200 OK

### GET /api/courses/{id}/details
- Ожидает: (ничего)
- Ответ: подробный объект курса

### POST /api/courses/{courseID}/enroll
- Ожидает: (ничего)
- Ответ: объект UserCourse

### GET /api/courses/user
- Ожидает: user_id (query)
- Ответ: массив курсов пользователя

### PUT /api/courses/status
- Ожидает:
```json
{
  "user_id": 1,
  "course_id": 2,
  "status": "string"
}
```
- Ответ: обновлённый UserCourse

---

## Lessons

### GET /api/courses/{courseID}/lessons
- Ожидает: (ничего)
- Ответ: массив уроков

### POST /api/courses/{courseID}/lessons
- Ожидает:
```json
{
  "title": "string",
  "description": "string",
  ...
}
```
- Ответ: созданный урок

### GET /api/courses/{courseID}/lessons/{id}
- Ожидает: (ничего)
- Ответ: объект урока

### PUT /api/courses/{courseID}/lessons/{id}
- Ожидает: поля для обновления
- Ответ: обновлённый урок

### DELETE /api/courses/{courseID}/lessons/{id}
- Ожидает: (ничего)
- Ответ: 204 No Content

---

## Tests

### POST /api/courses/{courseID}/lessons/{lessonID}/tests
- Ожидает:
```json
{
  "title": "string",
  ...
}
```
- Ответ: созданный тест

### GET /api/courses/{courseID}/lessons/{lessonID}/tests/{id}
- Ожидает: (ничего)
- Ответ: объект теста

### PUT /api/courses/{courseID}/lessons/{lessonID}/tests/{id}
- Ожидает: поля для обновления
- Ответ: обновлённый тест

### DELETE /api/courses/{courseID}/lessons/{lessonID}/tests/{id}
- Ожидает: (ничего)
- Ответ: 204 No Content

### GET /api/courses/{courseID}/lessons/{lessonID}/tests
- Ожидает: (ничего)
- Ответ: массив тестов

---

## Videos

### GET /api/courses/{courseID}/lessons/{lessonID}/videos/{vidID}/stream/
- Ожидает: (ничего)
- Ответ: video/mp4

### POST /api/courses/{courseID}/lessons/{lessonID}/videos/upload
- Ожидает: multipart/form-data (файл)
- Ответ: объект видео

### GET /api/courses/{courseID}/lessons/{lessonID}/videos/{vidID}/download/
- Ожидает: (ничего)
- Ответ: файл

### GET /api/courses/{courseID}/lessons/{lessonID}/video/{vidID}/summary
- Ожидает: (ничего)
- Ответ: summary JSON

---

## Chat

### POST /api/chat
- Ожидает:
```json
{
  "question": "string"
}
```
- Ответ:
```json
{
  "chat_id": "uuid",
  "answer": { ... }
}
```

### WebSocket /ws/chat/{id}
- Ожидает: JSON-сообщения вида
```json
{
  "type": "message",
  "content": "string",
  "timestamp": "2024-05-01T12:00:00Z"
}
```
- Ответ: JSON-сообщения вида
```json
{
  "type": "message",
  "content": "string",
  "timestamp": "2024-05-01T12:00:00Z"
}
```

---

*Если для какого-то роута нужны точные поля — уточните, и я добавлю подробности!* 