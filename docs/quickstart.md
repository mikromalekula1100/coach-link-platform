# CoachLink — Быстрый старт

## Требования

- Docker и Docker Compose
- Make (обычно предустановлен на macOS/Linux)
- bash (для E2E тестов)

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
| `make test-e2e`       | Запустить E2E smoke-тест                              |

## Первый запуск

```bash
# 1. Клонируйте репозиторий
git clone https://github.com/mikromalekula1100/coach-link-platform.git
cd coach-link-platform

# 2. Запустите платформу
make up

# 3. Дождитесь запуска (10-30 секунд при первой сборке — дольше)
make ps

# 4. Откройте веб-интерфейс для тестирования
open http://localhost:3000

# 5. Или запустите автоматический E2E тест
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
| PostgreSQL           | 5432 | База данных                        |
| NATS                 | 4222 | Брокер сообщений                   |
| Redis                | 6379 | Кеш                               |

## Полный сброс

Если что-то пошло не так или нужно начать с чистого листа:

```bash
make clean
make up
```

Это удалит все данные из баз данных и пересоздаст контейнеры.

## API документация

Полная OpenAPI-спецификация: `docs/api/openapi.yaml`

Откройте в [Swagger Editor](https://editor.swagger.io/) для интерактивного просмотра.
