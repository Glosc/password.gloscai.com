package tasks

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

var ErrNotFound = errors.New("task not found")

type SQLStore struct {
	db *sql.DB
}

func NewSQLStore(db *sql.DB) *SQLStore {
	return &SQLStore{db: db}
}

func (s *SQLStore) List(ctx context.Context) ([]Task, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, title, completed, created_at
		FROM tasks
		ORDER BY created_at DESC, id DESC
		LIMIT 100`)
	if err != nil {
		return nil, fmt.Errorf("listing tasks: %w", err)
	}
	defer rows.Close()

	items := make([]Task, 0)
	for rows.Next() {
		var task Task
		if err := rows.Scan(&task.ID, &task.Title, &task.Completed, &task.CreatedAt); err != nil {
			return nil, fmt.Errorf("scanning task: %w", err)
		}
		items = append(items, task)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("reading tasks: %w", err)
	}
	return items, nil
}

func (s *SQLStore) Create(ctx context.Context, title string) (Task, error) {
	var task Task
	err := s.db.QueryRowContext(ctx, `
		INSERT INTO tasks (title)
		VALUES ($1)
		RETURNING id, title, completed, created_at`, title).
		Scan(&task.ID, &task.Title, &task.Completed, &task.CreatedAt)
	if err != nil {
		return Task{}, fmt.Errorf("creating task: %w", err)
	}
	return task, nil
}

func (s *SQLStore) Lookup(ctx context.Context, id int64) (Task, error) {
	var task Task
	if err := s.db.QueryRowContext(ctx, "SELECT id, title, completed, created_at FROM tasks WHERE id = $1", id).Scan(&task.ID, &task.Title, &task.Completed, &task.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Task{}, ErrNotFound
		}
		return Task{}, fmt.Errorf("looking up task %d: %w", id, err)
	}
	return task, nil
}

func (s *SQLStore) SetCompleted(ctx context.Context, id int64, completed bool) (Task, error) {
	result, err := s.db.ExecContext(ctx, "UPDATE tasks SET completed = $1 WHERE id = $2", completed, id)
	if err != nil {
		return Task{}, fmt.Errorf("updating task %d: %w", id, err)
	}
	updated, err := result.RowsAffected()
	if err != nil {
		return Task{}, fmt.Errorf("reading updated row count: %w", err)
	}
	if updated == 0 {
		return Task{}, ErrNotFound
	}
	return s.Lookup(ctx, id)
}

func (s *SQLStore) Delete(ctx context.Context, id int64) error {
	result, err := s.db.ExecContext(ctx, "DELETE FROM tasks WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("deleting task %d: %w", id, err)
	}
	deleted, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("reading deleted row count: %w", err)
	}
	if deleted == 0 {
		return ErrNotFound
	}
	return nil
}
