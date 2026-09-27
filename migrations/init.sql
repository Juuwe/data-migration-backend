CREATE TABLE IF NOT EXISTS users (
    id BIGSERIAL PRIMARY KEY,
    email VARCHAR(255) NOT NULL,
    password VARCHAR(255) NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email
    ON users (email);

CREATE TABLE IF NOT EXISTS migration_methods (
    id BIGSERIAL PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    description TEXT,
    status VARCHAR(20) NOT NULL DEFAULT 'draft'
        CHECK (status IN ('draft', 'published', 'deleted')),
    image_key VARCHAR(255),
    video_key VARCHAR(255),
    time_in_gb NUMERIC(10, 2),
    reliability NUMERIC(5, 4),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    published_at TIMESTAMPTZ,
    creator_id BIGINT NOT NULL,
    CONSTRAINT fk_migration_methods_creator
        FOREIGN KEY (creator_id) REFERENCES users(id) ON DELETE RESTRICT
);

ALTER TABLE migration_methods
    ADD COLUMN IF NOT EXISTS image_key VARCHAR(255),
    ADD COLUMN IF NOT EXISTS video_key VARCHAR(255);

UPDATE migration_methods AS methods
SET image_key = COALESCE(NULLIF(methods.image_key, ''), regexp_replace(to_jsonb(methods)->>'image_url', '^.*/', '')),
    video_key = COALESCE(NULLIF(methods.video_key, ''), regexp_replace(to_jsonb(methods)->>'video_url', '^.*/', ''))
WHERE methods.image_key IS NULL OR methods.video_key IS NULL;

ALTER TABLE migration_methods
    DROP COLUMN IF EXISTS image_url,
    DROP COLUMN IF EXISTS video_url;

CREATE UNIQUE INDEX IF NOT EXISTS ux_migration_methods_creator_draft
    ON migration_methods (creator_id)
    WHERE status = 'draft';

CREATE UNIQUE INDEX IF NOT EXISTS idx_migration_methods_title
    ON migration_methods (title);

CREATE TABLE IF NOT EXISTS migration_method_likes (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    method_id BIGINT NOT NULL,
    CONSTRAINT fk_migration_method_likes_user
        FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE RESTRICT,
    CONSTRAINT fk_migration_method_likes_method
        FOREIGN KEY (method_id) REFERENCES migration_methods(id) ON DELETE RESTRICT
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_user_method
    ON migration_method_likes (user_id, method_id);

INSERT INTO users (id, email, password)
VALUES
    (1, 'student@example.com', '!'),
    (2, 'analyst@example.com', '!'),
    (3, 'engineer@example.com', '!'),
    (4, 'architect@example.com', '!'),
    (5, 'reviewer@example.com', '!')
ON CONFLICT DO NOTHING;

SELECT setval(pg_get_serial_sequence('users', 'id'), (SELECT MAX(id) FROM users), true);

WITH media(name, image_key, video_key) AS (
    VALUES
        ('online', 'online.png', 'online.mp4'),
        ('offline', 'offline.png', 'offline.mp4'),
        ('replication', 'replication.png', 'replication.mp4'),
        ('hybrid', 'hybrid.png', 'hybrid.mp4'),
        ('elt', 'elt.png', 'elt.mp4'),
        ('physical', 'physical.png', 'physical.mp4'),
        ('audit', 'audit.png', 'audit.mp4')
), seed(title, description, status, media_name, time_in_gb, reliability, email) AS (
    VALUES
        ('Онлайн-миграция', NULL, 'draft', 'online', NULL, NULL, 'student@example.com'),
        ('Репликация данных', 'Организация постоянной синхронизации между исходной и целевой инфраструктурой.', 'published', 'replication', 0.08, 0.9995, 'student@example.com'),
        ('Аудит и валидация', 'Проверка стратегии и результатов переноса данных.', 'published', 'audit', 0.10, 0.9900, 'student@example.com'),
        ('Потоковая миграция', 'Постепенный перенос новых записей без остановки источника.', 'published', 'online', 0.07, 0.9980, 'student@example.com'),
        ('Миграция без простоя', 'Переключение на целевую систему после синхронизации изменений.', 'published', 'hybrid', 0.11, 0.9997, 'student@example.com'),
        ('Синхронизация архивов', 'Перенос архивов с проверкой целостности каждого пакета.', 'published', 'replication', 0.18, 0.9960, 'student@example.com'),
        ('Проверка совместимости', 'Оценка схем и форматов перед переносом данных.', 'published', 'audit', 0.20, 0.9950, 'student@example.com'),
        ('Архивная миграция', 'Карточка логически удаленной услуги.', 'deleted', NULL, 0.50, 0.9500, 'student@example.com'),

        ('Офлайн-миграция', 'Пакетный перенос больших объемов в технологическое окно.', 'published', 'offline', 0.12, 0.9990, 'engineer@example.com'),
        ('Гибридная миграция', 'Поэтапный перенос с офлайн-загрузкой базового массива.', 'published', 'hybrid', 0.15, 0.9970, 'engineer@example.com'),
        ('Физическая миграция', 'Перенос больших массивов на защищенных накопителях.', 'published', 'physical', 0.05, 0.9999, 'engineer@example.com'),
        ('Черновик инженерной миграции', NULL, 'draft', 'physical', NULL, NULL, 'engineer@example.com'),

        ('ETL-миграция', 'Перенос данных с изменением формата и схемы.', 'published', 'elt', 0.25, 0.9950, 'analyst@example.com'),
        ('Преобразование схемы', 'Сопоставление и преобразование полей исходной базы.', 'published', 'elt', 0.22, 0.9940, 'analyst@example.com'),
        ('Очистка данных', 'Удаление дубликатов и нормализация перед переносом.', 'published', 'audit', 0.30, 0.9920, 'analyst@example.com'),
        ('Черновик аналитической миграции', NULL, 'draft', 'elt', NULL, NULL, 'analyst@example.com'),

        ('Миграция ЦОД', 'Перенос между центрами обработки данных.', 'published', 'offline', 0.16, 0.9980, 'architect@example.com'),
        ('Планирование перехода', 'Планирование этапов и контрольных точек миграции.', 'published', 'hybrid', 0.21, 0.9970, 'architect@example.com'),
        ('Резервная площадка', 'Синхронизация данных с резервной площадкой.', 'published', 'physical', 0.14, 0.9990, 'architect@example.com'),
        ('Черновик архитектурной миграции', NULL, 'draft', 'offline', NULL, NULL, 'architect@example.com'),

        ('Тестовый перенос', 'Пробный перенос набора данных перед основным запуском.', 'published', 'online', 0.28, 0.9910, 'reviewer@example.com'),
        ('Контроль качества', 'Сравнение результатов переноса с исходными данными.', 'published', 'audit', 0.19, 0.9960, 'reviewer@example.com'),
        ('Проверка отката', 'Проверка возврата к исходной системе после сбоя.', 'published', 'replication', 0.24, 0.9930, 'reviewer@example.com'),
        ('Черновик проверки миграции', NULL, 'draft', 'audit', NULL, NULL, 'reviewer@example.com')
)
INSERT INTO migration_methods (
    title, description, status, image_key, video_key,
    time_in_gb, reliability, published_at, creator_id
)
SELECT seed.title, seed.description, seed.status, media.image_key, media.video_key,
       seed.time_in_gb, seed.reliability,
       CASE WHEN seed.status = 'draft' THEN NULL ELSE NOW() END,
       users.id
FROM seed
JOIN users ON users.email = seed.email
LEFT JOIN media ON media.name = seed.media_name
ON CONFLICT DO NOTHING;

INSERT INTO migration_method_likes (user_id, method_id)
SELECT users.id, methods.id
FROM (
    VALUES
        ('student@example.com', 'Репликация данных'),
        ('analyst@example.com', 'Гибридная миграция'),
        ('architect@example.com', 'Гибридная миграция'),
        ('analyst@example.com', 'ETL-миграция'),
        ('engineer@example.com', 'ETL-миграция'),
        ('reviewer@example.com', 'ETL-миграция'),
        ('student@example.com', 'Физическая миграция'),
        ('analyst@example.com', 'Физическая миграция'),
        ('engineer@example.com', 'Физическая миграция'),
        ('architect@example.com', 'Физическая миграция'),
        ('student@example.com', 'Аудит и валидация'),
        ('analyst@example.com', 'Аудит и валидация'),
        ('engineer@example.com', 'Аудит и валидация'),
        ('architect@example.com', 'Аудит и валидация'),
        ('reviewer@example.com', 'Аудит и валидация'),
        ('student@example.com', 'Офлайн-миграция'),
        ('student@example.com', 'ETL-миграция'),
        ('student@example.com', 'Миграция ЦОД'),
        ('analyst@example.com', 'Потоковая миграция'),
        ('analyst@example.com', 'Планирование перехода'),
        ('engineer@example.com', 'Миграция без простоя'),
        ('engineer@example.com', 'Очистка данных'),
        ('architect@example.com', 'Потоковая миграция'),
        ('architect@example.com', 'Тестовый перенос'),
        ('reviewer@example.com', 'Миграция ЦОД'),
        ('reviewer@example.com', 'Синхронизация архивов')
) AS seed(email, title)
JOIN users ON users.email = seed.email
JOIN migration_methods methods ON methods.title = seed.title
ON CONFLICT (user_id, method_id) DO NOTHING;
