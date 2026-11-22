-- Индекс для фильтрации опубликованных услуг по времени на 1 Гб
-- и стабильной сортировки/пагинации по id.
CREATE INDEX IF NOT EXISTS idx_migration_methods_status_time_id
    ON migration_methods (status, time_in_gb, id);

ANALYZE migration_methods;
