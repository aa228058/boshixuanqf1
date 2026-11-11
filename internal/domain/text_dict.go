package domain

import (
	"time"

	"github.com/google/uuid"
)

// TextDict is a tenant-owned named collection of reusable text snippets.
// Items are managed as content assets; this type does not imply automatic message variation.
type TextDict struct {
	DictID    uuid.UUID `json:"dict_id"`
	TenantID  uuid.UUID `json:"tenant_id"`
	Name      string    `json:"name"`
	Items     []string  `json:"items"`
	CreatedAt time.Time `json:"created_at"`
}
