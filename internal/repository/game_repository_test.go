package repository

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pieremarie/chessmobile/internal/hub"
)

type gameRow struct {
	values []any
	err    error
}

func (r gameRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	for i, v := range r.values {
		switch p := dest[i].(type) {
		case *uuid.UUID:
			*p = v.(uuid.UUID)
		case **string:
			*p = v.(*string)
		case *string:
			*p = v.(string)
		case *int:
			*p = v.(int)
		}
	}
	return nil
}

type gameDBFake struct {
	row     pgx.Row
	tag     pgconn.CommandTag
	err     error
	queries []string
	args    [][]any
	tx      *gameTxFake
}

func (d *gameDBFake) QueryRow(_ context.Context, q string, args ...any) pgx.Row {
	d.queries = append(d.queries, q)
	d.args = append(d.args, args)
	return d.row
}
func (d *gameDBFake) Exec(_ context.Context, q string, args ...any) (pgconn.CommandTag, error) {
	d.queries = append(d.queries, q)
	d.args = append(d.args, args)
	return d.tag, d.err
}
func (d *gameDBFake) Begin(context.Context) (pgx.Tx, error) {
	if d.err != nil {
		return nil, d.err
	}
	return d.tx, nil
}

type gameTxFake struct {
	pgx.Tx
	row                   pgx.Row
	err                   error
	queries               []string
	args                  [][]any
	committed, rolledBack bool
}

func (d *gameTxFake) QueryRow(_ context.Context, q string, args ...any) pgx.Row {
	d.queries = append(d.queries, q)
	d.args = append(d.args, args)
	return d.row
}
func (d *gameTxFake) Exec(_ context.Context, q string, args ...any) (pgconn.CommandTag, error) {
	d.queries = append(d.queries, q)
	d.args = append(d.args, args)
	return pgconn.NewCommandTag("UPDATE 1"), d.err
}
func (d *gameTxFake) Commit(context.Context) error   { d.committed = true; return d.err }
func (d *gameTxFake) Rollback(context.Context) error { d.rolledBack = true; return nil }
func TestGameRepositoryReads(t *testing.T) {
	white, black, id := uuid.New(), uuid.New(), uuid.New()
	cases := []struct {
		name string
		row  pgx.Row
		want error
	}{
		{"active", gameRow{values: []any{white, black}}, nil},
		{"missing", gameRow{err: pgx.ErrNoRows}, hub.ErrNotFound},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db := &gameDBFake{row: tc.row}
			got, err := NewGameRepository(db).Players(context.Background(), id)
			if !errors.Is(err, tc.want) {
				t.Fatalf("error %v, want %v", err, tc.want)
			}
			if tc.want == nil && got != (hub.Players{White: white, Black: black}) {
				t.Fatalf("players: %+v", got)
			}
			if len(db.args) != 1 || db.args[0][0] != id {
				t.Fatal("wrong query arguments")
			}
		})
	}
}
func TestGameRepositoryMoveTransaction(t *testing.T) {
	id := uuid.New()
	cases := []struct {
		name       string
		row        pgx.Row
		execErr    error
		wantCommit bool
	}{
		{"saved", gameRow{values: []any{2}}, nil, true},
		{"missing", gameRow{err: pgx.ErrNoRows}, nil, false},
		{"insert fails", gameRow{values: []any{2}}, errors.New("db failure"), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tx := &gameTxFake{row: tc.row, err: tc.execErr}
			db := &gameDBFake{tx: tx}
			err := NewGameRepository(db).SaveMove(context.Background(), id, "e2e4", "fen after")
			if (err == nil) != tc.wantCommit {
				t.Fatalf("error=%v", err)
			}
			if tx.committed != tc.wantCommit || !tx.rolledBack {
				t.Fatalf("commit=%v rollback=%v", tx.committed, tx.rolledBack)
			}
			if tc.wantCommit {
				if len(tx.queries) != 2 || !strings.Contains(tx.queries[0], "RETURNING ply") || tx.args[0][1] != "fen after" || tx.args[1][2] != "e2e4" {
					t.Fatalf("queries=%v args=%v", tx.queries, tx.args)
				}
			}
		})
	}
}
func TestGameRepositoryResult(t *testing.T) {
	id := uuid.New()
	cases := []struct {
		name, result string
		tag          pgconn.CommandTag
		want         error
	}{{"saved", "1-0", pgconn.NewCommandTag("UPDATE 1"), nil}, {"finished", "0-1", pgconn.NewCommandTag("UPDATE 0"), hub.ErrFinished}, {"invalid", "other", pgconn.NewCommandTag(""), nil}}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db := &gameDBFake{tag: tc.tag}
			err := NewGameRepository(db).SaveResult(context.Background(), id, tc.result)
			if tc.name == "invalid" {
				if err == nil || len(db.queries) != 0 {
					t.Fatal("invalid result reached DB")
				}
				return
			}
			if !errors.Is(err, tc.want) {
				t.Fatalf("error=%v want=%v", err, tc.want)
			}
		})
	}
}
