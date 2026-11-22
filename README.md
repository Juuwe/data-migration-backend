# Сервис методов миграции данных

## HTTP-роуты

GET-запросы, создание и публикация черновика, постановка лайка, регистрация и вход возвращают JSON. Удаление, выход и ошибки API возвращают только HTTP-код без тела. Защищённые маршруты без действительного токена возвращают 401.

Swagger UI доступен по адресу `http://localhost:8081/swagger/index.html`. Для обновления `docs/swagger.yaml` запускайте `./generate-swagger.sh`: он вызывает `swaggo` и сохраняет необязательную авторизацию для публичных GET. Прямой `swag init` не выражает вариант запроса без токена.

| Метод | URL | Ответ | Описание | Функция |
|---|---|---|---|---|
| GET | `/api/methods` | 200 — `methods`, `min_time`, `max_time`, `total`, `page`, `limit` | Список опубликованных методов миграции; фильтр `?min_time={min}&max_time={max}`, пагинация `?page={page}&limit={limit}`. | `GetGrid` |
| GET | `/api/methods/feed` | 200 — `method` с `is_liked`, `feed_position`, `feed_size` | Лента из Redis: без параметров — первая карточка сессии; `?id={id}` — опубликованный метод по ID; `?id={id}&next=true` — следующая карточка. После 100 карточек генерируется новый набор. `is_liked` показывает лайк текущего пользователя. | `GetFeedItem` |
| GET | `/api/methods/draft` | 200 — `draft`; 404, если черновика нет | Черновик текущего пользователя. | `ShowAddMethodPage` |
| POST | `/api/methods` | 201 — `draft`, заголовок `Location` | Создает черновик из `multipart/form-data`: `title`, `image`, `video`. | `CreateDraftMethod` |
| PUT | `/api/methods/publish` | 200 — `method` | Публикует черновик; принимает JSON с `description`, `time_in_gb`, `reliability`. | `PublishDraftMethod` |
| DELETE | `/api/methods/{id}` | 204 — без тела | Помечает свой метод миграции как удаленный. | `SoftDeleteMethod` |
| POST | `/api/methods/{id}/like` | 200 — `method` с `id`, `is_liked`, `likes_count` | Ставит лайк при `{"like":1}`, снимает при `{"like":0}`. | `SetLike` |
| POST | `/api/users` | 201 — `id`, `email` | Регистрирует пользователя по `email` и `password`. | `Register` |
| POST | `/api/users/login` | 200 — `access_token`, `token_type`, `expires_in` | Вход по `email` и `password`. | `Authenticate` |
| POST | `/api/users/logout` | 204 — без тела | Добавляет access-токен в Redis-блэклист до истечения срока; требует Authorization: Bearer. | `Logout` |

## Таблицы и данные

Модели находятся в `internal/ds/models.go`, схема БД — в `migrations/init.sql`.

### `users`

| Поле | Тип | Ограничение | Описание |
|---|---|---|---|
| `id` | `INT GENERATED ALWAYS AS IDENTITY` | **PK** | ID пользователя |
| `email` | `VARCHAR(255)` | `NOT NULL`, уникальный индекс | Адрес электронной почты |
| `password` | `VARCHAR(255)` | `NOT NULL` | Пароль в открытом виде для учебного проекта |

### `migration_methods`

| Поле | Тип | Ограничение | Описание |
|---|---|---|---|
| `id` | `INT GENERATED ALWAYS AS IDENTITY` | **PK** | ID метода |
| `creator_id` | `INT` | `NOT NULL`, **FK** → `users.id`, `ON DELETE RESTRICT` | Автор метода |
| `title` | `VARCHAR(255)` | `NOT NULL`, уникальный индекс | Название |
| `description` | `TEXT` | Может быть `NULL` | Описание |
| `status` | `VARCHAR(20)` | `NOT NULL`, по умолчанию `draft`; только `draft`, `published`, `deleted` | Статус метода |
| `image_key` | `VARCHAR(255)` | Может быть `NULL` | Ключ изображения в MinIO |
| `video_key` | `VARCHAR(255)` | Может быть `NULL` | Ключ видео в MinIO |
| `time_in_gb` | `NUMERIC(10,2)` | Может быть `NULL` | Время на 1 Гб |
| `reliability` | `NUMERIC(5,4)` | Может быть `NULL` | Коэффициент надёжности |
| `created_at` | `TIMESTAMPTZ` | `NOT NULL`, по умолчанию текущее время | Дата создания |
| `published_at` | `TIMESTAMPTZ` | Может быть `NULL` | Дата публикации |

Частичный уникальный индекс по `creator_id` действует при `status = 'draft'`: у пользователя может быть не больше одного черновика.

### `migration_method_likes`

| Поле | Тип | Ограничение | Описание |
|---|---|---|---|
| `id` | `INT GENERATED ALWAYS AS IDENTITY` | **PK** | ID лайка |
| `user_id` | `INT` | `NOT NULL`, **FK** → `users.id`, `ON DELETE RESTRICT` | Кто поставил лайк |
| `method_id` | `INT` | `NOT NULL`, **FK** → `migration_methods.id`, `ON DELETE RESTRICT` | Какой метод миграции отмечен |

На пару (`user_id`, `method_id`) действует уникальный индекс: один пользователь может поставить методу миграции только один лайк.

В базе хранятся ключи медиафайлов, а сами файлы находятся в MinIO. При чтении метода миграции API формирует из ключей `image_url` и `video_url`.

Новые имена файлов генерируются из случайных латинских символов и цифр. Если сохранение в БД не удалось, сервис удаляет загруженные объекты. Для карточек без медиа используются локальные `/static/images/default.svg` и `/static/videos/default.mp4`.

## Пагинация и индексы

`GET /api/methods` принимает `page` (по умолчанию 1) и `limit` (по умолчанию 10, максимум 100) и возвращает `total`, `page`, `limit`.

Для фильтра по времени создан индекс:

```sql
CREATE INDEX IF NOT EXISTS idx_migration_methods_status_time_id
    ON migration_methods (status, time_in_gb, id);
```

Для существующей БД примените `migrations/002_indexes.sql`. Команды `EXPLAIN ANALYZE` для сравнения с индексом и без него описаны в `docs/indexes.md`.

## Генерация данных

```bash
go run ./cmd/seed
```

Скрипт доводит число опубликованных услуг до 100000, использует faker для названий и описаний, вставляет данные батчами и выполняет `ANALYZE`.

## Лента в Redis

Лента хранится в Redis в ключе `data-migration-backend:feed:{session}` и создаётся при первом запросе `GET /api/methods/feed`. Идентификатор сессии хранится в HttpOnly cookie `data_migration_feed_session`. В ленте 100 случайных неповторяющихся опубликованных услуг; после последней карточки создаётся новый набор. Позиция карточки возвращается в `feed_position`, размер набора — в `feed_size`.
