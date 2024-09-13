// Package driver
package driver

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresPool is an interface that represents a connection pool to a driver.
type PostgresPool interface {
	// Acquire returns a connection from the pool.
	Acquire(ctx context.Context) (*pgxpool.Conn, error)

	// BeginTx starts a new transaction and returns a Tx.
	BeginTx(ctx context.Context, txOptions pgx.TxOptions) (pgx.Tx, error)

	// Exec executes an SQL command and returns the command tag.
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)

	// Query executes an SQL query and returns the resulting rows.
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)

	// QueryRow executes an SQL query and returns a single row.
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row

	// SendBatch sends a batch of queries to the server. The batch is executed as a single transaction.
	SendBatch(ctx context.Context, batch *pgx.Batch) pgx.BatchResults

	// Close closes the pool and all its connections.
	Close()
}

type PostgresTx interface {
	// Begin starts a pseudo nested transaction.
	Begin(ctx context.Context) (pgx.Tx, error)

	// Commit commits the transaction if this is a real transaction or releases the savepoint if this is a pseudo nested
	// transaction. Commit will return an errors where errors.Is(ErrTxClosed) is true if the Tx is already closed, but is
	// otherwise safe to call multiple times. If the commit fails with a rollback status (e.g., the transaction was already
	// in a broken state), then an errors where errors.Is(ErrTxCommitRollback) is true will be returned.
	Commit(ctx context.Context) error

	// Rollback rolls back the transaction if this is a real transaction or rolls back to the savepoint if this is a
	// pseudo nested transaction. Rollback will return an errors where errors.Is(ErrTxClosed) is true if the Tx is already
	// closed, but is otherwise safe to call multiple times. Hence, a defer tx.Rollback() is safe even if tx.Commit() will
	// be called first in a non-errors condition. Any other failure of a real transaction will result in the connection
	// being closed.
	Rollback(ctx context.Context) error

	CopyFrom(ctx context.Context, tableName pgx.Identifier, columnNames []string, rowSrc pgx.CopyFromSource) (int64, error)
	SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults
	LargeObjects() pgx.LargeObjects

	Prepare(ctx context.Context, name, sql string) (*pgconn.StatementDescription, error)

	Exec(ctx context.Context, sql string, arguments ...any) (commandTag pgconn.CommandTag, err error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row

	// Conn returns the underlying *Conn that on which this transaction is executing.
	Conn() *pgx.Conn
}

// DB holds the driver connection pool
type DB struct {
	Pool PostgresPool
}

// dbConn represents a connection to a driver.
// It contains an SQL pool from the pgxpool package.
var dbConn = &DB{}

// maxOpenDbConn defines the maximum number of open driver connections.
// It is used to limit the number of concurrent connections to the driver.
const maxOpenDbConn = 10

// maxDbLifetime is the maximum lifetime of a driver connection in the pool.
// When a connection reaches its maximum lifetime, it will be closed and a new connection will be created.
// The value is set to 5 minutes (five * time.Minute).
const maxDbLifetime = 5 * time.Minute

// ConnectSQL connects to the Postgres server and returns a DB instance and an error. It requires a server.Server struct as a parameter.
// The function constructs the connection string using the server.Server fields and the pgxpool.ParseConfig function.
// It then sets the max connection count and connection lifetime on the config.
// Next, it creates a connection pool using pgxpool.NewWithConfig function.
// The function assigns the created pool to the SQL field of the dbConn variable.
// It also calls the testDB function to check if the connection to the driver is successful.
// If any errors occur during the process, it returns nil and the errors. Otherwise, it returns the dbConn variable and nil.
func ConnectSQL(dsn string) (*DB, error) {

	// parse the config
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, err
	}

	config.MaxConns = int32(maxOpenDbConn)
	config.MaxConnLifetime = maxDbLifetime

	// create the pool
	pool, err := pgxpool.NewWithConfig(context.Background(), config) // 使用ConnectConfig
	if err != nil {
		return nil, err
	}

	dbConn.Pool = pool
	// Set transaction options to serializable

	if err = testDB(pool); err != nil {
		return nil, err
	}

	return dbConn, nil
}

// testDB acquires and releases a connection from the pool
func testDB(p *pgxpool.Pool) error {
	conn, err := p.Acquire(context.Background())
	if err != nil {
		return err
	}
	defer conn.Release()
	return nil
}
