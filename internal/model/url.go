package model

import "time"

// URL представляет собой запись о сокращённой ссылке.
type URL struct {
	ShortCode   string     `json:"short_code"`
	OriginalURL string     `json:"original_url"`
	CreatedAt   time.Time  `json:"created_at"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	Clicks      int64      `json:"clicks"`
}

// IsExpired сообщает, истёк ли срок жизни ссылки.
func (u *URL) IsExpired() bool {
	if u.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*u.ExpiresAt)
}
