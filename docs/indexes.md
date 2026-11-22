# Индексы и лента в Redis

## Индекс для фильтра по времени

Индекс создаётся в `migrations/002_indexes.sql`:

```sql
CREATE INDEX IF NOT EXISTS idx_migration_methods_status_time_id
    ON migration_methods (status, time_in_gb, id);
```

Он покрывает условие `status = 'published' AND time_in_gb BETWEEN ...`, которое использует список услуг и пагинация.

### Замер с индексом

```sql
ANALYZE migration_methods;

EXPLAIN (ANALYZE, BUFFERS)
SELECT id, title, time_in_gb
FROM migration_methods
WHERE status = 'published'
  AND time_in_gb BETWEEN 0.10 AND 0.11
ORDER BY id
LIMIT 10 OFFSET 0;
```

### Замер без индекса

```sql
DROP INDEX IF EXISTS idx_migration_methods_status_time_id;

EXPLAIN (ANALYZE, BUFFERS)
SELECT id, title, time_in_gb
FROM migration_methods
WHERE status = 'published'
  AND time_in_gb BETWEEN 0.10 AND 0.11
ORDER BY id
LIMIT 10 OFFSET 0;
```

### Вернуть индекс

```sql
CREATE INDEX idx_migration_methods_status_time_id
    ON migration_methods (status, time_in_gb, id);
ANALYZE migration_methods;
```

В плане запроса с индексом должен появиться `Index Scan` или `Bitmap Index Scan`, а `Execution Time` должен быть меньше, чем при `Seq Scan`.

Пример на 100000 услугах: `Seq Scan` — около 35 мс, `Index Only Scan` — около 1.3 мс.

## Генерация 100000 услуг

```bash
go run ./cmd/seed
```

Скрипт создаёт опубликованные услуги до общего количества `100000`, обновляет статистику `ANALYZE` и не дублирует данные при повторном запуске.

## Лента в Redis

Лента создаётся при первом запросе `GET /api/methods/feed`. В cookie `data_migration_feed_session` хранится идентификатор сессии.

Посмотреть ключи и размер ленты:

```bash
docker compose exec redis redis-cli KEYS 'data-migration-backend:feed:*'
docker compose exec redis redis-cli LLEN 'data-migration-backend:feed:<session>'
```

Первые 10 карточек с порядковыми номерами:

```bash
docker compose exec redis redis-cli LRANGE 'data-migration-backend:feed:<session>' 0 9
```

После просмотра 100 карточек создаётся новый набор из 100 неповторяющихся услуг. Позиция карточки возвращается в поле `feed_position`, общий размер — в `feed_size`.
