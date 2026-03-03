# Withdrawal Service

# Технологии
- Go 1.25
- PostgreSQL 18.1
- Rest API

# Структура
.
├── cmd/
│   └── app/
│       └── main.go
├── internal/
│   ├── config/
│   ├── middleware/
│   ├── postgres/
│   ├── server/
│   └── core/
|      ├── balances/
|      └── withdrawal/
├── testhelpers/
├── .env
├── http_tests.txt
├── schema.sql
├── go.mod
└── README.md

# БД

`users` — пользователи
`balances` — баланс пользователя. Уникальный индекс по `(user_id, currency)`
`withdrawals` — заявки на вывод.
  `idempotency_key` уникален для идемпотентности
  `status` — 'pending', 'confirmed', 'rejected', 'failed'
Индексы на `user_id`, `status`, `created_at`

# Идемпотентность

Используется уникальный `idempotency_key`
Запрос с уже существующим ключом:
  - если payload совпадает — возвращается существующая запись
  - если payload отличается — возвращается ошибка 422
В базе уникальный индекс `idempotency_key` предотвращает дублирование

# Консистентность

Вся операция `create withdrawal` выполняется в транзакции Postgres
Баланс блокируется через `SELECT ... FOR UPDATE` перед списанием

# Безопасность

Проверка `Authorization: Bearer <TOKEN>` через env
Валидация входных данных (`amount > 0`, `currency = 'USDT'`, обязательные поля)
Ошибки для API: 400/409/422/500, без утечки внутренней информации

# Запуск (локально без Docker)
psql -U postgres -c "CREATE DATABASE test;"
psql -U postgres -d test -h localhost -f schema.sql
go mod tidy
go run cmd/app/main.go

# Тесты
go test ./...

# Тесты API

# 1. Успешное создание
POST http://localhost:8080/v1/withdrawals
Content-Type: application/json
Authorization: Bearer secret-token-1234567890abcdef

{
  "user_id": "user-123",
  "amount": 5,
  "currency": "USDT",
  "destination": "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t",
  "idempotency_key": "xGTCi8q8ZxGTCi8q8ZxGTCi8q8Z"
}


# 1. Ответ 201
{
  "id": "f5163f56-11ab-48b3-9cf7-be64dbc186cf",
  "user_id": "user-123",
  "amount": 5,
  "currency": "USDT",
  "destination": "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t",
  "idempotency_key": "xGTCi8q8ZxGTCi8q8ZxGTCi8q8Z",
  "status": "pending",
  "created_at": "2026-03-03T08:39:18.9971415Z",
  "updated_at": "2026-03-03T08:39:18.9971415Z"
}

# 2. Повтор того же запроса → идемпотентность
POST http://localhost:8080/v1/withdrawals
Content-Type: application/json
Authorization: Bearer secret-token-1234567890abcdef

{
  "user_id": "user-123",
  "amount": 5,
  "currency": "USDT",
  "destination": "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t",
  "idempotency_key": "xGTCi8q8ZxGTCi8q8ZxGTCi8q8Z"
}


# 2. Ответ 200
{
  "id": "f5163f56-11ab-48b3-9cf7-be64dbc186cf",
  "user_id": "user-123",
  "amount": 5,
  "currency": "USDT",
  "destination": "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t",
  "idempotency_key": "xGTCi8q8ZxGTCi8q8ZxGTCi8q8Z",
  "status": "pending",
  "created_at": "2026-03-03T08:39:18.9971415Z",
  "updated_at": "2026-03-03T08:39:18.9971415Z"
}

# 3. Тот же idempotency_key, но другие данные
POST http://localhost:8080/v1/withdrawals
Content-Type: application/json
Authorization: Bearer secret-token-1234567890abcdef

{
  "user_id": "user-123",
  "amount": 7,
  "currency": "USDT",
  "destination": "0xDifferentAddr999",
  "idempotency_key": "xGTCi8q8ZxGTCi8q8ZxGTCi8q8Z"
}

# 3. Ответ 422
{
  "error": "idempotency key exists but payload differs"
}

# 4. Отрицательная сумма → 400
POST http://localhost:8080/v1/withdrawals
Content-Type: application/json
Authorization: Bearer secret-token-1234567890abcdef

{
  "user_id": "user-123",
  "amount": -100,
  "currency": "USDT",
  "destination": "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t",
  "idempotency_key": "xGTCi8q8ZxGTCi8q8ZxGTCi8q8Z"
}

# 4. Ответ 400
{
  "error": "amount must be positive"
}

# 5. Нулевая сумма → 400
POST http://localhost:8080/v1/withdrawals
Content-Type: application/json
Authorization: Bearer secret-token-1234567890abcdef

{
  "user_id": "user-123",
  "amount": 0,
  "currency": "USDT",
  "destination": "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t",
  "idempotency_key": "xGTCi8q8ZxGTCi8q8ZxGTCi8q8Z"
}

# 5. Ответ 400
{
  "error": "amount must be positive"
}

# 6. Сумма больше баланса → 409
POST http://localhost:8080/v1/withdrawals
Content-Type: application/json
Authorization: Bearer secret-token-1234567890abcdef

{
  "user_id": "user-123",
  "amount": 25,
  "currency": "USDT",
  "destination": "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t",
  "idempotency_key": "6548q8ZxGTCi8q8ZxGTCi8q8Z"
}

# 6. Ответ 409
{
  "error": "insufficient balance"
}

# 7. Получить конкретную заявку
GET http://localhost:8080/v1/withdrawals/f5163f56-11ab-48b3-9cf7-be64dbc186cf
Authorization: Bearer secret-token-1234567890abcdef

# 7. Ответ 200
{
  "id": "f5163f56-11ab-48b3-9cf7-be64dbc186cf",
  "user_id": "user-123",
  "amount": 5,
  "currency": "USDT",
  "destination": "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t",
  "idempotency_key": "xGTCi8q8ZxGTCi8q8ZxGTCi8q8Z",
  "status": "pending",
  "created_at": "2026-03-03T13:39:18.997141+05:00",
  "updated_at": "2026-03-03T13:39:18.997141+05:00"
}

# 8. Несуществующий ID
GET http://localhost:8080/v1/withdrawals/00000000-0000-0000-0000-000000000000
Authorization: Bearer secret-token-1234567890abcdef

# 8. Ответ 404
{
  "error": "not found"
}

# 9. Без токена
POST http://localhost:8080/v1/withdrawals
Content-Type: application/json

{
  "user_id": "user-123",
  "amount": 3,
  "currency": "USDT",
  "destination": "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t",
  "idempotency_key": "xGTCi8q8ZxGTCi8q8ZxGTCi8q8Z"
}

# 9. Ответ 401
{
  "error": "missing authorization header"
}

# 10. Неверный токен
POST http://localhost:8080/v1/withdrawals
Content-Type: application/json
Authorization: Bearer secret-token-3234567890abcdef

{
  "user_id": "user-123",
  "amount": 3,
  "currency": "USDT",
  "destination": "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t",
  "idempotency_key": "xGTCi8q8ZxGTCi8q8ZxGTCi8q8Z"
}

# 10. Ответ 401
{
  "error": "unauthorized"
}
