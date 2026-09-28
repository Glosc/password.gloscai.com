package vault

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"testing"

	"github.com/Glosc/password.gloscai.com/server/database"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func TestSQLStoreIsolationAndAtomicRotation(t *testing.T) {
	dsn := os.Getenv("VAULT_TEST_DSN")
	if dsn == "" {
		t.Skip("set VAULT_TEST_DSN for PostgreSQL integration test")
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()
	if err := database.Migrate(ctx, db); err != nil {
		t.Fatal(err)
	}
	var alice, bob int64
	if err := db.QueryRowContext(ctx, `INSERT INTO users (sso_subject) VALUES ('vault-test-alice') RETURNING id`).Scan(&alice); err != nil {
		t.Fatal(err)
	}
	defer db.ExecContext(ctx, `DELETE FROM users WHERE id = $1`, alice)
	if err := db.QueryRowContext(ctx, `INSERT INTO users (sso_subject) VALUES ('vault-test-bob') RETURNING id`).Scan(&bob); err != nil {
		t.Fatal(err)
	}
	defer db.ExecContext(ctx, `DELETE FROM users WHERE id = $1`, bob)
	master, _ := randomKey()
	store, err := NewSQLStore(db, master)
	if err != nil {
		t.Fatal(err)
	}
	metadata := json.RawMessage(`{"v":1,"keyWrap":{"iv":"x","data":"y"}}`)
	if err := store.Initialize(ctx, alice, metadata); err != nil {
		t.Fatal(err)
	}
	if err := store.Initialize(ctx, bob, metadata); err != nil {
		t.Fatal(err)
	}
	id := "65c68698-970c-42eb-a232-9803cd79c674"
	oldPayload := `{"iv":"old","data":"ciphertext"}`
	version, err := store.Add(ctx, alice, []ItemInput{{ID: id, Payload: oldPayload}})
	if err != nil || version != 2 {
		t.Fatalf("add: version=%d error=%v", version, err)
	}
	others, err := store.List(ctx, bob)
	if err != nil || len(others) != 0 {
		t.Fatalf("cross-user list: %v, %v", others, err)
	}
	if _, err := store.Update(ctx, bob, ItemInput{ID: id, Payload: oldPayload}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-user update: %v", err)
	}
	if _, err := store.Rotate(ctx, alice, 1, metadata, []ItemInput{{ID: id, Payload: `{"iv":"new"}`}}); !errors.Is(err, ErrConflict) {
		t.Fatalf("stale version: %v", err)
	}
	items, err := store.List(ctx, alice)
	if err != nil || len(items) != 1 || items[0].Payload != oldPayload {
		t.Fatalf("rollback failed: %v, %v", items, err)
	}
	newPayload := `{"iv":"new","data":"new-ciphertext"}`
	version, err = store.Rotate(ctx, alice, 2, json.RawMessage(`{"v":2}`), []ItemInput{{ID: id, Payload: newPayload}})
	if err != nil || version != 3 {
		t.Fatalf("rotate: version=%d error=%v", version, err)
	}
	items, err = store.List(ctx, alice)
	if err != nil || items[0].Payload != newPayload {
		t.Fatalf("rotated item: %v, %v", items, err)
	}
	state, err := store.State(ctx, alice)
	if err != nil || state.Version != 3 || string(state.Metadata) != `{"v":2}` {
		t.Fatalf("rotated metadata: %+v, %v", state, err)
	}
}
