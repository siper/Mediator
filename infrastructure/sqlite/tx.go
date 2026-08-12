package sqlite

import (
	"context"
	"database/sql"

	"stersh.ru/mediator/domain"
)

type DBTX interface {
	Exec(query string, args ...any) (sql.Result, error)
	Query(query string, args ...any) (*sql.Rows, error)
	QueryRow(query string, args ...any) *sql.Row
}

type TxManager struct {
	db *sql.DB
}

func NewTxManager(db *sql.DB) *TxManager {
	return &TxManager{db: db}
}

func (m *TxManager) Run(ctx context.Context, fn func(*domain.Repos) error) (err error) {
	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
		if err != nil {
			_ = tx.Rollback()
			return
		}
		err = tx.Commit()
	}()

	repos := &domain.Repos{
		Media:   NewSQLiteMediaRepository(tx),
		Part:    NewSQLitePartRepository(tx),
		Group:   NewSQLitePartGroupRepository(tx),
		History: NewSQLiteHistoryRepository(tx),
		Queue:   NewSQLiteQueueRepository(tx),
	}
	return fn(repos)
}

var _ domain.UnitOfWork = (*TxManager)(nil)
