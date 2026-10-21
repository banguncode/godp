package router

import (
	"net/http"

	"godp/internal/api/handler"
	"godp/internal/api/middleware"
	"godp/internal/config"
)

// SetupRouter constructs the HTTP mux and applies middleware.
func SetupRouter(cfg *config.Config) http.Handler {
	mux := http.NewServeMux()
	fmdHandler := handler.NewFMDHandler()

	// ── System / Health Probe ───────────────────────────────────────────────
	mux.HandleFunc("GET /health", handler.HandleHealth)

	// ── Biometric FMD Endpoints ─────────────────────────────────────────────
	mux.HandleFunc("POST /api/v1/fmd/extract", fmdHandler.HandleExtractFMD)
	mux.HandleFunc("POST /api/v1/fmd/enroll", fmdHandler.HandleEnroll)
	mux.HandleFunc("POST /api/v1/fmd/compare", fmdHandler.HandleCompare)
	mux.HandleFunc("POST /api/v1/fmd/identify", fmdHandler.HandleIdentify)
	mux.HandleFunc("POST /api/v1/fmd/convert", fmdHandler.HandleConvert)
	mux.HandleFunc("POST /api/v1/quality/nfiq", fmdHandler.HandleQuality)

	// ── Middleware Pipeline (Executed outer-to-inner) ──────────────────────
	var h http.Handler = mux
	h = middleware.Security(cfg)(h)
	h = middleware.CORS(cfg.CORSAllowOrigin)(h)
	h = middleware.Logger(h)
	h = middleware.Recovery(h)

	return h
}
