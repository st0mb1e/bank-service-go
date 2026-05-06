# Банковский REST API

Выполнил: Ярцев Владислав

Этот репозиторий - реализация банковского веб-сервиса на Go: регистрация и вход, счета и операции по ним, виртуальные карты, кредиты с графиком платежей, аналитика, интеграция с API Центробанка (ключевая ставка) и с SMTP для писем, фоновая обработка просроченных платежей по расписанию. Ниже описано, как запустить проект у себя, как его проверить и как устроены каталоги.

---

## Что нужно для запуска

- **Go** - 1.26.2
- **PostgreSQL 17** локально или через Docker. Расширение `pgcrypto` создаётся миграцией

Внешние библиотеки перечислены в `go.mod`; при первой сборке или тесте они подтянутся автоматически (`go mod download`), отдельно ставить пакеты не обязательно.

---

## Запуск сервиса

1. Скопировать пример переменных окружения и заполнить секреты:

   ```bash
   cp .env.example .env
   ```

   Обязательно задать непустые строки: `JWT_SECRET`, `HMAC_SECRET`, `PGP_PASSPHRASE`. При желании можно заполнить переменные окружения SMTP для отправки почты

2. Поднять базу (из корня репозитория):

   ```bash
   docker compose up -d
   ```

3. Применить миграции. Утилита ищет каталог `migrations/` относительно **текущей рабочей директории**, поэтому команды выполняются из корня проекта:

   ```bash
   go run ./cli/migrate
   ```

4. Запустить HTTP-сервер:

   ```bash
   go run ./cli/app
   ```

   Порт задаётся переменной `APP_PORT` (по умолчанию в примере - 8080). Уровень логирования - `LOG_LEVEL` (обрабатывается через `logrus`).

---

## Поддерживаемые команды

| Действие | Команда |
|----------|---------|
| Запуск API | `go run ./cli/app` |
| Миграции вверх | `go run ./cli/migrate` |
| Сборка всего модуля | `go build ./...` |
| Юнит-тесты | `go test ./...` |
| Поднятие/сворачивание базы данных через Docker | `docker compose up -d` / `docker compose down` |

---

## Структура проекта (по каталогам)

| Каталог | Назначение |
|---------|------------|
| `cli/app` | Точка входа: конфиг, роутинг, middleware, подключение к БД |
| `cli/migrate` | Применение SQL-миграций через `golang-migrate` |
| `config` | Чтение настроек из переменных окружения |
| `dao/entity` | Структуры данных, согласованные с таблицами БД |
| `dao/repo` | SQL-репозитории, параметризованные запросы |
| `dto` | JSON для регистрации/логина и валидация полей |
| `service` | Бизнес-логика (счета, переводы, карты, кредиты, аналитика, JWT, шедулер) |
| `handler/auth`, `handler/api` | Обработчики публичных и защищённых маршрутов |
| `middleware` | JWT и запись идентификатора пользователя в контекст запроса |
| `integration/cbr` | Запрос ключевой ставки ЦБ РФ (SOAP + разбор XML) |
| `integration/mail` | Отправка писем через SMTP |
| `cryptoutil` | Лун, HMAC, симметричное PGP-подобное шифрование для данных карт |
| `httputil` | Вспомогательные ответы JSON |
| `migrations` | SQL-файлы схемы |
| `bank-service-go` | Ручная коллекция **Bruno** для примеров запросов (см. ниже) |

---

## Ручное тестирование через curl

Ниже предполагается, что API доступен по `http://localhost:8080`.

Удобно задать переменные:

```bash
export BASE=http://localhost:8080
```

### 1. Регистрация

```bash
curl -s -X POST "$BASE/register" \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","username":"testuser","password":"password12"}'
```

Ожидается ответ с полями `id`, `email`, `username`.

### 2. Вход и сохранение токена

```bash
TOKEN=$(curl -s -X POST "$BASE/login" \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"password12"}' | jq -r '.token')
echo "$TOKEN"
```

Если утилиты `jq` нет, можно скопировать значение `token` из ответа вручную и выполнить `export TOKEN='...'`.

### 3. Создать счёт

```bash
curl -s -X POST "$BASE/accounts" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json"
```

В ответе будет `id` счёта - его нужно подставлять дальше как `ACCOUNT_ID`.

### 4. Пополнение и списание

```bash
curl -s -X POST "$BASE/accounts/ACCOUNT_ID/deposit" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"amount":"1000.00"}'

curl -s -X POST "$BASE/accounts/ACCOUNT_ID/withdraw" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"amount":"100.00"}'
```

### 5. Перевод между счетами

Нужны два разных счёта (например два `POST /accounts`). Подставьте реальные UUID:

```bash
curl -s -X POST "$BASE/transfer" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"from_account_id":"UUID_ИСТОЧНИКА","to_account_id":"UUID_ПОЛУЧАТЕЛЯ","amount":"50.00"}'
```

### 6. Карты

Выпуск (к счёту пользователя):

```bash
curl -s -X POST "$BASE/cards" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"account_id":"ACCOUNT_ID"}'
```

Список масок, просмотр с расшифровкой (после выпуска подставьте `CARD_ID`):

```bash
curl -s "$BASE/cards" -H "Authorization: Bearer $TOKEN"

curl -s "$BASE/cards/CARD_ID" -H "Authorization: Bearer $TOKEN"
```

Оплата с карты (списание с привязанного счёта):

```bash
curl -s -X POST "$BASE/cards/CARD_ID/pay" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"amount":"25.00"}'
```

### 7. Кредит и график

Оформление (нужны два счёта пользователя - зачисление и погашение):

```bash
curl -s -X POST "$BASE/credits" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "principal":"50000.00",
    "term_months":12,
    "disbursement_account_id":"UUID_ЗАЧИСЛЕНИЯ",
    "repayment_account_id":"UUID_ПОГАШЕНИЯ"
  }'
```

График платежей:

```bash
curl -s "$BASE/credits/CREDIT_ID/schedule" -H "Authorization: Bearer $TOKEN"
```

Для расчёта ставки сервис обращается к сервису ЦБ; нужен доступ в интернет с машины, где запущен API.

### 8. Аналитика и прогноз

```bash
curl -s "$BASE/analytics?month=2026-05" -H "Authorization: Bearer $TOKEN"

curl -s "$BASE/analytics/credit-load" -H "Authorization: Bearer $TOKEN"

curl -s "$BASE/accounts/ACCOUNT_ID/predict?days=30" -H "Authorization: Bearer $TOKEN"
```

Формат месяца в запросе аналитики - `YYYY-MM`.

---

## Ручная коллекция Bruno

В каталоге `bank-service-go` лежит файл **`auth/register.bru`** - это запрос к `POST /register`, собранный вручную в приложении [Bruno](https://www.usebruno.com/). Его можно открыть как часть коллекции и при желании дополнить аналогичными `.bru` для логина и защищённых маршрутов (с тем же базовым URL и заголовком `Authorization`). Это не обязательная часть репозитория для сборки, но удобна для повторяемых проверок без набора `curl`.

---

## Юнит-тесты

Автотесты не поднимают PostgreSQL: проверяются отдельные функции (валидация полей регистрации, алгоритм Луна, расчёт аннуитетного платежа).

```bash
go test ./...
```

Полезные варианты:

```bash
go test -v ./dto ./cryptoutil ./service
go test -race ./...
```

При успешном проходе все пакеты с тестами отмечаются как `ok`.
