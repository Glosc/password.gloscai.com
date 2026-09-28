package vault

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"regexp"
	"strings"

	"github.com/Glosc/password.gloscai.com/server/sso"
)

type Store interface {
	State(context.Context, int64) (State, error)
	Initialize(context.Context, int64, json.RawMessage) error
	List(context.Context, int64) ([]Item, error)
	Add(context.Context, int64, []ItemInput) (int64, error)
	Update(context.Context, int64, ItemInput) (int64, error)
	Delete(context.Context, int64, string) (int64, error)
	Rotate(context.Context, int64, int64, json.RawMessage, []ItemInput) (int64, error)
	Reset(context.Context, int64) error
}

type Handler struct {
	store          Store
	allowedOrigins map[string]bool
}

func NewHandler(store Store, origins []string) *Handler {
	allowed := make(map[string]bool, len(origins))
	for _, origin := range origins {
		allowed[origin] = true
	}
	return &Handler{store: store, allowedOrigins: allowed}
}

func (h *Handler) Register(mux *http.ServeMux, requireUser func(http.Handler) http.Handler) {
	for _, route := range []struct {
		pattern string
		fn      http.HandlerFunc
	}{
		{"GET /api/v1/vault", h.state},
		{"POST /api/v1/vault", h.initialize},
		{"DELETE /api/v1/vault", h.reset},
		{"GET /api/v1/vault/items", h.list},
		{"POST /api/v1/vault/items", h.add},
		{"PUT /api/v1/vault/items/{id}", h.update},
		{"DELETE /api/v1/vault/items/{id}", h.delete},
		{"PUT /api/v1/vault/keys", h.rotate},
	} {
		mux.Handle(route.pattern, requireUser(h.security(route.fn)))
	}
}

func (h *Handler) security(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		if r.Method != http.MethodGet {
			origin := r.Header.Get("Origin")
			if origin != "" && !h.allowedOrigins[origin] && origin != "https://"+r.Host && origin != "http://"+r.Host {
				fail(w, http.StatusForbidden, "forbidden_origin", "request origin is not allowed")
				return
			}
			if r.Header.Get("X-Vault-Request") != "1" {
				fail(w, http.StatusForbidden, "csrf_required", "vault request header is required")
				return
			}
			if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
				fail(w, http.StatusUnsupportedMediaType, "invalid_content_type", "JSON body required")
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func userID(r *http.Request) int64 {
	user, _ := sso.UserFrom(r.Context())
	return user.ID
}

func (h *Handler) state(w http.ResponseWriter, r *http.Request) {
	state, err := h.store.State(r.Context(), userID(r))
	if err != nil {
		internal(w)
		return
	}
	respond(w, http.StatusOK, state)
}

func (h *Handler) initialize(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Metadata json.RawMessage `json:"metadata"`
	}
	if !decode(w, r, &input, 1<<16) {
		return
	}
	if !validMeta(input.Metadata) {
		fail(w, http.StatusBadRequest, "invalid_metadata", "invalid vault metadata")
		return
	}
	err := h.store.Initialize(r.Context(), userID(r), input.Metadata)
	if errors.Is(err, ErrConflict) {
		fail(w, http.StatusConflict, "already_initialized", "vault already initialized")
		return
	}
	if err != nil {
		internal(w)
		return
	}
	respond(w, http.StatusCreated, map[string]int64{"version": 1})
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	items, err := h.store.List(r.Context(), userID(r))
	if errors.Is(err, ErrNotFound) {
		fail(w, http.StatusNotFound, "not_found", "vault not initialized")
		return
	}
	if err != nil {
		internal(w)
		return
	}
	respond(w, http.StatusOK, items)
}

func (h *Handler) add(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Items []ItemInput `json:"items"`
	}
	if !decode(w, r, &input, 20<<20) {
		return
	}
	if len(input.Items) == 0 || len(input.Items) > 250 || !validItems(input.Items) {
		fail(w, http.StatusBadRequest, "invalid_items", "invalid encrypted items")
		return
	}
	version, err := h.store.Add(r.Context(), userID(r), input.Items)
	if handleStoreError(w, err) {
		return
	}
	respond(w, http.StatusCreated, map[string]int64{"version": version})
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !validID(id) {
		fail(w, http.StatusBadRequest, "invalid_id", "invalid item ID")
		return
	}
	var input struct {
		Payload string `json:"payload"`
	}
	if !decode(w, r, &input, 1<<18) {
		return
	}
	if !validPayload(input.Payload) {
		fail(w, http.StatusBadRequest, "invalid_payload", "invalid encrypted item")
		return
	}
	version, err := h.store.Update(r.Context(), userID(r), ItemInput{ID: id, Payload: input.Payload})
	if handleStoreError(w, err) {
		return
	}
	respond(w, http.StatusOK, map[string]int64{"version": version})
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !validID(id) {
		fail(w, http.StatusBadRequest, "invalid_id", "invalid item ID")
		return
	}
	version, err := h.store.Delete(r.Context(), userID(r), id)
	if handleStoreError(w, err) {
		return
	}
	respond(w, http.StatusOK, map[string]int64{"version": version})
}

func (h *Handler) rotate(w http.ResponseWriter, r *http.Request) {
	var input struct {
		ExpectedVersion int64           `json:"expected_version"`
		Metadata        json.RawMessage `json:"metadata"`
		Items           []ItemInput     `json:"items"`
	}
	if !decode(w, r, &input, 25<<20) {
		return
	}
	if input.ExpectedVersion < 1 || !validMeta(input.Metadata) || len(input.Items) > 5000 || !validItems(input.Items) {
		fail(w, http.StatusBadRequest, "invalid_rotation", "invalid rotation data")
		return
	}
	version, err := h.store.Rotate(r.Context(), userID(r), input.ExpectedVersion, input.Metadata, input.Items)
	if handleStoreError(w, err) {
		return
	}
	respond(w, http.StatusOK, map[string]int64{"version": version})
}

func (h *Handler) reset(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Confirm string `json:"confirm"`
	}
	if !decode(w, r, &input, 1024) {
		return
	}
	if input.Confirm != "DELETE" {
		fail(w, http.StatusBadRequest, "confirmation_required", "confirmation required")
		return
	}
	if err := h.store.Reset(r.Context(), userID(r)); err != nil {
		internal(w)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

var uuidPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

func validID(id string) bool { return uuidPattern.MatchString(id) }
func validPayload(payload string) bool {
	return len(payload) >= 20 && len(payload) <= 131072 && json.Valid([]byte(payload))
}
func validMeta(meta json.RawMessage) bool {
	return len(meta) >= 20 && len(meta) <= 16384 && json.Valid(meta)
}
func validItems(items []ItemInput) bool {
	seen := make(map[string]bool, len(items))
	for _, item := range items {
		if !validID(item.ID) || !validPayload(item.Payload) || seen[item.ID] {
			return false
		}
		seen[item.ID] = true
	}
	return true
}

func decode(w http.ResponseWriter, r *http.Request, target any, limit int64) bool {
	r.Body = http.MaxBytesReader(w, r.Body, limit)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		fail(w, http.StatusBadRequest, "invalid_request", "invalid JSON request")
		return false
	}
	if err := decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
		fail(w, http.StatusBadRequest, "invalid_request", "one JSON object required")
		return false
	}
	return true
}

func handleStoreError(w http.ResponseWriter, err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, ErrNotFound) {
		fail(w, http.StatusNotFound, "not_found", "vault or item not found")
		return true
	}
	if errors.Is(err, ErrConflict) {
		fail(w, http.StatusConflict, "version_conflict", "vault changed; reload and retry")
		return true
	}
	internal(w)
	return true
}

func internal(w http.ResponseWriter) {
	fail(w, http.StatusInternalServerError, "internal_error", "vault operation failed")
}
func respond(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{"data": data})
}
func fail(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{"code": code, "message": message}})
}
