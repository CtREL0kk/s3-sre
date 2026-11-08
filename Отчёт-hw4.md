### Использованы следующие компоненты:

- VictoriaMetrics Single - собирает и хранит метрики
- prometheus/client_golang - библиотека для экспорта метрик Go runtime и бизнес-метрик приложения на `/metrics`, порт 8081
- postgres-exporter — дополнительный контейнер в pod'е PostgreSQL, экспортирует метрики БД на порту 9187

### Бизнес-метрики:

- `s3wrapper_uploads_total` - количество попыток загрузки файлов с результатом success/error
- `s3wrapper_downloads_total` - количество попыток открытия файлов для скачивания с результатом success/error
- `s3wrapper_folders_created_total` - количество созданных папок
- `s3wrapper_shares_total` - количество успешных выдач и обновлений прав доступа
- `s3wrapper_upload_duration_seconds` - время загрузки файлов
- `s3wrapper_file_size_bytes` - распределение размеров загруженных файлов

VictoriaMetrics сама находит pod'ы по аннотациям `prometheus.io/` через Kubernetes service discovery и опрашивает их каждые 15 секунд. Собираются метрики двух реплик бэкенда, PostgreSQL exporter и самой VictoriaMetrics
