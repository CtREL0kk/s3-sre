
## s3wrapper_uploads_total - количество публикаций
SLI: доля успешных публикаций \
SLO: >= 99,5% \
SLA: >= 99% \
Формула для vmui: 
```
100 * sum(increase(s3wrapper_uploads_total{namespace="app",app="app",status="success"}[5m])) / sum(increase(s3wrapper_uploads_total{namespace="app",app="app"}[5m]))
```

## s3wrapper_downloads_total - количество скачиваний
SLI: доля успешных скачиваний \
SLO: >= 99,5% \
SLA: >= 99% \
Формула для vmui:
```promql
100 * sum(increase(s3wrapper_downloads_total{namespace="app",app="app",status="success"}[5m])) / sum(increase(s3wrapper_downloads_total{namespace="app",app="app"}[5m]))
```

## s3wrapper_upload_duration_seconds - длительность ответа при публикации
SLI: p95 длительности успешных загрузок \
SLO: <= 1 секунды \
SLA: <= 2 секунд \
Формула для vmui:
```promql
histogram_quantile(0.95, sum by (le) (rate(s3wrapper_upload_duration_seconds_bucket{namespace="app",app="app",status="success"}[5m])))
```

## Пробер
Добавлен пробер, который загружает и скачивает файл `prober/bigtxtfile.txt`, проверяет содержимое и удаляет загруженный файл


## Скринкаст
На скринкасте скейлю проберы, чтобы увеличить нагрузку на сервис, ожидаю увеличение duration и соответственно изменение SLI. \
Переключаю режим на readonly, пробы upload будут падать, ожидаю падение SLI 
