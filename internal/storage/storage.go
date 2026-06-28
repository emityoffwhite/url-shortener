package storage

import (
	"context"
	"errors"

	"github.com/emityoffwhite/url-shortener/internal/model"
)

// ErrNotFound возвращается, когда короткая ссылка не найдена в хранилище.
var ErrNotFound = errors.New("short url not found")

// ErrAlreadyExists возвращается при попытке сохранить уже существующий код.
var ErrAlreadyExists = errors.New("short code already exists")

// Storage описывает контракт хранилища ссылок.
// Любая реализация (in-memory, Postgres, Redis) должна удовлетворять этому интерфейсу,
// что позволяет подменять backend без изменения бизнес-логики в service.
type Storage interface {
	Save(ctx context.Context, u *model.URL) error
	Get(ctx context.Context, shortCode string) (*model.URL, error)
	IncrementClicks(ctx context.Context, shortCode string) error
	Delete(ctx context.Context, shortCode string) error
}
