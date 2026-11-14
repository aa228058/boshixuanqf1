package domain

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

// LegacyMessageVariables 是已退役的内置占位符：群发执行器不再替换它们，
// 留着只会把 "{{first_name}}" 原样发给群成员。自定义词典名仍受支持。
var LegacyMessageVariables = map[string]bool{
	"first_name": true, "group_name": true, "user_id": true,
	"now": true, "username": true, "phone": true,
}

// IsLegacyMessageVariable 判断是否是已退役的内置占位符。
func IsLegacyMessageVariable(name string) bool {
	return LegacyMessageVariables[strings.ToLower(strings.TrimSpace(name))]
}

// MessageTemplate represents a reusable message template.
type MessageTemplate struct {
	TemplateID uuid.UUID `json:"template_id"`
	TenantID   uuid.UUID `json:"tenant_id"`
	Name       string    `json:"name"`
	Content    string    `json:"content"`
	Variables  []string  `json:"variables"`
	CreatedAt  time.Time `json:"created_at"`
}
