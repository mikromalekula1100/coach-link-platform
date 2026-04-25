# CoachLink — Быстрый старт

## Требования

- Docker и Docker Compose
- Make (обычно предустановлен на macOS/Linux)
- **macOS:** [Colima](https://github.com/abiosoft/colima) (Docker-runtime для macOS на Apple Silicon)

## Команды

| Команда               | Описание                                              |
|-----------------------|-------------------------------------------------------|
| `make up`             | Собрать и запустить все сервисы                        |
| `make down`           | Остановить все сервисы                                |
| `make restart`        | Перезапустить все сервисы                              |
| `make build`          | Только собрать образы (без запуска)                   |
| `make logs`           | Показать логи всех сервисов (в реальном времени)      |
| `make logs-auth`      | Логи Auth Service                                     |
| `make logs-user`      | Логи User Service                                     |
| `make logs-training`  | Логи Training Service                                 |
| `make logs-notification` | Логи Notification Service                          |
| `make logs-gateway`   | Логи API Gateway                                      |
| `make logs-web`       | Логи Web UI                                           |
| `make ps`             | Показать статус контейнеров                           |
| `make clean`          | Полный сброс: остановить, удалить volumes, пересобрать|
| `make swagger`        | Открыть Swagger UI в браузере                         |
| `make test-unit`      | Юнит-тесты бизнес-логики (Go, без Docker)             |
| `make test-e2e`       | Запустить E2E smoke-тест (bash, только happy path)    |
| `make test-integration` | Запустить интеграционные тесты (Go, требует make up) |

## Первый запуск

```bash
# 1. Клонируйте репозиторий
git clone https://github.com/mikromalekula1100/coach-link-platform.git
cd coach-link-platform

# 2. (macOS) Запустите Colima, если ещё не запущена
colima start

# 3. Запустите платформу
make up

# 4. Дождитесь запуска (10-30 секунд при первой сборке — дольше)
make ps

# 5. Откройте веб-интерфейс для тестирования
open http://localhost:3000

# 6. Запустите интеграционные тесты (110 тестов, все эндпоинты)
make test-integration

# Или быстрый smoke-тест (только happy path)
make test-e2e
```

## Сервисы и порты

| Сервис               | Порт | Описание                           |
|----------------------|------|------------------------------------|
| Web UI               | 3000 | Тестовый веб-интерфейс             |
| Swagger UI           | 8090 | Интерактивная документация API     |
| API Gateway          | 8080 | Единая точка входа для API         |
| Auth Service         | 8001 | Регистрация, вход, JWT             |
| User Service         | 8002 | Профили, связи, группы             |
| Training Service     | 8003 | Планы, назначения, отчёты          |
| Notification Service | 8004 | Уведомления                        |
| Analytics Service    | 8005 | Аналитика тренировок               |
| AI Service           | 8006 | LLM-рекомендации (через Ollama)    |
| Ollama               | 11434| Локальная LLM (gemma3:4b)          |
| PostgreSQL           | 5432 | База данных                        |
| NATS                 | 4222 | Брокер сообщений                   |
| Redis                | 6379 | Кеш                               |

## Тестирование

В проекте три уровня тестов:

| Команда | Что проверяет | Кол-во тестов | Нужен Docker |
|---------|---------------|---------------|--------------|
| `make test-unit` | Бизнес-логика каждого сервиса в изоляции | 52 теста | **Нет** |
| `make test-e2e` | Один happy-path сценарий (bash-скрипт) | ~17 шагов | Да |
| `make test-integration` | Все эндпоинты через API Gateway (Go) | ~96 тестов | Да |

### Юнит-тесты (`make test-unit`)

Запускаются **локально без Docker**. Покрывают бизнес-логику сервисного слоя через моки — реальные БД и HTTP-клиенты не нужны:

```bash
make test-unit
```

Можно запустить тесты отдельного сервиса:

```bash
cd services/ai-service       && go test ./internal/service/... -v
cd services/analytics-service && go test ./internal/service/... -v
cd services/training-service  && go test ./internal/service/... -v
cd services/notification-service && go test ./internal/service/... -v
```

Тесты для **auth-service** и **user-service** собираются и проходят внутри Docker при `make test-integration` (у них нет `go.sum` для локальной сборки вне Docker).

Что покрывают юнит-тесты:

| Сервис | Что тестируется |
|--------|----------------|
| **ai-service** | Успех/ошибки генерации, лимит 5 отчётов, содержимое промптов, недоступность Ollama |
| **analytics-service** | Агрегация статистики, группировка по неделям/месяцам, сортировка, ошибки клиента |
| **training-service** | `ComputeIsOverdue` (5 кейсов), доступ к заданиям (403/404), архивация, отправка отчётов, владение шаблонами |
| **notification-service** | Валидация FCM-токена, пагинация, пометка прочитанным, создание из события |
| **auth-service** | Валидация логина (все форматы), регистрация, дубликаты, refresh/logout |
| **user-service** | Поиск (минимум 2 символа), профили, связи (ролевые ограничения), группы |

### Интеграционные тесты (`make test-integration`)

Запускаются в Docker и тестируют всю систему через API Gateway — так же, как мобильное приложение. Каждый запуск использует уникальные логины, поэтому можно запускать многократно без очистки БД.

```bash
make up               # убедитесь, что сервисы запущены
make test-integration
```

Покрывают:
- Регистрация и авторизация (валидация, дубликаты, refresh-токены)
- Профили и поиск пользователей (пагинация, фильтрация по роли)
- Связи тренер-спортсмен (заявки, принятие/отклонение, ролевые проверки)
- Тренировочные группы (CRUD, управление участниками, права доступа)
- Шаблоны тренировок (CRUD, проверки владельца)
- Тренировочные планы и задания (создание, отчёты, архивация, удаление)
- Уведомления (список, пометка прочитанным, device token)
- Аналитика (сводка, прогресс, обзор тренера, ролевые проверки)
- AI-рекомендации (ролевые проверки, обработка недоступности Ollama)

## LLM (Ollama)

AI-сервис использует Ollama для генерации рекомендаций по тренировкам. Модель (`gemma3:4b`, ~3 ГБ) загружается **автоматически** при первом `make up`. Модель сохраняется в Docker volume и не скачивается повторно при перезапусках. В git она не попадает.

Проверить статус:
```bash
docker compose -f deployments/docker-compose.yml exec ollama ollama list
```

**Ускорение на Apple Silicon (M1/M2/M3):** для использования GPU (Metal) установите Ollama нативно (`brew install ollama && ollama serve`) и укажите в docker-compose для ai-service переменную `OLLAMA_URL: http://host.docker.internal:11434`.


## Остановка

```bash
make down           # остановить контейнеры (данные в volumes сохраняются)
colima stop         # (macOS) остановить Docker VM — освободит ~12 ГБ RAM
```

Для повторного запуска:
```bash
colima start        # (macOS)
make up
```

## Полный сброс

Если что-то пошло не так или нужно начать с чистого листа:

```bash
make clean          # остановить контейнеры и удалить volumes (все данные!)
make up
```

## API документация

Полная OpenAPI-спецификация: `docs/api/openapi.yaml`

Откройте в [Swagger Editor](https://editor.swagger.io/) для интерактивного просмотра.
