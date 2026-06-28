package storage

import (
	"context"
	"sync"
	"testing"

	"github.com/emityoffwhite/url-shortener/internal/model"
)

func TestMemoryStorage_SaveAndGet(t *testing.T) {
	s := NewMemoryStorage()
	ctx := context.Background()

	u := &model.URL{ShortCode: "abc1234", OriginalURL: "https://example.com"}
	if err := s.Save(ctx, u); err != nil {
		t.Fatalf("Save() unexpected error: %v", err)
	}

	got, err := s.Get(ctx, "abc1234")
	if err != nil {
		t.Fatalf("Get() unexpected error: %v", err)
	}
	if got.OriginalURL != u.OriginalURL {
		t.Errorf("Get() OriginalURL = %q, want %q", got.OriginalURL, u.OriginalURL)
	}
}

func TestMemoryStorage_SaveDuplicate(t *testing.T) {
	s := NewMemoryStorage()
	ctx := context.Background()

	u := &model.URL{ShortCode: "dup0001", OriginalURL: "https://example.com"}
	if err := s.Save(ctx, u); err != nil {
		t.Fatalf("first Save() unexpected error: %v", err)
	}

	err := s.Save(ctx, u)
	if err != ErrAlreadyExists {
		t.Errorf("second Save() error = %v, want ErrAlreadyExists", err)
	}
}

func TestMemoryStorage_GetNotFound(t *testing.T) {
	s := NewMemoryStorage()
	_, err := s.Get(context.Background(), "missing")
	if err != ErrNotFound {
		t.Errorf("Get() error = %v, want ErrNotFound", err)
	}
}

func TestMemoryStorage_IncrementClicks(t *testing.T) {
	s := NewMemoryStorage()
	ctx := context.Background()

	u := &model.URL{ShortCode: "click01", OriginalURL: "https://example.com"}
	_ = s.Save(ctx, u)

	for i := 0; i < 5; i++ {
		if err := s.IncrementClicks(ctx, "click01"); err != nil {
			t.Fatalf("IncrementClicks() unexpected error: %v", err)
		}
	}

	got, _ := s.Get(ctx, "click01")
	if got.Clicks != 5 {
		t.Errorf("Clicks = %d, want 5", got.Clicks)
	}
}

func TestMemoryStorage_Delete(t *testing.T) {
	s := NewMemoryStorage()
	ctx := context.Background()

	u := &model.URL{ShortCode: "del0001", OriginalURL: "https://example.com"}
	_ = s.Save(ctx, u)

	if err := s.Delete(ctx, "del0001"); err != nil {
		t.Fatalf("Delete() unexpected error: %v", err)
	}

	_, err := s.Get(ctx, "del0001")
	if err != ErrNotFound {
		t.Errorf("Get() after Delete() error = %v, want ErrNotFound", err)
	}
}

// TestMemoryStorage_ConcurrentAccess проверяет отсутствие гонок данных
// при параллельных операциях. Запускается с флагом -race в CI.
func TestMemoryStorage_ConcurrentAccess(t *testing.T) {
	s := NewMemoryStorage()
	ctx := context.Background()
	u := &model.URL{ShortCode: "race0001", OriginalURL: "https://example.com"}
	_ = s.Save(ctx, u)

	var wg sync.WaitGroup
	const goroutines = 100

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = s.IncrementClicks(ctx, "race0001")
			_, _ = s.Get(ctx, "race0001")
		}()
	}
	wg.Wait()

	got, _ := s.Get(ctx, "race0001")
	if got.Clicks != int64(goroutines) {
		t.Errorf("Clicks = %d, want %d", got.Clicks, goroutines)
	}
}
