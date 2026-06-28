package service

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/emityoffwhite/url-shortener/internal/model"
	"github.com/emityoffwhite/url-shortener/internal/storage"
)

const (
	shortCodeLength = 7
	charset         = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	maxGenAttempts  = 5
)

// ErrInvalidURL возвращается, если переданный URL не прошёл валидацию.
var ErrInvalidURL = errors.New("invalid url: must be absolute and use http or https scheme")

// URLService содержит бизнес-логику сокращения ссылок.
type URLService struct {
	store storage.Storage
}

// NewURLService создаёт сервис с переданной реализацией хранилища.
// Принимает интерфейс, а не конкретный тип - это и есть dependency injection,
// благодаря которому в тестах легко подставить мок-хранилище.
func NewURLService(store storage.Storage) *URLService {
	return &URLService{store: store}
}

// Shorten создаёт короткую ссылку для original. Если ttl > 0, ссылка истечёт через ttl.
func (s *URLService) Shorten(ctx context.Context, original string, ttl time.Duration) (*model.URL, error) {
	if err := validateURL(original); err != nil {
		return nil, err
	}

	var shortCode string
	var err error

	// Коллизии при генерации случайного кода крайне маловероятны
	// (62^7 ≈ 3.5 * 10^12 комбинаций), но на проде лучше явно их обработать,
	// чем понадеяться на удачу.
	for attempt := 0; attempt < maxGenAttempts; attempt++ {
		shortCode, err = generateShortCode(shortCodeLength)
		if err != nil {
			return nil, fmt.Errorf("generate short code: %w", err)
		}

		u := &model.URL{
			ShortCode:   shortCode,
			OriginalURL: original,
			CreatedAt:   time.Now(),
		}
		if ttl > 0 {
			expires := time.Now().Add(ttl)
			u.ExpiresAt = &expires
		}

		err = s.store.Save(ctx, u)
		if err == nil {
			return u, nil
		}
		if !errors.Is(err, storage.ErrAlreadyExists) {
			return nil, fmt.Errorf("save url: %w", err)
		}
		// При коллизии цикл просто пробует сгенерировать новый код.
	}

	return nil, fmt.Errorf("failed to generate unique short code after %d attempts", maxGenAttempts)
}

// Resolve возвращает оригинальный URL по короткому коду и увеличивает счётчик переходов.
func (s *URLService) Resolve(ctx context.Context, shortCode string) (*model.URL, error) {
	u, err := s.store.Get(ctx, shortCode)
	if err != nil {
		return nil, err
	}

	if u.IsExpired() {
		return nil, storage.ErrNotFound
	}

	// Увеличение счётчика не должно блокировать редирект пользователя,
	// поэтому делаем это в отдельной горутине.
	go func() {
		// Используем новый контекст без отмены: исходный ctx умрёт вместе с HTTP-запросом,
		// а инкремент клика должен завершиться независимо от него.
		bgCtx := context.Background()
		_ = s.store.IncrementClicks(bgCtx, shortCode)
	}()

	return u, nil
}

// Delete удаляет короткую ссылку.
func (s *URLService) Delete(ctx context.Context, shortCode string) error {
	return s.store.Delete(ctx, shortCode)
}

func validateURL(raw string) error {
	parsed, err := url.ParseRequestURI(raw)
	if err != nil {
		return ErrInvalidURL
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return ErrInvalidURL
	}
	if parsed.Host == "" {
		return ErrInvalidURL
	}
	return nil
}

func generateShortCode(length int) (string, error) {
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	for i, v := range b {
		b[i] = charset[int(v)%len(charset)]
	}
	return string(b), nil
}
