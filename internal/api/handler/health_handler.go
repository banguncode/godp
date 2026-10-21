package handler

import (
	"net/http"
	"runtime"
	"time"

	"godp/internal/api/dto"
)

var startTime = time.Now()

// HandleHealth provides a liveness and readiness probe for container orchestrators and monitoring tools.
func HandleHealth(w http.ResponseWriter, r *http.Request) {
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)

	dto.JSON(w, http.StatusOK, map[string]interface{}{
		"status":      "UP",
		"service":     "godp-biometric-engine",
		"version":     "1.0.0",
		"uptime_sec":  int64(time.Since(startTime).Seconds()),
		"go_version":  runtime.Version(),
		"goroutines":  runtime.NumGoroutine(),
		"memory_alloc_mb": float64(mem.Alloc) / 1024 / 1024,
	})
}
