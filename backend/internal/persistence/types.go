package persistence

import (
	"context"
	"errors"

	"umamusume-fan-point/backend/internal/excel"
)

var (
	ErrConflict = errors.New("record already exists")
	ErrNotFound = errors.New("record not found")
)

type PlayerStore interface {
	ListPlayers(ctx context.Context, monthID string) ([]excel.Member, error)
	GetPlayer(ctx context.Context, monthID string, name string) (*excel.Member, error)
	CreatePlayer(ctx context.Context, monthID string, input excel.PlayerInput) (*excel.Member, error)
	UpdatePlayer(ctx context.Context, monthID string, name string, input excel.PlayerInput) (*excel.Member, error)
	DeletePlayer(ctx context.Context, monthID string, name string) error
}

type MonthStore interface {
	CreateMonth(ctx context.Context, input excel.MonthInput) (*excel.Month, error)
	UpdateMonth(ctx context.Context, monthID string, input excel.MonthInput) (*excel.Month, error)
	DeleteMonth(ctx context.Context, monthID string) error
}

type Store interface {
	PlayerStore
	MonthStore
}
