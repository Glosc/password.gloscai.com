package vault

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"
)

var (
	ErrNotFound = errors.New("vault or item not found")
	ErrConflict = errors.New("vault version conflict")
)

type State struct {
	Initialized bool            `json:"initialized"`
	Metadata    json.RawMessage `json:"metadata,omitempty"`
	Version     int64           `json:"version,omitempty"`
}

type Item struct {
	ID        string    `json:"id"`
	Payload   string    `json:"payload"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type ItemInput struct {
	ID      string `json:"id"`
	Payload string `json:"payload"`
}

type SQLStore struct {
	db     *sql.DB
	master []byte
}

func NewSQLStore(db *sql.DB, master []byte) (*SQLStore, error) {
	if len(master) != 32 {
		return nil, fmt.Errorf("vault master key must be 32 bytes")
	}
	return &SQLStore{db: db, master: append([]byte(nil), master...)}, nil
}

func (s *SQLStore) State(ctx context.Context, userID int64) (State, error) {
	var keyCipher, metaCipher []byte
	var version int64
	err := s.db.QueryRowContext(ctx, `SELECT server_key_cipher, metadata_cipher, version FROM vaults WHERE user_id = $1`, userID).Scan(&keyCipher, &metaCipher, &version)
	if errors.Is(err, sql.ErrNoRows) {
		return State{Initialized: false}, nil
	}
	if err != nil {
		return State{}, err
	}
	key, err := s.unwrapKey(userID, keyCipher)
	if err != nil {
		return State{}, err
	}
	meta, err := open(key, metaCipher, metaAAD(userID))
	if err != nil {
		return State{}, err
	}
	return State{Initialized: true, Metadata: meta, Version: version}, nil
}

func (s *SQLStore) Initialize(ctx context.Context, userID int64, metadata json.RawMessage) error {
	key, err := randomKey()
	if err != nil {
		return err
	}
	keyCipher, err := seal(s.master, key, keyAAD(userID))
	if err != nil {
		return err
	}
	metaCipher, err := seal(key, metadata, metaAAD(userID))
	if err != nil {
		return err
	}
	var inserted int64
	err = s.db.QueryRowContext(ctx, `INSERT INTO vaults (user_id, server_key_cipher, metadata_cipher) VALUES ($1, $2, $3) ON CONFLICT DO NOTHING RETURNING user_id`, userID, keyCipher, metaCipher).Scan(&inserted)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrConflict
	}
	return err
}

func (s *SQLStore) List(ctx context.Context, userID int64) ([]Item, error) {
	key, err := s.loadKey(ctx, userID)
	if err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx, `SELECT id, payload_cipher, created_at, updated_at FROM vault_items WHERE user_id = $1 ORDER BY updated_at DESC, id`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]Item, 0)
	for rows.Next() {
		var item Item
		var ciphertext []byte
		if err := rows.Scan(&item.ID, &ciphertext, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		payload, err := open(key, ciphertext, itemAAD(userID, item.ID))
		if err != nil {
			return nil, err
		}
		item.Payload = string(payload)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *SQLStore) Add(ctx context.Context, userID int64, inputs []ItemInput) (int64, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	key, version, err := s.lockedKey(ctx, tx, userID)
	if err != nil {
		return 0, err
	}
	for _, input := range inputs {
		ciphertext, err := seal(key, []byte(input.Payload), itemAAD(userID, input.ID))
		if err != nil {
			return 0, err
		}
		result, err := tx.ExecContext(ctx, `INSERT INTO vault_items (user_id, id, payload_cipher) VALUES ($1, $2, $3) ON CONFLICT DO NOTHING`, userID, input.ID, ciphertext)
		if err != nil {
			return 0, err
		}
		n, err := result.RowsAffected()
		if err != nil || n != 1 {
			return 0, ErrConflict
		}
	}
	if _, err := tx.ExecContext(ctx, `UPDATE vaults SET version = version + 1, updated_at = now() WHERE user_id = $1`, userID); err != nil {
		return 0, err
	}
	return version + 1, tx.Commit()
}

func (s *SQLStore) Update(ctx context.Context, userID int64, input ItemInput) (int64, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	key, version, err := s.lockedKey(ctx, tx, userID)
	if err != nil {
		return 0, err
	}
	ciphertext, err := seal(key, []byte(input.Payload), itemAAD(userID, input.ID))
	if err != nil {
		return 0, err
	}
	result, err := tx.ExecContext(ctx, `UPDATE vault_items SET payload_cipher = $3, updated_at = now() WHERE user_id = $1 AND id = $2`, userID, input.ID, ciphertext)
	if err != nil {
		return 0, err
	}
	n, err := result.RowsAffected()
	if err != nil || n != 1 {
		return 0, ErrNotFound
	}
	if _, err := tx.ExecContext(ctx, `UPDATE vaults SET version = version + 1, updated_at = now() WHERE user_id = $1`, userID); err != nil {
		return 0, err
	}
	return version + 1, tx.Commit()
}

func (s *SQLStore) Delete(ctx context.Context, userID int64, id string) (int64, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	_, version, err := s.lockedKey(ctx, tx, userID)
	if err != nil {
		return 0, err
	}
	result, err := tx.ExecContext(ctx, `DELETE FROM vault_items WHERE user_id = $1 AND id = $2`, userID, id)
	if err != nil {
		return 0, err
	}
	n, err := result.RowsAffected()
	if err != nil || n != 1 {
		return 0, ErrNotFound
	}
	if _, err := tx.ExecContext(ctx, `UPDATE vaults SET version = version + 1, updated_at = now() WHERE user_id = $1`, userID); err != nil {
		return 0, err
	}
	return version + 1, tx.Commit()
}

func (s *SQLStore) Rotate(ctx context.Context, userID, expected int64, metadata json.RawMessage, inputs []ItemInput) (int64, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	key, version, err := s.lockedKey(ctx, tx, userID)
	if err != nil {
		return 0, err
	}
	if version != expected {
		return 0, ErrConflict
	}
	var count int
	if err := tx.QueryRowContext(ctx, `SELECT count(*) FROM vault_items WHERE user_id = $1`, userID).Scan(&count); err != nil {
		return 0, err
	}
	if count != len(inputs) {
		return 0, ErrConflict
	}
	for _, input := range inputs {
		ciphertext, err := seal(key, []byte(input.Payload), itemAAD(userID, input.ID))
		if err != nil {
			return 0, err
		}
		result, err := tx.ExecContext(ctx, `UPDATE vault_items SET payload_cipher = $3, updated_at = now() WHERE user_id = $1 AND id = $2`, userID, input.ID, ciphertext)
		if err != nil {
			return 0, err
		}
		n, err := result.RowsAffected()
		if err != nil || n != 1 {
			return 0, ErrConflict
		}
	}
	metaCipher, err := seal(key, metadata, metaAAD(userID))
	if err != nil {
		return 0, err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE vaults SET metadata_cipher = $2, version = version + 1, updated_at = now() WHERE user_id = $1`, userID, metaCipher); err != nil {
		return 0, err
	}
	return version + 1, tx.Commit()
}

func (s *SQLStore) Reset(ctx context.Context, userID int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM vaults WHERE user_id = $1`, userID)
	return err
}

func (s *SQLStore) loadKey(ctx context.Context, userID int64) ([]byte, error) {
	var wrapped []byte
	if err := s.db.QueryRowContext(ctx, `SELECT server_key_cipher FROM vaults WHERE user_id = $1`, userID).Scan(&wrapped); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return s.unwrapKey(userID, wrapped)
}

func (s *SQLStore) lockedKey(ctx context.Context, tx *sql.Tx, userID int64) ([]byte, int64, error) {
	var wrapped []byte
	var version int64
	if err := tx.QueryRowContext(ctx, `SELECT server_key_cipher, version FROM vaults WHERE user_id = $1 FOR UPDATE`, userID).Scan(&wrapped, &version); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, 0, ErrNotFound
		}
		return nil, 0, err
	}
	key, err := s.unwrapKey(userID, wrapped)
	return key, version, err
}

func (s *SQLStore) unwrapKey(userID int64, wrapped []byte) ([]byte, error) {
	return open(s.master, wrapped, keyAAD(userID))
}

func keyAAD(userID int64) string  { return "vault-key:" + strconv.FormatInt(userID, 10) }
func metaAAD(userID int64) string { return "vault-meta:" + strconv.FormatInt(userID, 10) }
func itemAAD(userID int64, id string) string {
	return "vault-item:" + strconv.FormatInt(userID, 10) + ":" + id
}
