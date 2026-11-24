-- Короткие английские описания для CLIP-поиска по изображению (50–100 символов).
ALTER TABLE migration_methods
    ADD COLUMN IF NOT EXISTS description_en VARCHAR(255);

-- Сид-услуги из migrations/init.sql: описания на русском, английские задаются явно.
UPDATE migration_methods AS methods
SET description_en = seed.description_en
FROM (
    VALUES
        ('Онлайн-миграция', 'Online data migration with minimal service downtime.'),
        ('Репликация данных', 'Continuous synchronization between source and target systems.'),
        ('Аудит и валидация', 'Validation of migration strategy and transferred data quality.'),
        ('Потоковая миграция', 'Gradual transfer of new records without stopping the source.'),
        ('Миграция без простоя', 'Switch to the target system after synchronizing changes.'),
        ('Синхронизация архивов', 'Archive transfer with integrity checks for every data package.'),
        ('Проверка совместимости', 'Assessment of schemas and formats before data migration.'),
        ('Архивная миграция', 'Legacy data transfer from an archived information system.'),
        ('Офлайн-миграция', 'Batch transfer of large data volumes during a downtime window.'),
        ('Гибридная миграция', 'Phased migration with offline loading of the base dataset.'),
        ('Физическая миграция', 'Transfer of large datasets on secure physical storage devices.'),
        ('ETL-миграция', 'Data transfer with format and schema transformation.'),
        ('Преобразование схемы', 'Mapping and transformation of source database fields.'),
        ('Очистка данных', 'Duplicate removal and normalization before data transfer.'),
        ('Миграция ЦОД', 'Data transfer between data centers and cloud regions.'),
        ('Планирование перехода', 'Planning migration stages and control checkpoints.'),
        ('Резервная площадка', 'Data synchronization with a disaster recovery site.'),
        ('Тестовый перенос', 'Trial data transfer before the main migration run.'),
        ('Контроль качества', 'Comparison of migrated data with the original source.'),
        ('Проверка отката', 'Rollback verification after a failed migration attempt.')
) AS seed(title, description_en)
WHERE methods.title = seed.title
  AND methods.description_en IS NULL;

-- Сгенерированные сидом строки уже содержат английское описание в description.
UPDATE migration_methods
SET description_en = btrim(description)
WHERE description_en IS NULL
  AND description ~ '^[ -~]{50,100}$';
