package model

type ChatLog struct {
	Id      int            `gorm:"primary_key;AUTO_INCREMENT;not null"`
	UserId  string         `gorm:"type:char(64);not null;index;"`
	Message ChatGptMessage `gorm:"type:text;not null"`
}

func (m *ChatLog) TableName() string {
	return "chat_log"
}
