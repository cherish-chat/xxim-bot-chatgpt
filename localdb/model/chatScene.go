package model

import (
	"database/sql/driver"
	"encoding/json"
	chatgptpb "github.com/cherish-chat/chatgpt-server-python/pb"
)

type ChatScene struct {
	UserId  string         `gorm:"primary_key;type:char(64);not null"`
	Message ChatGptMessage `gorm:"type:text;not null"`
}

func (m *ChatScene) TableName() string {
	return "chat_scene"
}

type ChatGptMessage struct {
	*chatgptpb.ChatGptMessage
}

// Scan 实现 Scan 方法
func (m *ChatGptMessage) Scan(src interface{}) error {
	return json.Unmarshal(src.([]byte), m)
}

// Value 实现 Value 方法
func (m ChatGptMessage) Value() (driver.Value, error) {
	return json.Marshal(m)
}
