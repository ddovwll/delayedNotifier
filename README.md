# DelayedNotifier

Сервис отложенных уведомлений, который принимает HTTP-запросы, сохраняет задачи в PostgreSQL, планирует отправку через RabbitMQ и доставляет сообщения по выбранному каналу (Telegram или email). Для ускорения работы используется Redis‑кэш, а UI на чистом HTML/JS помогает протестировать работу без `curl`.

## Возможности
- создание, просмотр статуса и отмена уведомлений через REST API либо встроенный UI
- отложенная доставка с использованием плагина `x-delayed-message` в RabbitMQ
- повторные попытки отправки с настраиваемой стратегией ретраев
- кэширование статусов уведомлений в Redis для уменьшения нагрузки на БД
- поддержка Telegram-бота для регистрации получателей и отправки сообщений
- интеграция с SMTP (smtp4dev по умолчанию) для тестовой отправки email
- авто-генерируемая Swagger-документация и UI на `/swagger/`

## Архитектура
```
Client/UI --> HTTP API (Go net/http) --> NotificationService
                                        |--> PostgreSQL (notifications)
                                        |--> DeliveryTaskService -> RabbitMQ (x-delay)
RabbitMQ consumer --> NotificationService --> NotifierFactory --> Email / Telegram
                                    |
                                    +--> Redis cache (статусы)
```

## Технологический стек
- Go 1.25
- PostgreSQL 15
- RabbitMQ с плагином `rabbitmq_delayed_message_exchange`
- Redis 7
- smtp4dev (тестовый SMTP)
- Docker / Docker Compose

## Структура проекта
```
cmd/                     Точка входа приложения
internal/
  application/          Бизнес-сервисы и контракты
  domain/               Доменные модели и интерфейсы хранилищ
  infrastructure/       Интеграции: PostgreSQL, RabbitMQ, Redis, Telegram, SMTP
  web_api/              HTTP-контроллеры, DTO и статический UI
migrations/             SQL-миграции
Dockerfile, docker/     Образы и конфигурация окружения
```

## Основные компоненты
- **HTTP API** (`internal/web_api/controllers`) — обрабатывает POST/GET/DELETE запросы, рендерит HTML-страницу.
- **Сервис уведомлений** (`internal/application/services/notification_service.go`) — координирует сохранение, доставку и кэширование статусов.
- **Очередь** (`internal/infrastructure/message_queue`) — планирует доставку через RabbitMQ и обрабатывает задачи в фоне.
- **Нотификаторы** (`internal/infrastructure/notifier`) — Telegram и Email отправители, выбираются через `NotifierFactory`.
- **Кэш** (`internal/infrastructure/cache/redis.go`) — хранит сериализованные уведомления и ускоряет чтение статусов.
- **Миграции** (`migrations`) — базовая схема для таблиц `notifications` и `telegram_receivers`.

## Быстрый старт (Docker Compose)
1. Установите Docker и Docker Compose.
2. При необходимости обновите значения в `docker-compose.yml` (например, токен Telegram-бота или SMTP-данные).
3. Запустите окружение:
   ```bash
   docker compose up --build
   ```
4. Дождитесь логов `server is running`. Приложение станет доступно:
   - API: `http://localhost:8080`
   - UI: `http://localhost:8080/`
   - RabbitMQ UI: `http://localhost:15672` (логин/пароль: `user/user`)
   - smtp4dev UI: `http://localhost:8081`

> Скрипт `docker/app/wait-for-deps.sh` ждёт готовности PostgreSQL, RabbitMQ, Redis и SMTP перед стартом приложения.

### Миграции БД
Миграции не применяются автоматически. После поднятия контейнеров выполните из корня репозитория:
```bash
docker compose exec -T postgres psql -U postgres -d delayed_notifier < migrations/0001_create_notifications_table.up.sql
```
Либо примените SQL-файлы из каталога `migrations/` вручную любым удобным инструментом.

## Локальный запуск без Docker
1. Запустите PostgreSQL, RabbitMQ (с поддержкой `x-delayed-message`), Redis и SMTP-сервер (например, smtp4dev).
2. Примените миграции из каталога `migrations/`.
3. Настройте переменные окружения (см. таблицу ниже) или создайте `.env`.
4. Запустите приложение:
   ```bash
   go run ./cmd
   ```

## Переменные окружения
| Переменная | Назначение | Пример |
|------------|------------|--------|
| `POSTGRES_HOST` | хост PostgreSQL | `localhost` |
| `POSTGRES_PORT` | порт PostgreSQL | `5432` |
| `POSTGRES_USER` | пользователь БД | `postgres` |
| `POSTGRES_PASSWORD` | пароль БД | `postgres` |
| `POSTGRES_DB` | имя БД | `delayed_notifier` |
| `RABBITMQ_URL` | строка подключения к RabbitMQ | `amqp://user:pass@localhost:5672/` |
| `RABBITMQ_EXCHANGE` | имя exchange с типом `x-delayed-message` | `delayed-exchange` |
| `RABBITMQ_QUEUE` | очередь задержанных сообщений | `delayed-queue` |
| `RABBITMQ_ROUTING_KEY` | routing key для публикаций | `delayed-routing-key` |
| `REDIS_ADDRESS` | адрес Redis в формате `host:port` | `localhost:6379` |
| `REDIS_PASSWORD` | пароль Redis (может быть пустым) | `password` |
| `TELEGRAM_TOKEN` | токен Telegram-бота | `123456:ABCDEF` |
| `MAIL_FROM` | адрес отправителя email | `notifier@example.com` |
| `MAIL_SUBJECT` | тема письма | `Delayed Notification` |
| `MAIL_SMTP_ADDR` | адрес SMTP сервера | `smtp4dev:25` |
| `MAIL_HOST` | хост для SMTP аутентификации | `smtp4dev` |
| `MAIL_USERNAME` | логин SMTP | `user` |
| `MAIL_PASSWORD` | пароль SMTP | `pass` |
| `HTTP_HOST` | хост приложения (для заголовка `Location`) | `localhost` |
| `HTTP_PORT` | порт HTTP сервера | `8080` |

## HTTP API
| Метод | Путь | Описание |
|-------|------|----------|
| `POST` | `/notify` | Создать уведомление с отложенной отправкой |
| `GET` | `/notify/{id}` | Получить статус уведомления (`Scheduled`, `Sent`, `Failed`, `Cancelled`) |
| `DELETE` | `/notify/{id}` | Отменить запланированное уведомление |

Пример запроса на создание:
```bash
curl -X POST http://localhost:8080/notify \
  -H "Content-Type: application/json" \
  -d '{
        "channel": 0,
        "recipient": "telegram_username",
        "message": "Напоминание о встрече",
        "scheduled_at": "2024-05-20T18:30:00Z"
      }'
```
Ответ `201 Created` содержит `Location` и UUID уведомления в теле.

Пример проверки статуса:
```bash
curl http://localhost:8080/notify/<uuid>
```

## Встроенный UI
Страница `internal/web_api/public/index.html` позволяет создать уведомление, увидеть список созданных задач и отменить их. Для корректной работы необходимо, чтобы API было доступно по тому же домену/порту.

## Swagger UI
- Интерфейс доступен по адресу `http://localhost:8080/swagger/index.html` (при запуске через Docker Compose или локально на тех же портах).
- Доступные эндпойнты описаны в `docs/swagger.yaml` / `docs/swagger.json` — файлы генерируются автоматически утилитой [`swag`](https://github.com/swaggo/swag).
- Чтобы обновить документацию после изменений API, используйте команду:
  ```bash
  swag init -g cmd/main.go -o docs
  ```
- Если `swag` не установлен, поставьте его: `go install github.com/swaggo/swag/cmd/swag@latest`

## Работа с Telegram
1. Создайте бота через [@BotFather](https://t.me/BotFather) и сохраните токен в `TELEGRAM_TOKEN`.
2. Запустите бота и отправьте любое сообщение — сервис автоматически сохранит username и chat_id пользователя (в таблицу `telegram_receivers`).
3. После этого уведомления, отправленные на Telegram-канал, будут приходить пользователю.

## Электронная почта
В примере используется smtp4dev. Для реальных SMTP серверов установите корректные `MAIL_SMTP_ADDR`, `MAIL_HOST`, `MAIL_USERNAME`, `MAIL_PASSWORD`. Учтите, что в текущей версии приложения аутентификация SMTP не реализована.

## Тесты и проверка
```bash
GOCACHE=$(pwd)/.gocache go test ./...
```
На данный момент модульные тесты отсутствуют; цель команды — убедиться, что проект собирается.

## Диагностика
- Логи приложения пишутся в stdout (`log.SetFlags(log.LstdFlags | log.Lshortfile)`).
- RabbitMQ consumer и Telegram listener стартуют вместе с сервером и завершаются при получении SIGINT/SIGTERM.
- Для принудительного пересоздания окружения используйте `docker compose down -v`.

## Известные ограничения и идеи для развития
- Валидация входных данных минимальна — требуется запрет на создание уведомлений «в прошлом», более понятные сообщения об ошибках.
- Нужны модульные и интеграционные тесты на сервисы и работу очереди.
- Для production-окружения стоит добавить миграции при старте, обработку недоступности Bot API, полноценную SMTP-аутентификацию, метрики и централизованный логгер.
