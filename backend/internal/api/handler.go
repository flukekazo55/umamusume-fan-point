package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"umamusume-fan-point/backend/internal/excel"
	"umamusume-fan-point/backend/internal/persistence"
)

type Loader interface {
	Load() (*excel.Workbook, error)
}

type Handler struct {
	loader  Loader
	players persistence.PlayerStore
	monthDB persistence.MonthStore
	ttl     time.Duration

	mu        sync.Mutex
	cached    *excel.Workbook
	cachedErr error
	expiresAt time.Time
}

func NewHandler(loader Loader, ttl time.Duration) *Handler {
	handler := &Handler{
		loader: loader,
		ttl:    ttl,
	}
	if players, ok := loader.(persistence.PlayerStore); ok {
		handler.players = players
	}
	if months, ok := loader.(persistence.MonthStore); ok {
		handler.monthDB = months
	}
	return handler
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/health", h.health)
	mux.HandleFunc("GET /api/months", h.months)
	mux.HandleFunc("POST /api/months", h.createMonth)
	mux.HandleFunc("GET /api/months/{id}", h.month)
	mux.HandleFunc("GET /api/months/{id}/export", h.exportMonth)
	mux.HandleFunc("PUT /api/months/{id}", h.updateMonth)
	mux.HandleFunc("DELETE /api/months/{id}", h.deleteMonth)
	mux.HandleFunc("GET /api/months/{monthID}/players", h.listPlayers)
	mux.HandleFunc("POST /api/months/{monthID}/players", h.createPlayer)
	mux.HandleFunc("GET /api/months/{monthID}/players/{name}", h.getPlayer)
	mux.HandleFunc("PUT /api/months/{monthID}/players/{name}", h.updatePlayer)
	mux.HandleFunc("DELETE /api/months/{monthID}/players/{name}", h.deletePlayer)
}

func (h *Handler) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) months(w http.ResponseWriter, _ *http.Request) {
	workbook, err := h.load()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, workbook)
}

func (h *Handler) month(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, errors.New("month id is required"))
		return
	}

	workbook, err := h.load()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	for _, month := range workbook.Months {
		if month.ID == id {
			writeJSON(w, http.StatusOK, month)
			return
		}
	}

	writeError(w, http.StatusNotFound, errors.New("month not found"))
}

func (h *Handler) exportMonth(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, errors.New("month id is required"))
		return
	}

	workbook, err := h.load()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	for _, month := range workbook.Months {
		if month.ID != id {
			continue
		}
		data, err := excel.ExportMonthWorkbook(month)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		fileName := strings.ReplaceAll(excel.ExportFileName(month), `"`, "")
		w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, fileName))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(data)
		return
	}

	writeError(w, http.StatusNotFound, errors.New("month not found"))
}

func (h *Handler) createMonth(w http.ResponseWriter, r *http.Request) {
	store, ok := h.monthStore(w)
	if !ok {
		return
	}

	var input excel.MonthInput
	if !decodeJSON(w, r, &input) {
		return
	}
	month, err := store.CreateMonth(r.Context(), input)
	if err != nil {
		h.writeStoreError(w, err)
		return
	}
	h.invalidate()
	writeJSON(w, http.StatusCreated, month)
}

func (h *Handler) updateMonth(w http.ResponseWriter, r *http.Request) {
	store, ok := h.monthStore(w)
	if !ok {
		return
	}

	var input excel.MonthInput
	if !decodeJSON(w, r, &input) {
		return
	}
	month, err := store.UpdateMonth(r.Context(), r.PathValue("id"), input)
	if err != nil {
		h.writeStoreError(w, err)
		return
	}
	h.invalidate()
	writeJSON(w, http.StatusOK, month)
}

func (h *Handler) deleteMonth(w http.ResponseWriter, r *http.Request) {
	store, ok := h.monthStore(w)
	if !ok {
		return
	}

	if err := store.DeleteMonth(r.Context(), r.PathValue("id")); err != nil {
		h.writeStoreError(w, err)
		return
	}
	h.invalidate()
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) listPlayers(w http.ResponseWriter, r *http.Request) {
	store, ok := h.playerStore(w)
	if !ok {
		return
	}

	players, err := store.ListPlayers(r.Context(), r.PathValue("monthID"))
	if err != nil {
		h.writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, players)
}

func (h *Handler) getPlayer(w http.ResponseWriter, r *http.Request) {
	store, ok := h.playerStore(w)
	if !ok {
		return
	}

	player, err := store.GetPlayer(r.Context(), r.PathValue("monthID"), r.PathValue("name"))
	if err != nil {
		h.writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, player)
}

func (h *Handler) createPlayer(w http.ResponseWriter, r *http.Request) {
	store, ok := h.playerStore(w)
	if !ok {
		return
	}

	var input excel.PlayerInput
	if !decodeJSON(w, r, &input) {
		return
	}
	player, err := store.CreatePlayer(r.Context(), r.PathValue("monthID"), input)
	if err != nil {
		h.writeStoreError(w, err)
		return
	}
	h.invalidate()
	writeJSON(w, http.StatusCreated, player)
}

func (h *Handler) updatePlayer(w http.ResponseWriter, r *http.Request) {
	store, ok := h.playerStore(w)
	if !ok {
		return
	}

	var input excel.PlayerInput
	if !decodeJSON(w, r, &input) {
		return
	}
	player, err := store.UpdatePlayer(r.Context(), r.PathValue("monthID"), r.PathValue("name"), input)
	if err != nil {
		h.writeStoreError(w, err)
		return
	}
	h.invalidate()
	writeJSON(w, http.StatusOK, player)
}

func (h *Handler) deletePlayer(w http.ResponseWriter, r *http.Request) {
	store, ok := h.playerStore(w)
	if !ok {
		return
	}

	if err := store.DeletePlayer(r.Context(), r.PathValue("monthID"), r.PathValue("name")); err != nil {
		h.writeStoreError(w, err)
		return
	}
	h.invalidate()
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) load() (*excel.Workbook, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if time.Now().Before(h.expiresAt) && (h.cached != nil || h.cachedErr != nil) {
		return h.cached, h.cachedErr
	}

	h.cached, h.cachedErr = h.loader.Load()
	h.expiresAt = time.Now().Add(h.ttl)
	return h.cached, h.cachedErr
}

func (h *Handler) invalidate() {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.cached = nil
	h.cachedErr = nil
	h.expiresAt = time.Time{}
}

func (h *Handler) playerStore(w http.ResponseWriter) (persistence.PlayerStore, bool) {
	if h.players == nil {
		writeError(w, http.StatusNotImplemented, errors.New("player CRUD requires MongoDB; set MONGO_URI to enable it"))
		return nil, false
	}
	return h.players, true
}

func (h *Handler) monthStore(w http.ResponseWriter) (persistence.MonthStore, bool) {
	if h.monthDB == nil {
		writeError(w, http.StatusNotImplemented, errors.New("month CRUD requires MongoDB; set MONGO_URI to enable it"))
		return nil, false
	}
	return h.monthDB, true
}

func (h *Handler) writeStoreError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, persistence.ErrNotFound):
		writeError(w, http.StatusNotFound, err)
	case errors.Is(err, persistence.ErrConflict):
		writeError(w, http.StatusConflict, err)
	default:
		writeError(w, http.StatusBadRequest, err)
	}
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dest any) bool {
	defer r.Body.Close()
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dest); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]string{"error": err.Error()})
}
