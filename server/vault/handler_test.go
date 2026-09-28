package vault

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Glosc/password.gloscai.com/server/sso"
)

type fakeStore struct {
	users  []int64
	addErr error
}

func (s *fakeStore) State(_ context.Context, userID int64) (State, error) {
	s.users = append(s.users, userID)
	return State{Initialized: true, Version: 2, Metadata: json.RawMessage(`{"v":1}`)}, nil
}
func (s *fakeStore) Initialize(_ context.Context, userID int64, _ json.RawMessage) error {
	s.users = append(s.users, userID)
	return nil
}
func (s *fakeStore) List(_ context.Context, userID int64) ([]Item, error) {
	s.users = append(s.users, userID)
	return []Item{}, nil
}
func (s *fakeStore) Add(_ context.Context, userID int64, _ []ItemInput) (int64, error) {
	s.users = append(s.users, userID)
	return 3, s.addErr
}
func (s *fakeStore) Update(_ context.Context, userID int64, _ ItemInput) (int64, error) {
	s.users = append(s.users, userID)
	return 3, nil
}
func (s *fakeStore) Delete(_ context.Context, userID int64, _ string) (int64, error) {
	s.users = append(s.users, userID)
	return 3, nil
}
func (s *fakeStore) Rotate(_ context.Context, userID, _ int64, _ json.RawMessage, _ []ItemInput) (int64, error) {
	s.users = append(s.users, userID)
	return 3, ErrConflict
}
func (s *fakeStore) Reset(_ context.Context, userID int64) error {
	s.users = append(s.users, userID)
	return nil
}

func testMux(store Store) *http.ServeMux {
	mux := http.NewServeMux()
	NewHandler(store, []string{"https://password.example"}).Register(mux, func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("X-Test-User") == "" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			id := int64(12)
			if r.Header.Get("X-Test-User") == "other" {
				id = 99
			}
			next.ServeHTTP(w, r.WithContext(sso.WithUser(r.Context(), sso.User{ID: id})))
		})
	})
	return mux
}

func TestVaultRoutesRequireSessionAndScopeUser(t *testing.T) {
	store := &fakeStore{}
	mux := testMux(store)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/vault", nil)
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated status = %d", response.Code)
	}
	request.Header.Set("X-Test-User", "other")
	response = httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	if response.Code != http.StatusOK || len(store.users) != 1 || store.users[0] != 99 {
		t.Fatalf("user scope failed: %d, %v", response.Code, store.users)
	}
	if response.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("sensitive response may be cached")
	}
}

func TestVaultMutationRejectsCSRFAndConflicts(t *testing.T) {
	store := &fakeStore{addErr: ErrConflict}
	mux := testMux(store)
	body := `{"items":[{"id":"65c68698-970c-42eb-a232-9803cd79c674","payload":"{\"iv\":\"123\",\"data\":\"456\"}"}]}`
	request := httptest.NewRequest(http.MethodPost, "/api/v1/vault/items", strings.NewReader(body))
	request.Header.Set("X-Test-User", "yes")
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("missing CSRF header status = %d", response.Code)
	}
	request = httptest.NewRequest(http.MethodPost, "/api/v1/vault/items", strings.NewReader(body))
	request.Header.Set("X-Test-User", "yes")
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Vault-Request", "1")
	request.Header.Set("Origin", "https://evil.example")
	response = httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("forbidden origin status = %d", response.Code)
	}
	request.Header.Set("Origin", "https://password.example")
	response = httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	if response.Code != http.StatusConflict {
		t.Fatalf("conflict status = %d; body %s", response.Code, response.Body.String())
	}
}
