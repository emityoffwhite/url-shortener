# URL Shortener

REST API сервис для сокращения ссылок на Go. Написан со слоистой архитектурой (handler → service → storage), полностью на стандартной библиотеке `net/http` (Go 1.22+), с тестами, race detector в CI и Docker-образом.

## Возможности

- Сокращение длинных URL до 7-символьного кода
- Редирект по короткой ссылке с подсчётом переходов
- Опциональное время жизни ссылки (TTL)
- Статистика по ссылке (сколько раз кликнули, когда создана)
- Graceful shutdown (корректно завершает запросы при остановке контейнера)

## Архитектура

```
cmd/server        - точка входа, сборка зависимостей, graceful shutdown
internal/handler   - HTTP слой: роутинг, парсинг запросов, статус-коды
internal/service    - бизнес-логика: генерация кода, валидация URL, TTL
internal/storage    - интерфейс Storage + in-memory реализация
internal/model     - доменная модель URL
internal/config    - конфигурация из переменных окружения
```

Слои зависят от интерфейсов, а не от конкретных реализаций. `internal/storage.Storage` — это интерфейс; `MemoryStorage` - одна из возможных реализаций. Чтобы добавить Postgres или Redis, не нужно трогать `service` или `handler` - только реализовать тот же интерфейс и подставить его в `main.go`.

## Технические решения и почему

- **`net/http.ServeMux` вместо `gin`/`chi`** - начиная с Go 1.22 стандартный роутер умеет методы (`POST /path`) и path-параметры (`{code}`), внешний роутер был бы избыточной зависимостью для такого размера API.
- **`sync.RWMutex` в `MemoryStorage`** - чтения (редиректы) происходят значительно чаще записей (создание ссылок), RWMutex позволяет нескольким чтениям идти параллельно.
- **Инкремент счётчика кликов в отдельной горутине** — пользователь не должен ждать обновления статистики, чтобы получить редирект.
- **`context.Context` пробрасывается через все слои** — позволяет корректно отменять операции и поддерживает таймауты на уровне HTTP-сервера.

## Быстрый старт

### Через Docker (рекомендуется)

```bash
docker compose up --build
```

Сервис поднимется на `http://localhost:8080`.

### Локально

```bash
go run ./cmd/server
```

Требуется Go 1.22+.

## Примеры запросов

**Создать короткую ссылку:**
```bash
curl -X POST http://localhost:8080/api/v1/shorten \
  -H "Content-Type: application/json" \
  -d '{"url": "https://github.com"}'
```
Ответ:
```json
{
  "short_code": "aB3xY9z",
  "short_url": "http://localhost:8080/aB3xY9z",
  "original_url": "https://github.com"
}
```

**Создать ссылку с TTL (например, на 1 час):**
```bash
curl -X POST http://localhost:8080/api/v1/shorten \
  -H "Content-Type: application/json" \
  -d '{"url": "https://github.com", "ttl_seconds": 3600}'
```

**Перейти по короткой ссылке:**
```bash
curl -L http://localhost:8080/aB3xY9z
```

**Посмотреть статистику:**
```bash
curl http://localhost:8080/api/v1/stats/aB3xY9z
```
Ответ:
```json
{
  "short_code": "aB3xY9z",
  "original_url": "https://github.com",
  "created_at": "2026-06-28T10:00:00Z",
  "clicks": 3
}
```

**Удалить ссылку:**
```bash
curl -X DELETE http://localhost:8080/api/v1/aB3xY9z
```

## Тестирование

```bash
go test -race -cover ./...
```

Проект покрыт unit-тестами на уровне service (бизнес-логика, валидация, TTL), storage (включая тест на конкурентный доступ из 100 горутин) и handler (HTTP integration-тесты через `httptest`).

## CI

GitHub Actions при каждом push/PR в `main`:
1. Проверяет форматирование (`gofmt`)
2. Запускает `go vet` и `golangci-lint`
3. Прогоняет тесты с race detector
4. Собирает Docker-образ

## Возможные улучшения

- [ ] Реализация `Storage` для PostgreSQL (интерфейс уже готов)
- [ ] Rate limiting на endpoint создания ссылок
- [ ] Кастомные алиасы (пользователь задаёт свой короткий код)
- [ ] Метрики Prometheus (`/metrics`)
- [ ] Кэш горячих ссылок в Redis перед основным хранилищем


