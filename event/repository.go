package event

import (
	"context"
	"github.com/jackc/pgx/v5/pgtype"
	"gofalre.io/shop/driver"
	"gofalre.io/shop/models"
	"gofalre.io/shop/sqlc"
	"time"

	"github.com/stripe/stripe-go/v79"
	"go.uber.org/zap"
)

var _ Repository = (*repository)(nil)

type Repository interface {
	Create(ctx context.Context, customer *models.Event) error
	GetByID(ctx context.Context, id string) (*models.Event, error)
	MarkAsProcessed(ctx context.Context, id string) error
}

type repository struct {
	conn   driver.PostgresPool
	logger *zap.Logger
}

func NewRepository(conn driver.PostgresPool, logger *zap.Logger) (Repository, error) {
	return &repository{
		conn:   conn,
		logger: logger,
	}, nil
}

func (r *repository) Create(ctx context.Context, event *models.Event) error {
	return sqlc.New(r.conn).CreateEvent(ctx, sqlc.CreateEventParams{
		ID:        event.ID,
		Type:      sqlc.EventType(event.Type),
		Processed: event.Processed,
		CreatedAt: pgtype.Timestamptz{Time: event.CreatedAt, Valid: true},
		UpdatedAt: pgtype.Timestamptz{Time: event.UpdatedAt, Valid: true},
	})
}

func (r *repository) GetByID(ctx context.Context, id string) (*models.Event, error) {
	sqlcEvent, err := sqlc.New(r.conn).GetEventByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return &models.Event{
		ID:        sqlcEvent.ID,
		Type:      stripe.EventType(sqlcEvent.Type),
		Processed: sqlcEvent.Processed,
	}, nil
}

func (r *repository) MarkAsProcessed(ctx context.Context, id string) error {
	return sqlc.New(r.conn).MarkEventAsProcessed(ctx, sqlc.MarkEventAsProcessedParams{
		ID:        id,
		UpdatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
	})
}
