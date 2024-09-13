package driver

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
)

type TransactionManager struct {
	conn   PostgresPool
	logger *zap.Logger
}

func NewTransactionManager(conn PostgresPool, logger *zap.Logger) *TransactionManager {
	return &TransactionManager{
		conn:   conn,
		logger: logger,
	}
}

func (m *TransactionManager) ExecuteTransaction(ctx context.Context, fn func(tx pgx.Tx) error) error {
	return m.ExecuteTransactionWithOptions(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead}, fn)
}

func (m *TransactionManager) ExecuteSerializableTransaction(ctx context.Context, fn func(tx pgx.Tx) error) error {
	return m.ExecuteTransactionWithRetry(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable}, fn, 3)
}

func (m *TransactionManager) ExecuteTransactionWithOptions(ctx context.Context, opts pgx.TxOptions, fn func(tx pgx.Tx) error) error {
	dbTx, err := m.conn.BeginTx(ctx, opts)
	if err != nil {
		return fmt.Errorf("begin transaction failed: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			m.rollback(ctx, dbTx)
			m.logger.Error("panic in transaction", zap.Any("panic", p))
			panic(p) // re-throw panic after Rollback
		} else if err != nil {
			m.rollback(ctx, dbTx)
		} else {
			if err = dbTx.Commit(ctx); err != nil {
				m.logger.Error("commit transaction failed", zap.Error(err))
			}
		}
	}()

	return fn(dbTx)
}

func (m *TransactionManager) ExecuteTransactionWithRetry(ctx context.Context, opts pgx.TxOptions, fn func(tx pgx.Tx) error, maxRetries int) error {
	var err error
	for i := 0; i < maxRetries; i++ {
		if err = m.ExecuteTransactionWithOptions(ctx, opts, fn); err == nil {
			return nil
		}
		if !m.isRetryableError(err) {
			return err
		}
		m.logger.Warn("Transaction failed, retrying", zap.Int("attempt", i+1), zap.Error(err))
		time.Sleep(time.Duration(i*100) * time.Millisecond) // 簡單的退避策略
	}
	return fmt.Errorf("transaction failed after %d attempts: %w", maxRetries, err)
}

func (m *TransactionManager) rollback(ctx context.Context, tx pgx.Tx) {
	if err := tx.Rollback(ctx); err != nil {
		m.logger.Error("rollback failed", zap.Error(err))
	}
}

func (m *TransactionManager) isRetryableError(err error) bool {
	// 這裡需要根據具體的錯誤類型來判斷是否可以重試
	// 例如，可以檢查是否是由於並發衝突導致的錯誤
	return true // 這裡簡化處理，實際使用時需要更精確的判斷
}
