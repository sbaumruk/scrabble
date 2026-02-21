package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ── Types ────────────────────────────────────────────────────────────────────

type BoardMeta struct {
	ID         string    `json:"id"`
	UserID     *string   `json:"userId,omitempty"`
	Name       string    `json:"name"`
	ShareToken *string   `json:"shareToken,omitempty"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

type BoardRecord struct {
	BoardMeta
	Board []string `json:"board"` // 15 rows of 15 chars
}

// ── Database ─────────────────────────────────────────────────────────────────

type DB struct {
	pool *pgxpool.Pool
}

func NewDB(ctx context.Context, connStr string) (*DB, error) {
	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		return nil, fmt.Errorf("connect to database: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}
	return &DB{pool: pool}, nil
}

func (d *DB) Close() {
	d.pool.Close()
}

// Migrate creates the boards table and indexes if they don't already exist.
func (d *DB) Migrate(ctx context.Context) error {
	_, err := d.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS boards (
			id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id     TEXT,
			name        TEXT NOT NULL,
			board_data  TEXT NOT NULL,
			share_token TEXT UNIQUE,
			created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
		CREATE INDEX IF NOT EXISTS idx_boards_user_id ON boards(user_id);
		CREATE INDEX IF NOT EXISTS idx_boards_share_token ON boards(share_token);
	`)
	return err
}

// ── Board CRUD ───────────────────────────────────────────────────────────────

// ListBoards returns all boards belonging to the given user.
// If userID is empty, returns all boards (legacy/anonymous mode).
func (d *DB) ListBoards(ctx context.Context, userID string) ([]BoardMeta, error) {
	var query string
	var args []interface{}

	if userID != "" {
		query = `SELECT id, user_id, name, share_token, created_at, updated_at
			FROM boards WHERE user_id = $1 ORDER BY updated_at DESC`
		args = []interface{}{userID}
	} else {
		query = `SELECT id, user_id, name, share_token, created_at, updated_at
			FROM boards ORDER BY updated_at DESC`
	}

	rows, err := d.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var boards []BoardMeta
	for rows.Next() {
		var b BoardMeta
		if err := rows.Scan(&b.ID, &b.UserID, &b.Name, &b.ShareToken, &b.CreatedAt, &b.UpdatedAt); err != nil {
			return nil, err
		}
		boards = append(boards, b)
	}
	if boards == nil {
		boards = []BoardMeta{}
	}
	return boards, rows.Err()
}

// GetBoard loads a board by ID. If userID is non-empty, also checks ownership.
func (d *DB) GetBoard(ctx context.Context, id string, userID string) (*BoardRecord, error) {
	var b BoardRecord
	var boardData string
	err := d.pool.QueryRow(ctx,
		`SELECT id, user_id, name, board_data, share_token, created_at, updated_at
			FROM boards WHERE id = $1`, id,
	).Scan(&b.ID, &b.UserID, &b.Name, &boardData, &b.ShareToken, &b.CreatedAt, &b.UpdatedAt)
	if err != nil {
		return nil, err
	}

	// Ownership check: if userID provided, board must belong to that user
	if userID != "" && (b.UserID == nil || *b.UserID != userID) {
		return nil, fmt.Errorf("board not found")
	}

	b.Board = strings.Split(boardData, "\n")
	// Ensure exactly 15 rows
	for len(b.Board) < 15 {
		b.Board = append(b.Board, "...............")
	}
	b.Board = b.Board[:15]
	return &b, nil
}

// GetBoardByShareToken loads a board by its share token (public access).
func (d *DB) GetBoardByShareToken(ctx context.Context, token string) (*BoardRecord, error) {
	var b BoardRecord
	var boardData string
	err := d.pool.QueryRow(ctx,
		`SELECT id, user_id, name, board_data, share_token, created_at, updated_at
			FROM boards WHERE share_token = $1`, token,
	).Scan(&b.ID, &b.UserID, &b.Name, &boardData, &b.ShareToken, &b.CreatedAt, &b.UpdatedAt)
	if err != nil {
		return nil, err
	}
	b.Board = strings.Split(boardData, "\n")
	for len(b.Board) < 15 {
		b.Board = append(b.Board, "...............")
	}
	b.Board = b.Board[:15]
	return &b, nil
}

// SaveBoard updates a board's data. Checks ownership via userID.
func (d *DB) SaveBoard(ctx context.Context, id string, userID string, boardRows []string) error {
	boardData := strings.Join(boardRows, "\n")

	var n int64
	var err error
	if userID != "" {
		tag, e := d.pool.Exec(ctx,
			`UPDATE boards SET board_data = $1, updated_at = NOW()
				WHERE id = $2 AND user_id = $3`,
			boardData, id, userID)
		n, err = tag.RowsAffected(), e
	} else {
		tag, e := d.pool.Exec(ctx,
			`UPDATE boards SET board_data = $1, updated_at = NOW()
				WHERE id = $2`,
			boardData, id)
		n, err = tag.RowsAffected(), e
	}
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("board not found")
	}
	return nil
}

// CreateBoard inserts a new blank board and returns its ID.
func (d *DB) CreateBoard(ctx context.Context, name string, userID string) (string, error) {
	blankRows := make([]string, 15)
	for i := range blankRows {
		blankRows[i] = "..............."
	}
	boardData := strings.Join(blankRows, "\n")

	var id string
	var err error
	if userID != "" {
		err = d.pool.QueryRow(ctx,
			`INSERT INTO boards (name, user_id, board_data) VALUES ($1, $2, $3) RETURNING id`,
			name, userID, boardData,
		).Scan(&id)
	} else {
		err = d.pool.QueryRow(ctx,
			`INSERT INTO boards (name, board_data) VALUES ($1, $2) RETURNING id`,
			name, boardData,
		).Scan(&id)
	}
	return id, err
}

// DeleteBoard removes a board. Checks ownership via userID.
func (d *DB) DeleteBoard(ctx context.Context, id string, userID string) error {
	var n int64
	var err error
	if userID != "" {
		tag, e := d.pool.Exec(ctx,
			`DELETE FROM boards WHERE id = $1 AND user_id = $2`, id, userID)
		n, err = tag.RowsAffected(), e
	} else {
		tag, e := d.pool.Exec(ctx,
			`DELETE FROM boards WHERE id = $1`, id)
		n, err = tag.RowsAffected(), e
	}
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("board not found")
	}
	return nil
}

// SetShareToken generates and sets a share token for a board. Returns the token.
func (d *DB) SetShareToken(ctx context.Context, id string, userID string) (string, error) {
	token := generateShareToken()
	var n int64
	var err error
	if userID != "" {
		tag, e := d.pool.Exec(ctx,
			`UPDATE boards SET share_token = $1, updated_at = NOW()
				WHERE id = $2 AND user_id = $3`,
			token, id, userID)
		n, err = tag.RowsAffected(), e
	} else {
		tag, e := d.pool.Exec(ctx,
			`UPDATE boards SET share_token = $1, updated_at = NOW()
				WHERE id = $2`,
			token, id)
		n, err = tag.RowsAffected(), e
	}
	if err != nil {
		return "", err
	}
	if n == 0 {
		return "", fmt.Errorf("board not found")
	}
	return token, nil
}

// GetShareToken returns the existing share token for a board, if any.
func (d *DB) GetShareToken(ctx context.Context, id string, userID string) (*string, error) {
	var token *string
	var query string
	var args []interface{}

	if userID != "" {
		query = `SELECT share_token FROM boards WHERE id = $1 AND user_id = $2`
		args = []interface{}{id, userID}
	} else {
		query = `SELECT share_token FROM boards WHERE id = $1`
		args = []interface{}{id}
	}

	err := d.pool.QueryRow(ctx, query, args...).Scan(&token)
	if err != nil {
		return nil, err
	}
	return token, nil
}

// MigrateBoards imports board files from a directory into the database.
// Used for one-time migration of legacy file-based boards.
func (d *DB) MigrateBoards(ctx context.Context, boardsDir string, userID string) (int, error) {
	entries, err := readBoardDir(boardsDir)
	if err != nil {
		return 0, fmt.Errorf("read boards directory: %w", err)
	}

	count := 0
	for _, name := range entries {
		board, err := parseBoardFile(boardsDir + "/" + name + ".txt")
		if err != nil {
			fmt.Printf("  Skipping %s: %v\n", name, err)
			continue
		}
		boardRows := boardToStrings(board)
		boardData := strings.Join(boardRows, "\n")

		// Check if a board with this name already exists for this user
		var exists bool
		if userID != "" {
			err = d.pool.QueryRow(ctx,
				`SELECT EXISTS(SELECT 1 FROM boards WHERE name = $1 AND user_id = $2)`,
				name, userID).Scan(&exists)
		} else {
			err = d.pool.QueryRow(ctx,
				`SELECT EXISTS(SELECT 1 FROM boards WHERE name = $1 AND user_id IS NULL)`,
				name).Scan(&exists)
		}
		if err != nil {
			return count, err
		}
		if exists {
			fmt.Printf("  Skipping %s (already exists)\n", name)
			continue
		}

		if userID != "" {
			_, err = d.pool.Exec(ctx,
				`INSERT INTO boards (name, user_id, board_data) VALUES ($1, $2, $3)`,
				name, userID, boardData)
		} else {
			_, err = d.pool.Exec(ctx,
				`INSERT INTO boards (name, board_data) VALUES ($1, $2)`,
				name, boardData)
		}
		if err != nil {
			return count, fmt.Errorf("insert board %s: %w", name, err)
		}
		fmt.Printf("  Imported: %s\n", name)
		count++
	}
	return count, nil
}

// ── Helpers ──────────────────────────────────────────────────────────────────

func generateShareToken() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// readBoardDir returns board names (without .txt extension) from a directory.
func readBoardDir(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".txt") {
			names = append(names, strings.TrimSuffix(e.Name(), ".txt"))
		}
	}
	return names, nil
}
