CREATE TABLE users (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TYPE method_status AS ENUM ('draft', 'published', 'deleted');

CREATE TABLE migration_methods (
    id BIGSERIAL PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    description TEXT,
    status method_status NOT NULL DEFAULT 'draft',
    image_key VARCHAR(512),
    video_key VARCHAR(512),
    time_in_gb NUMERIC(10,2) DEFAULT 0,
    reliability NUMERIC(5,4) DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    formed_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    creator_id BIGINT NOT NULL REFERENCES users(id) ON DELETE RESTRICT
);

CREATE TABLE migration_method_likes (
    id BIGSERIAL PRIMARY KEY,
    method_id BIGINT NOT NULL REFERENCES migration_methods(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    CONSTRAINT unique_user_method_like UNIQUE(method_id, user_id)
);
