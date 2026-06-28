package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/emityoffwhite/url-shortener/internal/storage"
)

func TestURLService_Shorten(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr error
	}{
		{name: "valid https url", input: "https://example.com/path", wantErr: nil},
		{name: "valid http url", input: "http://example.com", wantErr: nil},
		{name: "missing scheme", input: "example.com", wantErr: ErrInvalidURL},
		{name: "invalid scheme", input: "ftp://example.com", wantErr: ErrInvalidURL},
		{name: "empty string", input: "", wantErr: ErrInvalidURL},
		{name: "garbage input", input: "not a url at all", wantErr: ErrInvalidURL},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewURLService(storage.NewMemoryStorage())
			u, err := svc.Shorten(context.Background(), tt.input, 0)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("Shorten() error = %v, want %v", err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Fatalf("Shorten() unexpected error: %v", err)
			}
			if len(u.ShortCode) != shortCodeLength {
				t.Errorf("ShortCode length = %d, want %d", len(u.ShortCode), shortCodeLength)
			}
			if u.OriginalURL != tt.input {
				t.Errorf("OriginalURL = %q, want %q", u.OriginalURL, tt.input)
			}
		})
	}
}

func TestURLService_ShortenGeneratesUniqueCodesForSameURL(t *testing.T) {
	svc := NewURLService(storage.NewMemoryStorage())
	ctx := context.Background()

	first, err := svc.Shorten(ctx, "https://example.com", 0)
	if err != nil {
		t.Fatalf("first Shorten() unexpected error: %v", err)
	}

	second, err := svc.Shorten(ctx, "https://example.com", 0)
	if err != nil {
		t.Fatalf("second Shorten() unexpected error: %v", err)
	}

	if first.ShortCode == second.ShortCode {
		t.Errorf("expected different short codes for two calls, got same: %q", first.ShortCode)
	}
}

func TestURLService_Resolve(t *testing.T) {
	svc := NewURLService(storage.NewMemoryStorage())
	ctx := context.Background()

	created, err := svc.Shorten(ctx, "https://example.com", 0)
	if err != nil {
		t.Fatalf("Shorten() unexpected error: %v", err)
	}

	resolved, err := svc.Resolve(ctx, created.ShortCode)
	if err != nil {
		t.Fatalf("Resolve() unexpected error: %v", err)
	}
	if resolved.OriginalURL != "https://example.com" {
		t.Errorf("OriginalURL = %q, want %q", resolved.OriginalURL, "https://example.com")
	}
}

func TestURLService_ResolveNotFound(t *testing.T) {
	svc := NewURLService(storage.NewMemoryStorage())
	_, err := svc.Resolve(context.Background(), "doesnotexist")
	if !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("Resolve() error = %v, want ErrNotFound", err)
	}
}

func TestURLService_ResolveExpired(t *testing.T) {
	svc := NewURLService(storage.NewMemoryStorage())
	ctx := context.Background()

	// TTL в 1 наносекунду гарантированно истечёт к моменту вызова Resolve.
	created, err := svc.Shorten(ctx, "https://example.com", time.Nanosecond)
	if err != nil {
		t.Fatalf("Shorten() unexpected error: %v", err)
	}

	time.Sleep(time.Millisecond)

	_, err = svc.Resolve(ctx, created.ShortCode)
	if !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("Resolve() on expired url, error = %v, want ErrNotFound", err)
	}
}

func TestURLService_ResolveIncrementsClicks(t *testing.T) {
	svc := NewURLService(storage.NewMemoryStorage())
	ctx := context.Background()

	created, _ := svc.Shorten(ctx, "https://example.com", 0)

	_, err := svc.Resolve(ctx, created.ShortCode)
	if err != nil {
		t.Fatalf("Resolve() unexpected error: %v", err)
	}

	// IncrementClicks запускается в отдельной горутине внутри Resolve,
	// поэтому даём ей время выполниться перед проверкой.
	time.Sleep(50 * time.Millisecond)

	u, err := svc.Resolve(ctx, created.ShortCode)
	if err != nil {
		t.Fatalf("second Resolve() unexpected error: %v", err)
	}
	if u.Clicks < 1 {
		t.Errorf("Clicks = %d, want >= 1", u.Clicks)
	}
}

func TestURLService_Delete(t *testing.T) {
	svc := NewURLService(storage.NewMemoryStorage())
	ctx := context.Background()

	created, _ := svc.Shorten(ctx, "https://example.com", 0)

	if err := svc.Delete(ctx, created.ShortCode); err != nil {
		t.Fatalf("Delete() unexpected error: %v", err)
	}

	_, err := svc.Resolve(ctx, created.ShortCode)
	if !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("Resolve() after Delete(), error = %v, want ErrNotFound", err)
	}
}
