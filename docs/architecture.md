# Архитектура платформы CoachLink

## 1. Общее описание

CoachLink — открытая микросервисная платформа для автоматизации тренировочного процесса в лёгкой атлетике и взаимодействия в системе «тренер — спортсмен».

### 1.1 Высокоуровневая схема

```
┌─────────────────────────────────────────────────────────────────┐
│                     Mobile App (Flutter/Dart)                    │
│                                                                  │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐           │
│  │  Auth     │ │  Profile │ │ Training │ │  Groups  │   ...     │
│  │  Module   │ │  Module  │ │  Module  │ │  Module  │           │
│  └────┬─────┘ └────┬─────┘ └────┬─────┘ └────┬─────┘           │
│       └─────────────┴────────────┴─────────────┘                │
│                         │ HTTPS/REST                             │
└─────────────────────────┼───────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────────────┐
│                       API Gateway (Go)                           │
│           Маршрутизация · JWT-валидация · Rate Limiting          │
└──────┬──────────┬──────────┬──────────┬─────────────────────────┘
       │          │          │          │
       ▼          ▼          ▼          ▼
┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────────┐
│   Auth   │ │   User   │ │ Training │ │ Notification │
│ Service  │ │ Service  │ │ Service  │ │   Service    │
│   (Go)   │ │   (Go)   │ │   (Go)   │ │    (Go)      │
└────┬─────┘ └────┬─────┘ └────┬─────┘ └──────┬───────┘
     │            │            │               │
     ▼            ▼            ▼               ▼
┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────────┐
│ auth_db  │ │ user_db  │ │training_ │ │notification_ │
│(Postgres)│ │(Postgres)│ │   db     │ │     db       │
│          │ │          │ │(Postgres)│ │  (Postgres)  │
└──────────┘ └──────────┘ └──────────┘ └──────────────┘
                    ▲            ▲               ▲
                    │            │               │
                    └────────────┴───────────────┘
                              │
                         ┌────┴────┐
                         │  NATS   │
                         │ (broker)│
                         └─────────┘
```

### 1.2 Принципы архитектуры

1. **Независимость сервисов** — каждый сервис имеет собственную БД, развёртывается и масштабируется независимо.
2. **API-first** — все контракты описаны в OpenAPI 3.0 до начала разработки, фронт и бэк работают параллельно.
3. **Event-driven коммуникация** — сервисы обмениваются событиями через NATS для асинхронных операций (уведомления, синхронизация данных).
4. **Синхронные вызовы между сервисами** — по внутренним gRPC/HTTP API, только когда данные нужны здесь и сейчас.
5. **Расширяемость** — архитектура спроектирована с учётом будущих модулей (аналитика, Backend-Driven UI, AI-анализ).

---

## 2. Технологический стек

### 2.1 Backend

| Компонент           | Технология                                      |
|---------------------|-------------------------------------------------|
| Язык                | Go 1.22+                                        |
| HTTP-фреймворк      | [Echo](https://echo.labstack.com/) или [Fiber](https://gofiber.io/) |
| ORM / Query Builder | [sqlx](https://github.com/jmoiron/sqlx) (рекомендуется для контроля над SQL) |
| Миграции БД         | [goose](https://github.com/pressly/goose)       |
| Аутентификация      | JWT (access + refresh tokens) с библиотекой [golang-jwt](https://github.com/golang-jwt/jwt) |
| Хеширование паролей | bcrypt (`golang.org/x/crypto/bcrypt`)            |
| Валидация           | [go-playground/validator](https://github.com/go-playground/validator) |
| Message Broker      | [NATS](https://nats.io/) с JetStream            |
| Кеширование         | Redis 7+                                        |
| База данных         | PostgreSQL 16+                                  |
| Контейнеризация     | Docker + Docker Compose                          |
| API-документация    | OpenAPI 3.0 (Swagger)                           |
| Логирование         | [zerolog](https://github.com/rs/zerolog)        |

### 2.2 Mobile (Flutter)

| Компонент             | Технология                                       |
|-----------------------|--------------------------------------------------|
| Язык                  | Dart 3.x                                         |
| Фреймворк             | Flutter 3.x                                      |
| State Management      | BLoC (flutter_bloc)                              |
| HTTP-клиент           | Dio                                              |
| DI                    | GetIt + Injectable                               |
| Навигация             | GoRouter                                         |
| Хранение токенов      | flutter_secure_storage                           |
| Локальный кеш         | Hive или drift (SQLite)                          |
| Push-уведомления      | Firebase Cloud Messaging (FCM)                   |
| Генерация API-клиента | openapi-generator (dart)                         |

### 2.3 Инфраструктура

| Компонент            | Технология                  |
|----------------------|-----------------------------|
| Оркестрация (dev)    | Docker Compose              |
| CI/CD                | GitHub Actions              |
| Оркестрация (prod)   | Docker Compose / Kubernetes |
| Мониторинг           | Prometheus + Grafana        |
| Трассировка          | OpenTelemetry + Jaeger      |

> **Почему Go, а не C++/userver?**
> Go значительно проще для разработки микросервисов: быстрая компиляция, мощная стандартная библиотека, богатая экосистема для HTTP/gRPC/NATS. Для дипломного проекта это означает ускорение разработки в 3-5 раз без потери производительности (Go-сервисы держат десятки тысяч RPS). Если вы хотите продемонстрировать владение C++, рекомендую реализовать на userver один вычислительно-интенсивный сервис — например, будущий **Analytics Service**.

---

## 3. Микросервисы

### 3.1 API Gateway

**Ответственность:** единая точка входа для всех клиентских запросов.

**Функции:**
- Маршрутизация запросов к нужному сервису
- Валидация JWT access-токенов
- Проброс идентификатора пользователя и роли во внутренних заголовках (`X-User-ID`, `X-User-Role`)
- Rate limiting (по IP и по пользователю)
- CORS-заголовки
- Агрегация health-check'ов сервисов

**Маршрутизация:**

| Префикс               | Целевой сервис       |
|------------------------|----------------------|
| `/api/v1/auth/*`       | Auth Service         |
| `/api/v1/users/*`      | User Service         |
| `/api/v1/connections/*`| User Service         |
| `/api/v1/groups/*`     | User Service         |
| `/api/v1/training/*`   | Training Service     |
| `/api/v1/notifications/*` | Notification Service |

**Без авторизации (JWT не проверяется):**
- `POST /api/v1/auth/register`
- `POST /api/v1/auth/login`
- `POST /api/v1/auth/refresh`

Все остальные маршруты требуют валидный access-токен в заголовке `Authorization: Bearer <token>`.

---

### 3.2 Auth Service

**Ответственность:** регистрация, аутентификация, управление токенами.

**Порт:** 8001

**Функции:**
- Регистрация пользователя (ФИО, логин, email, пароль, роль)
- Валидация уникальности логина
- Валидация формата логина (`^[a-zA-Z0-9-]+$`)
- Хеширование пароля (bcrypt, cost=12)
- Выдача JWT access-token (срок жизни: 15 минут) и refresh-token (срок жизни: 30 дней)
- Обновление access-token по refresh-token
- Отзыв refresh-token (logout)

**JWT Claims (payload access-токена):**
```json
{
  "sub": "uuid пользователя",
  "login": "user_login",
  "role": "coach | athlete",
  "exp": 1234567890,
  "iat": 1234567890
}
```

**События (публикует в NATS):**
- `user.registered` — при успешной регистрации (user_id, login, full_name, email, role)
- `user.updated` — при обновлении профиля (если добавим эту функцию в Auth)

**База данных: `auth_db`**

```sql
-- Таблица пользователей
CREATE TABLE users (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    login       VARCHAR(50) UNIQUE NOT NULL,
    email       VARCHAR(255) NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    full_name   VARCHAR(255) NOT NULL,
    role        VARCHAR(20) NOT NULL CHECK (role IN ('coach', 'athlete')),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_users_login ON users(login);
CREATE INDEX idx_users_email ON users(email);

-- Таблица refresh-токенов
CREATE TABLE refresh_tokens (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash  VARCHAR(255) NOT NULL,
    expires_at  TIMESTAMPTZ NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_refresh_tokens_user_id ON refresh_tokens(user_id);
CREATE INDEX idx_refresh_tokens_expires_at ON refresh_tokens(expires_at);
```

---

### 3.3 User Service

**Ответственность:** профили пользователей, связи «тренер — спортсмен», тренировочные группы, поиск.

**Порт:** 8002

**Функции:**
- Хранение профилей пользователей (синхронизация из Auth Service через события NATS)
- Поиск пользователей по ФИО и логину (для привязки спортсмена к тренеру, для добавления в группу)
- Управление заявками на привязку спортсмен → тренер
- Управление тренировочными группами
- Предоставление данных о составе групп другим сервисам (внутренний API)

**Поиск:**
Для текстового поиска по ФИО и логину используется `tsvector`/`tsquery` PostgreSQL (полнотекстовый поиск) или `ILIKE` с триграммным индексом (`pg_trgm`). Рекомендуется `pg_trgm` — он лучше работает для поиска подстрок (автодополнение по мере ввода).

```sql
CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE INDEX idx_user_profiles_search ON user_profiles
    USING gin ((full_name || ' ' || login) gin_trgm_ops);
```

**Внутренний API (service-to-service):**
- `GET /internal/groups/{id}/members` — возвращает список athlete_id группы (вызывается Training Service)

**События (публикует в NATS):**
- `connection.requested` — спортсмен отправил заявку
- `connection.accepted` — тренер принял заявку
- `connection.rejected` — тренер отклонил заявку
- `group.athlete_added` — спортсмен добавлен в группу
- `group.athlete_removed` — спортсмен удалён из группы

**События (подписан):**
- `user.registered` — создаёт запись в `user_profiles`

**База данных: `user_db`**

```sql
-- Профили пользователей (синхронизируются из Auth Service)
CREATE TABLE user_profiles (
    id          UUID PRIMARY KEY,
    login       VARCHAR(50) UNIQUE NOT NULL,
    email       VARCHAR(255) NOT NULL,
    full_name   VARCHAR(255) NOT NULL,
    role        VARCHAR(20) NOT NULL CHECK (role IN ('coach', 'athlete')),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Индекс для текстового поиска (pg_trgm)
CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE INDEX idx_user_profiles_fullname_trgm ON user_profiles
    USING gin (full_name gin_trgm_ops);
CREATE INDEX idx_user_profiles_login_trgm ON user_profiles
    USING gin (login gin_trgm_ops);

-- Заявки на привязку спортсмен → тренер
CREATE TABLE connection_requests (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    athlete_id  UUID NOT NULL REFERENCES user_profiles(id),
    coach_id    UUID NOT NULL REFERENCES user_profiles(id),
    status      VARCHAR(20) NOT NULL DEFAULT 'pending'
                    CHECK (status IN ('pending', 'accepted', 'rejected')),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(athlete_id, coach_id)
);

CREATE INDEX idx_connection_requests_coach ON connection_requests(coach_id, status);
CREATE INDEX idx_connection_requests_athlete ON connection_requests(athlete_id, status);

-- Установленные связи «тренер — спортсмен»
CREATE TABLE coach_athlete_relations (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    coach_id    UUID NOT NULL REFERENCES user_profiles(id),
    athlete_id  UUID NOT NULL UNIQUE REFERENCES user_profiles(id), -- у спортсмена один тренер
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_relations_coach ON coach_athlete_relations(coach_id);

-- Тренировочные группы
CREATE TABLE training_groups (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    coach_id    UUID NOT NULL REFERENCES user_profiles(id),
    name        VARCHAR(255) NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_groups_coach ON training_groups(coach_id);

-- Состав групп
CREATE TABLE training_group_members (
    group_id    UUID NOT NULL REFERENCES training_groups(id) ON DELETE CASCADE,
    athlete_id  UUID NOT NULL REFERENCES user_profiles(id),
    added_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (group_id, athlete_id)
);

CREATE INDEX idx_group_members_athlete ON training_group_members(athlete_id);
```

---

### 3.4 Training Service

**Ответственность:** тренировочные планы, назначения, отчёты, шаблоны.

**Порт:** 8003

**Функции:**
- Создание тренировочного плана и назначение его спортсменам
- Назначение плана группе (запрашивает состав у User Service)
- Удаление планов и назначений
- Получение списка назначений (для тренера — выданные, для спортсмена — полученные)
- Фильтрация назначений по ФИО, логину, дате
- Архивирование выполненных назначений
- Приём отчётов от спортсменов
- Управление шаблонами тренировок
- Определение просроченных заданий (фоновая задача или вычисление при запросе)

**Логика просроченности:**
Задание считается просроченным, если:
- `status = 'assigned'` (отчёт не отправлен)
- `scheduled_date + 1 день < текущая_дата`

Рекомендуется вычислять просроченность при запросе списка (а не cron-задачей), добавляя виртуальное поле `is_overdue` в ответ API. Это проще и надёжнее.

**События (публикует в NATS):**
- `training.assigned` — задание назначено спортсмену
- `training.deleted` — задание удалено
- `report.submitted` — спортсмен отправил отчёт

**База данных: `training_db`**

```sql
-- Тренировочные планы
CREATE TABLE training_plans (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    coach_id        UUID NOT NULL,
    title           VARCHAR(255) NOT NULL,
    description     TEXT NOT NULL,
    scheduled_date  DATE NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_plans_coach ON training_plans(coach_id);
CREATE INDEX idx_plans_date ON training_plans(scheduled_date);

-- Назначения (план → спортсмен)
CREATE TABLE training_assignments (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    plan_id         UUID NOT NULL REFERENCES training_plans(id) ON DELETE CASCADE,
    athlete_id      UUID NOT NULL,
    coach_id        UUID NOT NULL,
    -- Денормализация для быстрой фильтрации:
    athlete_full_name VARCHAR(255) NOT NULL,
    athlete_login     VARCHAR(50) NOT NULL,
    status          VARCHAR(20) NOT NULL DEFAULT 'assigned'
                        CHECK (status IN ('assigned', 'completed', 'archived')),
    assigned_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at    TIMESTAMPTZ,
    archived_at     TIMESTAMPTZ
);

CREATE INDEX idx_assignments_coach ON training_assignments(coach_id, status);
CREATE INDEX idx_assignments_athlete ON training_assignments(athlete_id, status);
CREATE INDEX idx_assignments_date ON training_assignments(assigned_at);

-- Отчёты по тренировкам
CREATE TABLE training_reports (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    assignment_id       UUID NOT NULL UNIQUE REFERENCES training_assignments(id),
    athlete_id          UUID NOT NULL,
    content             TEXT NOT NULL,          -- текстовый комментарий
    duration_minutes    INTEGER NOT NULL,       -- длительность (минуты)
    perceived_effort    INTEGER NOT NULL        -- самочувствие / RPE (0-10)
                            CHECK (perceived_effort BETWEEN 0 AND 10),
    max_heart_rate      INTEGER,                -- макс. пульс (уд/мин)
    avg_heart_rate      INTEGER,                -- средний пульс (уд/мин)
    distance_km         DECIMAL(7,2),           -- дистанция (км)
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_reports_athlete ON training_reports(athlete_id);
CREATE INDEX idx_reports_assignment ON training_reports(assignment_id);

-- Шаблоны тренировок
CREATE TABLE training_templates (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    coach_id        UUID NOT NULL,
    title           VARCHAR(255) NOT NULL,
    description     TEXT NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_templates_coach ON training_templates(coach_id);
```

---

### 3.5 Notification Service

**Ответственность:** хранение и доставка уведомлений.

**Порт:** 8004

**Функции:**
- Приём событий из NATS и создание уведомлений
- Хранение уведомлений в БД
- Отдача списка уведомлений пользователю (с пагинацией)
- Пометка уведомлений как прочитанных
- Отправка push-уведомлений через Firebase Cloud Messaging (FCM)

**Типы уведомлений:**

| Тип                     | Получатель  | Описание                              |
|-------------------------|-------------|---------------------------------------|
| `connection_request`    | Тренер      | Спортсмен отправил заявку             |
| `connection_accepted`   | Спортсмен   | Тренер принял заявку                  |
| `connection_rejected`   | Спортсмен   | Тренер отклонил заявку                |
| `training_assigned`     | Спортсмен   | Назначено тренировочное задание       |
| `training_deleted`      | Спортсмен   | Тренер удалил задание                 |
| `report_submitted`      | Тренер      | Спортсмен отправил отчёт              |
| `group_added`           | Спортсмен   | Добавлен в тренировочную группу       |
| `group_removed`         | Спортсмен   | Удалён из тренировочной группы        |

**События (подписан на NATS):**
- `connection.requested`
- `connection.accepted`
- `connection.rejected`
- `training.assigned`
- `training.deleted`
- `report.submitted`
- `group.athlete_added`
- `group.athlete_removed`

**База данных: `notification_db`**

```sql
CREATE TABLE notifications (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL,
    type        VARCHAR(50) NOT NULL,
    title       VARCHAR(255) NOT NULL,
    body        TEXT,
    data        JSONB,                  -- дополнительные данные (id задания, ФИО и т.п.)
    is_read     BOOLEAN NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_notifications_user ON notifications(user_id, is_read, created_at DESC);

-- Токены устройств для push-уведомлений
CREATE TABLE device_tokens (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL,
    fcm_token   VARCHAR(500) NOT NULL,
    device_info VARCHAR(255),           -- опционально: модель устройства, ОС
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, fcm_token)
);

CREATE INDEX idx_device_tokens_user ON device_tokens(user_id);
```

---

## 4. Межсервисное взаимодействие

### 4.1 Синхронное (HTTP/gRPC)

Используется когда одному сервису нужны данные из другого сервиса **прямо сейчас**.

| Вызывающий         | Вызываемый     | Причина                                           |
|--------------------|----------------|---------------------------------------------------|
| Training Service   | User Service   | Получить список спортсменов в группе при назначении тренировки группе |
| Training Service   | User Service   | Получить ФИО/логин спортсмена для денормализации   |

Внутренние вызовы идут напрямую между сервисами (минуя Gateway), на отдельных внутренних портах или через отдельный префикс `/internal/`.

### 4.2 Асинхронное (NATS JetStream)

Используется для:
- **Уведомлений** — все сервисы публикуют события, Notification Service подписан и создаёт уведомления.
- **Синхронизации данных** — Auth Service публикует `user.registered`, User Service подписан и создаёт профиль.

**Структура топиков NATS:**

```
coachlink.user.registered
coachlink.connection.requested
coachlink.connection.accepted
coachlink.connection.rejected
coachlink.training.assigned
coachlink.training.deleted
coachlink.report.submitted
coachlink.group.athlete_added
coachlink.group.athlete_removed
```

**Формат сообщений (JSON):**

```json
{
  "event_id": "uuid",
  "event_type": "connection.requested",
  "timestamp": "2026-03-28T12:00:00Z",
  "payload": {
    "request_id": "uuid",
    "athlete_id": "uuid",
    "athlete_full_name": "Иванов Иван",
    "coach_id": "uuid"
  }
}
```

Для JetStream настраивается durable consumer для каждого подписчика, чтобы гарантировать доставку даже если сервис временно недоступен.

### 4.3 Диаграмма взаимодействий

```
                       ┌─────────────────────┐
                       │   Notification Svc   │
                       │  (подписан на все    │
                       │   события NATS)      │
                       └──────────┬──────────┘
                                  │ NATS
                                  │
    ┌──────────┐  NATS   ┌───────┴──────┐  NATS   ┌──────────────┐
    │  Auth    ├────────►│    NATS      │◄────────┤   Training   │
    │ Service  │         │   Broker     │         │   Service    │
    └──────────┘         │  (JetStream) │         └──────┬───────┘
                         └───────┬──────┘                │
                                 │ NATS                  │ HTTP (internal)
                                 │                       │
                         ┌───────┴──────┐                │
                         │    User      │◄───────────────┘
                         │   Service    │
                         └──────────────┘
```

---

## 5. Аутентификация и авторизация

### 5.1 Процесс регистрации

```
Client                Gateway              Auth Service         NATS            User Service
  │                      │                      │                 │                   │
  │ POST /auth/register  │                      │                 │                   │
  ├─────────────────────►│                      │                 │                   │
  │                      ├─────────────────────►│                 │                   │
  │                      │                      │ validate input  │                   │
  │                      │                      │ check unique    │                   │
  │                      │                      │ hash password   │                   │
  │                      │                      │ save to DB      │                   │
  │                      │                      │                 │                   │
  │                      │                      │ publish event   │                   │
  │                      │                      ├────────────────►│                   │
  │                      │                      │                 │  user.registered  │
  │                      │                      │                 ├──────────────────►│
  │                      │                      │                 │                   │ create profile
  │                      │                      │                 │                   │
  │                      │◄─────────────────────┤                 │                   │
  │◄─────────────────────┤  {access, refresh}   │                 │                   │
  │  200 OK + tokens     │                      │                 │                   │
```

### 5.2 Процесс входа

```
Client → POST /auth/login {login, password}
  → Auth Service: найти пользователя по логину, проверить пароль (bcrypt)
  → Создать access-token (JWT, 15 мин) и refresh-token (opaque, 30 дней)
  → Сохранить hash refresh-token в БД
  → Вернуть клиенту: {access_token, refresh_token, expires_in}
```

### 5.3 Обновление токена

```
Client → POST /auth/refresh {refresh_token}
  → Auth Service: проверить refresh-token (найти hash в БД, проверить срок)
  → Удалить старый refresh-token из БД
  → Создать новую пару access + refresh
  → Вернуть клиенту: {access_token, refresh_token, expires_in}
```

### 5.4 Авторизация в Gateway

Для каждого входящего запроса (кроме auth-маршрутов):
1. Извлечь access-token из `Authorization: Bearer <token>`
2. Проверить подпись JWT (HMAC-SHA256 с общим секретом или RSA)
3. Проверить `exp` (срок действия)
4. Извлечь `sub` (user_id) и `role`
5. Добавить заголовки:
   - `X-User-ID: <user_id>`
   - `X-User-Role: <role>`
6. Перенаправить запрос целевому сервису

Сервисы доверяют заголовкам от Gateway и используют их для авторизации (проверка роли, владения ресурсом).

---

## 6. Архитектура мобильного приложения (Flutter)

### 6.1 Структура проекта

```
lib/
├── main.dart                         # Точка входа
├── app.dart                          # MaterialApp, тема, роутер
│
├── core/                             # Инфраструктурный слой
│   ├── api/
│   │   ├── api_client.dart           # Dio-клиент, base URL, interceptors
│   │   ├── auth_interceptor.dart     # Добавляет Bearer токен к запросам
│   │   ├── token_refresh_interceptor.dart # Автоматически обновляет access-token
│   │   └── api_exceptions.dart       # Обработка HTTP-ошибок
│   ├── auth/
│   │   ├── auth_manager.dart         # Хранение/обновление токенов
│   │   └── auth_state.dart           # Текущее состояние авторизации
│   ├── di/
│   │   └── injection.dart            # GetIt-контейнер
│   ├── navigation/
│   │   ├── app_router.dart           # GoRouter конфигурация
│   │   └── routes.dart               # Константы маршрутов
│   └── theme/
│       ├── app_theme.dart            # Тема приложения
│       └── app_colors.dart           # Цветовая палитра
│
├── features/                         # Feature-based модули
│   ├── auth/                         # Вход и регистрация
│   │   ├── data/
│   │   │   ├── auth_repository.dart
│   │   │   └── dto/                  # RegisterRequest, LoginRequest, etc.
│   │   ├── domain/
│   │   │   └── models/               # User, AuthTokens
│   │   └── presentation/
│   │       ├── bloc/
│   │       │   ├── auth_bloc.dart
│   │       │   ├── auth_event.dart
│   │       │   └── auth_state.dart
│   │       └── screens/
│   │           ├── login_screen.dart
│   │           └── register_screen.dart
│   │
│   ├── profile/                      # Личный кабинет
│   │   ├── data/
│   │   ├── domain/
│   │   └── presentation/
│   │       └── screens/
│   │           └── profile_screen.dart
│   │
│   ├── connections/                  # Связи тренер-спортсмен
│   │   ├── data/
│   │   ├── domain/
│   │   └── presentation/
│   │       ├── bloc/
│   │       └── screens/
│   │           ├── coach_search_screen.dart    # Поиск тренера (для спортсмена)
│   │           ├── pending_requests_screen.dart # Входящие заявки (для тренера)
│   │           └── athletes_list_screen.dart   # Список спортсменов (для тренера)
│   │
│   ├── groups/                       # Тренировочные группы
│   │   ├── data/
│   │   ├── domain/
│   │   └── presentation/
│   │       ├── bloc/
│   │       └── screens/
│   │           ├── groups_list_screen.dart
│   │           ├── group_detail_screen.dart
│   │           └── add_athlete_to_group_screen.dart
│   │
│   ├── training/                     # Тренировочные задания
│   │   ├── data/
│   │   ├── domain/
│   │   └── presentation/
│   │       ├── bloc/
│   │       └── screens/
│   │           ├── create_plan_screen.dart       # Создание плана (тренер)
│   │           ├── coach_assignments_screen.dart  # Выданные задания (тренер)
│   │           ├── athlete_assignments_screen.dart# Назначенные задания (спортсмен)
│   │           ├── assignment_detail_screen.dart  # Детали задания
│   │           ├── archived_assignments_screen.dart # Архив (тренер)
│   │           ├── templates_screen.dart          # Шаблоны (тренер)
│   │           └── create_template_screen.dart    # Создание шаблона
│   │
│   ├── reports/                      # Отчёты
│   │   ├── data/
│   │   ├── domain/
│   │   └── presentation/
│   │       ├── bloc/
│   │       └── screens/
│   │           ├── submit_report_screen.dart     # Отправка отчёта (спортсмен)
│   │           └── view_report_screen.dart       # Просмотр отчёта (тренер)
│   │
│   └── notifications/                # Уведомления
│       ├── data/
│       ├── domain/
│       └── presentation/
│           ├── bloc/
│           └── screens/
│               └── notifications_screen.dart
│
└── shared/                           # Общие компоненты
    ├── widgets/
    │   ├── loading_indicator.dart
    │   ├── error_widget.dart
    │   ├── search_field.dart         # Поисковая строка с debounce
    │   ├── date_picker_field.dart
    │   └── selectable_list_tile.dart  # Элемент списка с чекбоксом
    └── extensions/
        ├── date_extensions.dart
        └── context_extensions.dart
```

### 6.2 Архитектурный паттерн: BLoC + Repository

Каждая feature использует трёхслойную архитектуру:

```
┌──────────────────────────────────────────────┐
│              Presentation (UI)               │
│         Screens, Widgets, BLoC               │
│                                               │
│  Screen  ──►  BLoC  ──►  State  ──►  Screen │
│              (events)     (states)            │
└──────────────────┬───────────────────────────┘
                   │ depends on
                   ▼
┌──────────────────────────────────────────────┐
│             Domain (Business Logic)          │
│           Models, Repository interface       │
└──────────────────┬───────────────────────────┘
                   │ implemented by
                   ▼
┌──────────────────────────────────────────────┐
│               Data (Infrastructure)          │
│       Repository impl, API calls, DTOs       │
└──────────────────────────────────────────────┘
```

### 6.3 Ключевые паттерны

**Token Refresh Interceptor (Dio):**
```dart
// Перехватчик автоматически обновляет access-token при получении 401
// 1. Запрос возвращает 401
// 2. Interceptor вызывает POST /auth/refresh с refresh-token
// 3. Сохраняет новые токены
// 4. Повторяет оригинальный запрос с новым access-token
// 5. Если refresh тоже невалиден → разлогинить пользователя
```

**Debounced Search:**
```dart
// Для поиска тренера / спортсмена:
// 1. Пользователь вводит текст
// 2. Debounce 300ms (не отправлять запрос на каждый символ)
// 3. Если текст >= 2 символов → GET /users/search?q=...
// 4. Показать результаты в выпадающем списке
```

**Навигация на основе роли:**
```dart
// После авторизации GoRouter перенаправляет на:
// - Для тренера: /coach/dashboard (список спортсменов, группы, задания)
// - Для спортсмена: /athlete/dashboard (задания, мой тренер, группы)
// Если спортсмен без тренера → /athlete/find-coach
```

### 6.4 Основные экраны

| Экран                  | Роль       | Описание                                       |
|------------------------|------------|------------------------------------------------|
| Login                  | Все        | Вход по логину и паролю                        |
| Register               | Все        | Регистрация с выбором роли                     |
| Profile                | Все        | Личный кабинет: ФИО, email, логин, роль        |
| Find Coach             | Спортсмен  | Поиск тренера с автодополнением, отправка заявки|
| My Coach               | Спортсмен  | Информация о текущем тренере                   |
| Pending Requests       | Тренер     | Входящие заявки от спортсменов                 |
| Athletes List          | Тренер     | Список спортсменов с чекбоксами и select all   |
| Create Training Plan   | Тренер     | Форма: title, description, date + выбор спортсменов/группы |
| Coach Assignments      | Тренер     | Список выданных заданий с фильтрацией          |
| Athlete Assignments    | Спортсмен  | Список полученных заданий                      |
| Assignment Detail      | Все        | Детали задания (description)                   |
| Submit Report          | Спортсмен  | Форма отчёта                                   |
| View Report            | Тренер     | Просмотр отчёта спортсмена                     |
| Archived Assignments   | Тренер     | Архив выполненных заданий                      |
| Groups List            | Все        | Список групп                                   |
| Group Detail           | Тренер     | Состав группы, добавление/удаление спортсменов |
| Athlete Groups         | Спортсмен  | Группы, в которых состоит                      |
| Templates              | Тренер     | Список шаблонов тренировок                     |
| Notifications          | Все        | Список уведомлений                             |

---

## 7. Инфраструктура

### 7.1 Docker Compose (для локальной разработки)

```yaml
# docker-compose.yml (структура)
services:
  # --- Инфраструктура ---
  postgres:
    image: postgres:16
    # Создаёт 4 БД при инициализации: auth_db, user_db, training_db, notification_db

  redis:
    image: redis:7-alpine

  nats:
    image: nats:2.10
    command: ["--jetstream"]

  # --- Сервисы ---
  api-gateway:
    build: ./services/api-gateway
    ports: ["8080:8080"]
    depends_on: [auth-service, user-service, training-service, notification-service]

  auth-service:
    build: ./services/auth-service
    ports: ["8001:8001"]
    depends_on: [postgres, nats]

  user-service:
    build: ./services/user-service
    ports: ["8002:8002"]
    depends_on: [postgres, nats]

  training-service:
    build: ./services/training-service
    ports: ["8003:8003"]
    depends_on: [postgres, nats]

  notification-service:
    build: ./services/notification-service
    ports: ["8004:8004"]
    depends_on: [postgres, nats, redis]
```

### 7.2 Структура репозитория (монорепо)

```
coach-link-platform/
├── docs/
│   ├── architecture.md               # Этот документ
│   └── api/
│       └── openapi.yaml              # OpenAPI-спецификация
│
├── services/                          # Backend-сервисы
│   ├── api-gateway/
│   │   ├── cmd/main.go
│   │   ├── internal/
│   │   │   ├── middleware/            # JWT validation, rate limiting
│   │   │   ├── proxy/                # Reverse proxy к сервисам
│   │   │   └── config/
│   │   ├── Dockerfile
│   │   └── go.mod
│   │
│   ├── auth-service/
│   │   ├── cmd/main.go
│   │   ├── internal/
│   │   │   ├── handler/              # HTTP-обработчики
│   │   │   ├── service/              # Бизнес-логика
│   │   │   ├── repository/           # Работа с БД
│   │   │   ├── model/                # Модели данных
│   │   │   └── config/
│   │   ├── migrations/               # SQL-миграции (goose)
│   │   ├── Dockerfile
│   │   └── go.mod
│   │
│   ├── user-service/
│   │   ├── cmd/main.go
│   │   ├── internal/
│   │   │   ├── handler/
│   │   │   ├── service/
│   │   │   ├── repository/
│   │   │   ├── model/
│   │   │   └── config/
│   │   ├── migrations/
│   │   ├── Dockerfile
│   │   └── go.mod
│   │
│   ├── training-service/
│   │   ├── cmd/main.go
│   │   ├── internal/
│   │   │   ├── handler/
│   │   │   ├── service/
│   │   │   ├── repository/
│   │   │   ├── model/
│   │   │   └── config/
│   │   ├── migrations/
│   │   ├── Dockerfile
│   │   └── go.mod
│   │
│   └── notification-service/
│       ├── cmd/main.go
│       ├── internal/
│       │   ├── handler/
│       │   ├── service/
│       │   ├── repository/
│       │   ├── consumer/             # NATS consumers
│       │   ├── push/                 # FCM integration
│       │   ├── model/
│       │   └── config/
│       ├── migrations/
│       ├── Dockerfile
│       └── go.mod
│
├── mobile/                            # Flutter-приложение
│   └── coach_link/
│       ├── lib/
│       │   └── ...                   # Структура из раздела 6.1
│       ├── pubspec.yaml
│       └── ...
│
├── deployments/
│   ├── docker-compose.yml
│   ├── docker-compose.dev.yml
│   └── init-databases.sql            # Скрипт инициализации БД
│
└── README.md
```

### 7.3 Конфигурация сервисов

Каждый сервис конфигурируется через переменные окружения:

```env
# Пример для auth-service
APP_PORT=8001
DB_HOST=postgres
DB_PORT=5432
DB_NAME=auth_db
DB_USER=coachlink
DB_PASSWORD=secret
NATS_URL=nats://nats:4222
JWT_SECRET=your-jwt-secret-key
JWT_ACCESS_TTL=15m
JWT_REFRESH_TTL=720h
BCRYPT_COST=12
```

---

## 8. Подготовка к будущим расширениям

### 8.1 Analytics Service (аналитика и графики)

**Когда:** после реализации базовой функциональности.

Отдельный микросервис, который:
- Подписан на `report.submitted` события через NATS
- Агрегирует данные отчётов (объём, интенсивность, километраж, пульс) по периодам
- Предоставляет API для получения статистики:
  - `GET /api/v1/analytics/athlete/{id}/summary?from=...&to=...`
  - `GET /api/v1/analytics/athlete/{id}/charts?metric=distance&period=month`
- Данные для графиков: объём нагрузки по неделям, динамика пульса, прогресс дистанции
- Может использовать отдельное хранилище (TimescaleDB) для эффективной работы с временными рядами

> Этот сервис — отличный кандидат для реализации на C++/userver, если хотите показать владение языком в дипломе.

### 8.2 Структурированные тренировки

Расширение Training Service:
- Добавить таблицу `exercises` (название, категория, описание)
- Добавить таблицу `plan_exercises` (plan_id, exercise_id, sets, reps, distance, pace, rest)
- План будет содержать как текстовое описание, так и структурированный список упражнений
- Во Flutter: интерактивный билдер тренировки с добавлением упражнений

### 8.3 Backend-Driven UI

Отдельный сервис, который:
- Описывает экраны приложения в виде JSON-схемы (виджеты, лейаут, данные)
- Flutter-приложение содержит рендерер, умеющий строить UI по JSON
- Позволяет обновлять интерфейс без обновления приложения
- Полезен для динамического контента: промо-баннеры, сезонные изменения, A/B тесты

### 8.4 AI-анализ (нейросеть)

Отдельный сервис (Python/FastAPI + модель):
- Принимает историю тренировок спортсмена
- Анализирует тренды, выявляет перетренированность
- Генерирует рекомендации для тренера
- Интеграция через NATS или REST API

---

## 9. Справочник API (краткий)

Полная OpenAPI-спецификация — в файле `docs/api/openapi.yaml`.

### Auth Service

| Метод | Путь                   | Описание                      |
|-------|------------------------|-------------------------------|
| POST  | /api/v1/auth/register  | Регистрация                   |
| POST  | /api/v1/auth/login     | Вход                          |
| POST  | /api/v1/auth/refresh   | Обновление access-token       |
| POST  | /api/v1/auth/logout    | Выход (отзыв refresh-token)   |

### User Service

| Метод  | Путь                                      | Описание                           |
|--------|-------------------------------------------|------------------------------------|
| GET    | /api/v1/users/me                          | Профиль текущего пользователя      |
| GET    | /api/v1/users/search?q=...&role=...       | Поиск пользователей                |
| POST   | /api/v1/connections/request               | Отправить заявку тренеру           |
| GET    | /api/v1/connections/requests/incoming      | Входящие заявки (тренер)           |
| GET    | /api/v1/connections/requests/outgoing      | Исходящая заявка (спортсмен)       |
| PUT    | /api/v1/connections/requests/{id}/accept   | Принять заявку (тренер)            |
| PUT    | /api/v1/connections/requests/{id}/reject   | Отклонить заявку (тренер)          |
| GET    | /api/v1/connections/athletes              | Список спортсменов тренера         |
| GET    | /api/v1/connections/coach                 | Тренер спортсмена                  |
| DELETE | /api/v1/connections/athletes/{id}         | Отвязать спортсмена                |
| POST   | /api/v1/groups                            | Создать группу                     |
| GET    | /api/v1/groups                            | Список групп                       |
| GET    | /api/v1/groups/{id}                       | Детали группы                      |
| PUT    | /api/v1/groups/{id}                       | Обновить группу                    |
| DELETE | /api/v1/groups/{id}                       | Удалить группу                     |
| POST   | /api/v1/groups/{id}/members               | Добавить спортсмена в группу       |
| DELETE | /api/v1/groups/{id}/members/{athleteId}   | Удалить спортсмена из группы       |

### Training Service

| Метод  | Путь                                         | Описание                              |
|--------|----------------------------------------------|---------------------------------------|
| POST   | /api/v1/training/plans                       | Создать план и назначить              |
| GET    | /api/v1/training/assignments                 | Список назначений (фильтрация)        |
| GET    | /api/v1/training/assignments/{id}            | Детали назначения                     |
| DELETE | /api/v1/training/assignments/{id}            | Удалить назначение                    |
| PUT    | /api/v1/training/assignments/{id}/archive    | Архивировать назначение               |
| GET    | /api/v1/training/assignments/archived        | Архив назначений                      |
| POST   | /api/v1/training/assignments/{id}/report     | Отправить отчёт (спортсмен)          |
| GET    | /api/v1/training/assignments/{id}/report     | Просмотреть отчёт                     |
| POST   | /api/v1/training/templates                   | Создать шаблон                        |
| GET    | /api/v1/training/templates                   | Список шаблонов                       |
| GET    | /api/v1/training/templates/{id}              | Детали шаблона                        |
| PUT    | /api/v1/training/templates/{id}              | Обновить шаблон                       |
| DELETE | /api/v1/training/templates/{id}              | Удалить шаблон                        |

### Notification Service

| Метод | Путь                                | Описание                     |
|-------|-------------------------------------|------------------------------|
| GET   | /api/v1/notifications               | Список уведомлений           |
| PUT   | /api/v1/notifications/{id}/read     | Пометить прочитанным         |
| PUT   | /api/v1/notifications/read-all      | Пометить все прочитанными    |
| POST  | /api/v1/notifications/device-token  | Зарегистрировать FCM-токен   |
