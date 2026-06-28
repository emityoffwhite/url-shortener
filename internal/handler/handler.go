package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/emityoffwhite/url-shortener/internal/service"
	"github.com/emityoffwhite/url-shortener/internal/storage"
)

// Handler группирует HTTP-обработчики и их зависимости.
type Handler struct {
	service *service.URLService
	logger  *slog.Logger
}

// NewHandler создаёт Handler с переданным сервисом.
func NewHandler(svc *service.URLService, logger *slog.Logger) *Handler {
	return &Handler{service: svc, logger: logger}
}

type shortenRequest struct {
	URL        string `json:"url"`
	TTLSeconds int    `json:"ttl_seconds,omitempty"`
}

type shortenResponse struct {
	ShortCode string `json:"short_code"`
	ShortURL  string `json:"short_url"`
	Original  string `json:"original_url"`
}

type errorResponse struct {
	Error string `json:"error"`
}

// Routes возвращает настроенный http.Handler со всеми маршрутами.
// Используется встроенный в Go 1.22+ ServeMux с поддержкой методов в паттернах,
// что избавляет от необходимости тянуть внешний роутер для простого REST API.
func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /api/v1/shorten", h.handleShorten)
	mux.HandleFunc("GET /api/v1/stats/{code}", h.handleStats)
	mux.HandleFunc("DELETE /api/v1/{code}", h.handleDelete)
	mux.HandleFunc("GET /{code}", h.handleRedirect)
	mux.HandleFunc("GET /healthz", h.handleHealth)

	return h.withLogging(mux)
}

func (h *Handler) handleShorten(w http.ResponseWriter, r *http.Request) {
	var req shortenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	var ttl time.Duration
	if req.TTLSeconds > 0 {
		ttl = time.Duration(req.TTLSeconds) * time.Second
	}

	u, err := h.service.Shorten(r.Context(), req.URL, ttl)
	if err != nil {
		if errors.Is(err, service.ErrInvalidURL) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		h.logger.Error("shorten failed", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	resp := shortenResponse{
		ShortCode: u.ShortCode,
		ShortURL:  "http://" + r.Host + "/" + u.ShortCode,
		Original:  u.OriginalURL,
	}
	writeJSON(w, http.StatusCreated, resp)
}

func (h *Handler) handleRedirect(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")

	u, err := h.service.Resolve(r.Context(), code)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			writeError(w, http.StatusNotFound, "short url not found or expired")
			return
		}
		h.logger.Error("resolve failed", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	http.Redirect(w, r, u.OriginalURL, http.StatusFound)
}

func (h *Handler) handleStats(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")

	u, err := h.service.Resolve(r.Context(), code)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			writeError(w, http.StatusNotFound, "short url not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSON(w, http.StatusOK, u)
}

func (h *Handler) handleDelete(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")

	if err := h.service.Delete(r.Context(), code); err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			writeError(w, http.StatusNotFound, "short url not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) withLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		h.logger.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"duration", time.Since(start),
		)
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, errorResponse{Error: msg})
}
