package localdb

import (
	chatgptpb "github.com/cherish-chat/chatgpt-server-python/pb"
	"github.com/cherish-chat/xxim-bot-chatgpt/localdb/model"
	"github.com/cherish-chat/xxim-server/common/utils"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"log"
)

type Sqlite3 struct {
	tx *gorm.DB
}

var singletonSqlite3 *Sqlite3

func InitSqlite3(dbpath string) *Sqlite3 {
	if singletonSqlite3 == nil {
		tx, e := gorm.Open(sqlite.Open(dbpath), &gorm.Config{})
		if e != nil {
			log.Fatalf("打开数据库失败: %s", e)
		}
		tx.AutoMigrate(&model.Kv{})
		tx.AutoMigrate(&model.ChatLog{})
		tx.AutoMigrate(&model.ChatScene{})
		singletonSqlite3 = &Sqlite3{tx: tx}
	}
	return singletonSqlite3
}

func GetDeviceId() string {
	// 查询 kv 表中的 device_id
	var kv model.Kv
	err := singletonSqlite3.tx.First(&kv, "k = ?", "device_id").Error
	if err != nil {
		// 如果没有 device_id 则生成一个
		deviceId := utils.GenId()
		singletonSqlite3.tx.Create(&model.Kv{K: "device_id", V: deviceId})
	}
	return kv.V
}

func GetChatScene(userId string) *chatgptpb.ChatGptMessage {
	var scene model.ChatScene
	err := singletonSqlite3.tx.First(&scene, "user_id = ?", userId).Error
	if err != nil {
		// default chat scene
		return nil
	}
	return scene.Message.ChatGptMessage
}

func SetChatScene(userId string, message *chatgptpb.ChatGptMessage) {
	var scene model.ChatScene
	err := singletonSqlite3.tx.First(&scene, "user_id = ?", userId).Error
	if err != nil {
		singletonSqlite3.tx.Create(&model.ChatScene{
			UserId:  userId,
			Message: model.ChatGptMessage{ChatGptMessage: message},
		})
	} else {
		singletonSqlite3.tx.Model(&scene).Update("message", model.ChatGptMessage{ChatGptMessage: message})
	}
}

func GetChatLog(userId string, limit int32) []*chatgptpb.ChatGptMessage {
	var logs []*model.ChatLog
	err := singletonSqlite3.tx.Where("user_id = ?", userId).Order("id desc").Limit(int(limit)).Find(&logs).Error
	if err != nil {
		return nil
	}
	var result []*chatgptpb.ChatGptMessage
	for _, l := range logs {
		result = append(result, l.Message.ChatGptMessage)
	}
	return result
}

func InsertChatLogs(userId string, chatLogs []*chatgptpb.ChatGptMessage) {
	var models []*model.ChatLog
	for _, l := range chatLogs {
		models = append(models, &model.ChatLog{
			UserId:  userId,
			Message: model.ChatGptMessage{ChatGptMessage: l},
		})
	}
	if len(models) > 0 {
		singletonSqlite3.tx.Create(&models)
	}
}
