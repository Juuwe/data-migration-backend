# Сервис методов миграции данных

## HTTP-роуты

GET-запросы, создание и публикация черновика, постановка лайка и регистрация возвращают JSON. Остальные запросы и ошибки обработчиков API возвращают только HTTP-код без тела.

| Метод | URL | Ответ | Описание | Функция |
|---|---|---|---|---|
| GET | `/api/methods` | 200 — `methods`, `min_time`, `max_time` | Список опубликованных методов миграции; фильтр `?min_time={min}&max_time={max}`. | `GetGrid` |
| GET | `/api/methods/feed` | 200 — `method` | Первый опубликованный метод миграции. | `GetFeedItem` |
| GET | `/api/methods/feed/{id}` | 200 — `method` | Опубликованный метод миграции по ID. | `GetFeedItem` |
| GET | `/api/methods/feed/{id}?next=true` | 200 — `method` | Следующий метод миграции по ID; после последнего возвращается первый. | `GetFeedItem` |
| GET | `/api/methods/draft` | 200 — `draft`; 404, если черновика нет | Черновик текущего пользователя. | `ShowAddMethodPage` |
| POST | `/api/methods` | 201 — `draft`, заголовок `Location` | Создает черновик из `multipart/form-data`: `title`, `image`, `video`. | `CreateDraftMethod` |
| PUT | `/api/methods/publish` | 200 — `method` | Публикует черновик; принимает JSON с `description`, `time_in_gb`, `reliability`. | `PublishDraftMethod` |
| DELETE | `/api/methods/{id}` | 204 — без тела | Помечает свой метод миграции как удаленный. | `SoftDeleteMethod` |
| POST | `/api/methods/{id}/like` | 200 — `method` с `id`, `is_liked`, `likes_count` | Ставит лайк при `{"like":1}`, снимает при `{"like":0}`. | `SetLike` |
| POST | `/api/users` | 201 — `id`, `email` | Регистрирует пользователя по `email` и `password`. | `Register` |
| POST | `/api/users/login` | 501 — без тела | Заглушка входа. | `Authenticate` |
| POST | `/api/users/logout` | 501 — без тела | Заглушка выхода. | `Logout` |

## Таблицы и данные

Модели находятся в `internal/ds/models.go`, схема БД — в `migrations/init.sql`.

### `users`

| Поле | Тип | Ограничение | Описание |
|---|---|---|---|
| `id` | `INT GENERATED ALWAYS AS IDENTITY` | **PK** | ID пользователя |
| `email` | `VARCHAR(255)` | `NOT NULL`, уникальный индекс | Адрес электронной почты |
| `password` | `VARCHAR(255)` | `NOT NULL` | Пароль в переданном виде |

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
