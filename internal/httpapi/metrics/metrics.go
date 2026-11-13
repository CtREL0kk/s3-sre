package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type S3WrapperMetrics struct {
	UploadsTotal          *prometheus.CounterVec
	DownloadsTotal        *prometheus.CounterVec
	SharesTotal           prometheus.Counter
	FoldersTotal          prometheus.Counter
	UploadDurationSeconds *prometheus.HistogramVec
	FileSizeBytes         prometheus.Histogram
}

func NewS3WrapperMetrics() *S3WrapperMetrics {
	const namespace = "s3wrapper"
	m := &S3WrapperMetrics{
		UploadsTotal:          promauto.NewCounterVec(prometheus.CounterOpts{Namespace: namespace, Name: "uploads_total", Help: "File upload attempts by result"}, []string{"status"}),
		DownloadsTotal:        promauto.NewCounterVec(prometheus.CounterOpts{Namespace: namespace, Name: "downloads_total", Help: "File open attempts by result"}, []string{"status"}),
		SharesTotal:           promauto.NewCounter(prometheus.CounterOpts{Namespace: namespace, Name: "shares_total", Help: "Successful grant creations or updates"}),
		FoldersTotal:          promauto.NewCounter(prometheus.CounterOpts{Namespace: namespace, Name: "folders_created_total", Help: "Successfully created folders"}),
		UploadDurationSeconds: promauto.NewHistogramVec(prometheus.HistogramOpts{Namespace: namespace, Name: "upload_duration_seconds", Help: "Duration of upload service operations", Buckets: prometheus.DefBuckets}, []string{"status"}),
		FileSizeBytes:         promauto.NewHistogram(prometheus.HistogramOpts{Namespace: namespace, Name: "file_size_bytes", Help: "Declared sizes of successfully uploaded files", Buckets: prometheus.ExponentialBuckets(1024, 4, 11)}),
	}
	for _, status := range []string{"success", "error"} {
		m.UploadsTotal.WithLabelValues(status)
		m.DownloadsTotal.WithLabelValues(status)
		m.UploadDurationSeconds.WithLabelValues(status)
	}
	return m
}

func result(success bool) string {
	if success {
		return "success"
	}
	return "error"
}

func (m *S3WrapperMetrics) RecordUpload(success bool, size int64, duration time.Duration) {
	status := result(success)
	m.UploadsTotal.WithLabelValues(status).Inc()
	m.UploadDurationSeconds.WithLabelValues(status).Observe(duration.Seconds())
	if success {
		m.FileSizeBytes.Observe(float64(size))
	}
}

func (m *S3WrapperMetrics) RecordDownload(success bool) {
	m.DownloadsTotal.WithLabelValues(result(success)).Inc()
}
func (m *S3WrapperMetrics) RecordFolder() { m.FoldersTotal.Inc() }
func (m *S3WrapperMetrics) RecordShare()  { m.SharesTotal.Inc() }
