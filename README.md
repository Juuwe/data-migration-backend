# Сервис методов миграции данных

## HTTP-роуты

GET-запросы, создание и публикация черновика, постановка лайка, регистрация и вход возвращают JSON. Удаление, выход и ошибки API возвращают только HTTP-код без тела. Защищённые маршруты без действительного токена возвращают 401.

Swagger UI доступен по адресу `http://localhost:8081/swagger/index.html`. Для обновления `docs/swagger.yaml` запускайте `./generate-swagger.sh`: он вызывает `swaggo` и сохраняет необязательную авторизацию для публичных GET. Прямой `swag init` не выражает вариант запроса без токена.

| Метод | URL | Ответ | Описание | Функция |
|---|---|---|---|---|
| GET | `/api/methods` | 200 — `methods`, `min_time`, `max_time` | Список опубликованных методов миграции; фильтр `?min_time={min}&max_time={max}`. | `GetGrid` |
| GET | `/api/methods/feed` | 200 — `method` с `is_liked` (`bool`) | Без параметров — первый опубликованный метод; `?id={id}` — опубликованный метод по ID; `?id={id}&next=true` — следующий после него (с переходом к первому после последнего). `is_liked` показывает лайк текущего пользователя. | `GetFeedItem` |
| GET | `/api/methods/draft` | 200 — `draft`; 404, если черновика нет | Черновик текущего пользователя. | `ShowAddMethodPage` |
| POST | `/api/methods` | 201 — `draft`, заголовок `Location` | Создает черновик из `multipart/form-data`: `title`, `image`, `video`. | `CreateDraftMethod` |
| PUT | `/api/methods/publish` | 200 — `method` | Публикует черновик; принимает JSON с `description`, `time_in_gb`, `reliability`. | `PublishDraftMethod` |
| DELETE | `/api/methods/{id}` | 204 — без тела | Помечает свой метод миграции как удаленный. | `SoftDeleteMethod` |
| POST | `/api/methods/{id}/like` | 200 — `method` с `id`, `is_liked`, `likes_count` | Ставит лайк при `{"like":1}`, снимает при `{"like":0}`. | `SetLike` |
| POST | `/api/users` | 201 — `id`, `email` | Регистрирует пользователя по `email` и `password`. | `Register` |
| POST | `/api/users/login` | 200 — `access_token`, `token_type`, `expires_in`, `refresh_token` | Вход по `email` и `password`. | `Authenticate` |
| POST | `/api/users/refresh` | 200 — новая пара токенов | Обновляет токены по `refresh_token`; старый одноразовый. | `Refresh` |
| POST | `/api/users/logout` | 204 — без тела | Отзывает предъявленные access- и refresh-токены; требует Bearer и JSON с `refresh_token`. | `Logout` |

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
