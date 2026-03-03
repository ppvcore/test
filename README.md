# Withdrawal Service (backend-test-assignment)

# Технологии
- Go 1.25
- PostgreSQL 18.1
- Rest API

# БД

- `users` — пользователи
- `balances` — баланс пользователя. Уникальный индекс по `(user_id, currency)`
- `withdrawals` — заявки на вывод.
  - `idempotency_key` уникален для идемпотентности
  - `status` — 'pending', 'confirmed', 'rejected', 'failed'
- Индексы на `user_id`, `status`, `created_at`

# Идемпотентность

- Используется уникальный `idempotency_key`
- Запрос с уже существующим ключом:
  - если payload совпадает — возвращается существующая запись
  - если payload отличается — возвращается ошибка 422
- В базе уникальный индекс `idempotency_key` предотвращает дублирование

# Консистентность

- Вся операция `create withdrawal` выполняется в транзакции Postgres
- Баланс блокируется через `SELECT ... FOR UPDATE` перед списанием

# Безопасность

- Проверка `Authorization: Bearer <TOKEN>` через env
- Валидация входных данных (`amount > 0`, `currency = 'USDT'`, обязательные поля)
- Ошибки для API: 400/409/422/500, без утечки внутренней информации

# Тесты

- TestWithdrawalRepo_CreateIdempotent
  - первый insert создает запись
  - повтор с тем же ключом возвращает существующую запись
  - повтор с тем же ключом, но другим payload возвращает существующую запись с отличающимися данными
- TestWithdrawalService_Create
  - успешное создание заявки
  - повтор с тем же idempotency_key возвращает существующую запись
  - конфликт payload с тем же ключом = ошибка
  - отрицательная сумма = ошибка
  - недостаточный баланс = ошибка
- TestWithdrawalService_ConcurrentSameIdempotencyKey
  - только один запрос создаёт запись
  - баланс списан корректно
  - все остальные повторные запросы возвращают существующую запись

# Запуск (локально без Docker)

psql -U postgres -c "CREATE DATABASE test;"<br>
psql -U postgres -d test -h localhost -f schema.sql<br>
go mod tidy<br>
go run cmd/app/main.go<br>

# Запуск тестов

go test ./...

# Тесты API

## 1. Успешное создание

POST http://localhost:8080/v1/withdrawals
<br>
Content-Type: application/json<br>
Authorization: Bearer secret-token-1234567890abcdef<br><br>

{<br>
"user_id": "user-123",<br>
"amount": 5,<br>
"currency": "USDT",<br>
"destination": "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t",<br>
"idempotency_key": "xGTCi8q8ZxGTCi8q8ZxGTCi8q8Z"<br>
}<br>

### Ответ 201<br>
{<br>
"id": "f5163f56-11ab-48b3-9cf7-be64dbc186cf",<br>
"user_id": "user-123",<br>
"amount": 5,<br>
"currency": "USDT",<br>
"destination": "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t",<br>
"idempotency_key": "xGTCi8q8ZxGTCi8q8ZxGTCi8q8Z",<br>
"status": "pending",<br>
"created_at": "2026-03-03T08:39:18.9971415Z",<br>
"updated_at": "2026-03-03T08:39:18.9971415Z"<br>
}<br>

## 2. Повтор того же запроса → идемпотентность

POST http://localhost:8080/v1/withdrawals
<br>
Content-Type: application/json<br>
Authorization: Bearer secret-token-1234567890abcdef<br><br>

{<br>
"user_id": "user-123",<br>
"amount": 5,<br>
"currency": "USDT",<br>
"destination": "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t",<br>
"idempotency_key": "xGTCi8q8ZxGTCi8q8ZxGTCi8q8Z"<br>
}<br>

### Ответ 200<br>
{<br>
"id": "f5163f56-11ab-48b3-9cf7-be64dbc186cf",<br>
"user_id": "user-123",<br>
"amount": 5,<br>
"currency": "USDT",<br>
"destination": "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t",<br>
"idempotency_key": "xGTCi8q8ZxGTCi8q8ZxGTCi8q8Z",<br>
"status": "pending",<br>
"created_at": "2026-03-03T08:39:18.9971415Z",<br>
"updated_at": "2026-03-03T08:39:18.9971415Z"<br>
}<br>

## 3. Тот же idempotency_key, но другие данные

POST http://localhost:8080/v1/withdrawals
<br>
Content-Type: application/json<br>
Authorization: Bearer secret-token-1234567890abcdef<br><br>

{<br>
"user_id": "user-123",<br>
"amount": 7,<br>
"currency": "USDT",<br>
"destination": "0xDifferentAddr999",<br>
"idempotency_key": "xGTCi8q8ZxGTCi8q8ZxGTCi8q8Z"<br>
}<br>

### Ответ 422<br>
{<br>
"error": "idempotency key exists but payload differs"<br>
}<br>

## 4. Отрицательная сумма → 400

POST http://localhost:8080/v1/withdrawals
<br>
Content-Type: application/json<br>
Authorization: Bearer secret-token-1234567890abcdef<br><br>

{<br>
"user_id": "user-123",<br>
"amount": -100,<br>
"currency": "USDT",<br>
"destination": "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t",<br>
"idempotency_key": "xGTCi8q8ZxGTCi8q8ZxGTCi8q8Z"<br>
}<br>

### Ответ 400<br>
{<br>
"error": "amount must be positive"<br>
}<br>

## 5. Нулевая сумма → 400

POST http://localhost:8080/v1/withdrawals
<br>
Content-Type: application/json<br>
Authorization: Bearer secret-token-1234567890abcdef<br><br>

{<br>
"user_id": "user-123",<br>
"amount": 0,<br>
"currency": "USDT",<br>
"destination": "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t",<br>
"idempotency_key": "xGTCi8q8ZxGTCi8q8ZxGTCi8q8Z"<br>
}<br>

### Ответ 400<br>
{<br>
"error": "amount must be positive"<br>
}<br>

## 6. Сумма больше баланса → 409

POST http://localhost:8080/v1/withdrawals
<br>
Content-Type: application/json<br>
Authorization: Bearer secret-token-1234567890abcdef<br><br>

{<br>
"user_id": "user-123",<br>
"amount": 25,<br>
"currency": "USDT",<br>
"destination": "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t",<br>
"idempotency_key": "6548q8ZxGTCi8q8ZxGTCi8q8Z"<br>
}<br>

### Ответ 409<br>
{<br>
"error": "insufficient balance"<br>
}<br>

## 7. Получить конкретную заявку

GET http://localhost:8080/v1/withdrawals/f5163f56-11ab-48b3-9cf7-be64dbc186cf
<br>
Authorization: Bearer secret-token-1234567890abcdef<br><br>

### Ответ 200<br>
{<br>
"id": "f5163f56-11ab-48b3-9cf7-be64dbc186cf",<br>
"user_id": "user-123",<br>
"amount": 5,<br>
"currency": "USDT",<br>
"destination": "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t",<br>
"idempotency_key": "xGTCi8q8ZxGTCi8q8ZxGTCi8q8Z",<br>
"status": "pending",<br>
"created_at": "2026-03-03T13:39:18.997141+05:00",<br>
"updated_at": "2026-03-03T13:39:18.997141+05:00"<br>
}<br>

## 8. Несуществующий ID

GET http://localhost:8080/v1/withdrawals/00000000-0000-0000-0000-000000000000
<br>
Authorization: Bearer secret-token-1234567890abcdef<br><br>

### Ответ 404<br>
{<br>
"error": "not found"<br>
}<br>

## 9. Без токена

POST http://localhost:8080/v1/withdrawals
<br>
Content-Type: application/json<br><br>

{<br>
"user_id": "user-123",<br>
"amount": 3,<br>
"currency": "USDT",<br>
"destination": "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t",<br>
"idempotency_key": "xGTCi8q8ZxGTCi8q8ZxGTCi8q8Z"<br>
}<br>

### Ответ 401<br>
{<br>
"error": "missing authorization header"<br>
}<br>

## 10. Неверный токен

POST http://localhost:8080/v1/withdrawals
<br>
Content-Type: application/json<br>
Authorization: Bearer secret-token-3234567890abcdef<br><br>

{<br>
"user_id": "user-123",<br>
"amount": 3,<br>
"currency": "USDT",<br>
"destination": "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t",<br>
"idempotency_key": "xGTCi8q8ZxGTCi8q8ZxGTCi8q8Z"<br>
}<br>

### Ответ 401<br>
{<br>
"error": "unauthorized"<br>
}<br>
