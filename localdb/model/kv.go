package model

type Kv struct {
	K string `gorm:"primary_key"`
	V string `gorm:"type:text"`
}

func (m *Kv) TableName() string {
	return "kv"
}
