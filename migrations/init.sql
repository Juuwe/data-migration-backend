CREATE TABLE IF NOT EXISTS users (
    id BIGSERIAL PRIMARY KEY,
    email VARCHAR(255) NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS migration_methods (
    id BIGSERIAL PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    status VARCHAR(20) NOT NULL DEFAULT 'draft'
        CHECK (status IN ('draft', 'published', 'deleted')),
    image_key VARCHAR(512),
    video_key VARCHAR(512),
    time_in_gb NUMERIC(10, 2) NOT NULL DEFAULT 0,
    reliability NUMERIC(5, 4) NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    formed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    creator_id BIGINT NOT NULL,
    CONSTRAINT fk_migration_methods_creator
        FOREIGN KEY (creator_id) REFERENCES users(id) ON DELETE RESTRICT
);

CREATE UNIQUE INDEX IF NOT EXISTS ux_migration_methods_creator_draft
    ON migration_methods (creator_id)
    WHERE status = 'draft';

CREATE TABLE IF NOT EXISTS migration_method_likes (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    method_id BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_migration_method_likes_user
        FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE RESTRICT,
    CONSTRAINT fk_migration_method_likes_method
        FOREIGN KEY (method_id) REFERENCES migration_methods(id) ON DELETE RESTRICT,
    CONSTRAINT unique_user_method_like UNIQUE (user_id, method_id)
);

INSERT INTO users (email)
VALUES
    ('student@example.com'),
    ('analyst@example.com'),
    ('engineer@example.com')
ON CONFLICT (email) DO NOTHING;

INSERT INTO migration_methods (
    title,
    description,
    status,
    image_key,
    video_key,
    time_in_gb,
    reliability,
    creator_id
)
SELECT seed.title,
       seed.description,
       seed.status,
       seed.image_key,
       seed.video_key,
       seed.time_in_gb,
       seed.reliability,
       users.id
FROM (
    VALUES
        (
            'Онлайн-миграция',
            'Перенос данных в реальном времени с минимальным или нулевым временем простоя системы...',
            'published',
            'online.jpeg',
            'online.mp4',
            0.18,
            0.9980,
            'analyst@example.com'
        ),
        (
            'Офлайн-миграция',
            'Разовый пакетный перенос больших объемов данных во время планового технологического окна...',
            'published',
            'offline.jpeg',
            'offline.mp4',
            0.12,
            0.9990,
            'engineer@example.com'
        ),
        (
            'Репликация данных',
            'Организация постоянной синхронизации данных между исходной и целевой инфраструктурой...',
            'published',
            'replication.jpeg',
            'replication.mp4',
            0.08,
            0.9995,
            'student@example.com'
        ),
        (
            'Гибридная миграция',
            'Поэтапный перенос инфраструктуры, сочетающий офлайн-загрузку базового массива данных...',
            'published',
            'hybrid.jpeg',
            'hybrid.mp4',
            0.15,
            0.9970,
            'engineer@example.com'
        ),
        (
            'ETL-миграция',
            'Перенос данных с их параллельным изменением: заменой формата, изменением схемы базы данных...',
            'published',
            'etl.jpeg',
            'elt.mp4',
            0.25,
            0.9950,
            'analyst@example.com'
        ),
        (
            'Физическая миграция',
            'Перенос критически больших массивов данных с использованием физических защищенных накопителей...',
            'published',
            'appliance.jpeg',
            'physical.mp4',
            0.05,
            0.9999,
            'engineer@example.com'
        ),
        (
            'Аудит и валидация',
            'Комплексное сопровождение процесса миграции: от разработки стратегии до тестовой верификации...',
            'published',
            'audit.jpeg',
            'audit.mp4',
            0.10,
            0.9900,
            'student@example.com'
        ),
        (
            'Новый метод миграции',
            'Черновик описания...',
            'draft',
            NULL,
            NULL,
            0,
            0,
            'student@example.com'
        ),
        (
            'Архивная миграция',
            'Карточка логически удаленной услуги...',
            'deleted',
            NULL,
            NULL,
            0.50,
            0.9500,
            'student@example.com'
        )
) AS seed(title, description, status, image_key, video_key, time_in_gb, reliability, email)
JOIN users ON users.email = seed.email
WHERE NOT EXISTS (
    SELECT 1
    FROM migration_methods existing
    WHERE existing.title = seed.title
      AND existing.creator_id = users.id
);

INSERT INTO migration_method_likes (user_id, method_id)
SELECT users.id, methods.id
FROM (
    VALUES
        ('student@example.com', 'Онлайн-миграция'),
        ('analyst@example.com', 'Онлайн-миграция'),
        ('engineer@example.com', 'Офлайн-миграция'),
        ('student@example.com', 'Репликация данных'),
        ('analyst@example.com', 'ETL-миграция'),
        ('engineer@example.com', 'Физическая миграция')
) AS seed(email, title)
JOIN users ON users.email = seed.email
JOIN migration_methods methods ON methods.title = seed.title
ON CONFLICT (user_id, method_id) DO NOTHING;
