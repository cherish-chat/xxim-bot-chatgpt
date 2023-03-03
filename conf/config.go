package conf

import (
	"github.com/cherish-chat/xxim-server/common/utils"
	"github.com/cherish-chat/xxim-server/sdk/config"
	"github.com/cherish-chat/xxim-server/sdk/conn"
	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/zrpc"
	"log"
	"os"
	"runtime"
)

type Config struct {
	UserId       string
	Password     string
	Platform     string
	WsAddr       string
	RsaPublicKey string

	CommandPrefix string
	ChatGptRpc    zrpc.RpcClientConf

	Sqlite3Path string

	MaxTokens  int32
	MaxChatLog int32
}

func (c Config) GetSdkConfig(deviceId string) config.Config {
	return config.Config{Client: conn.Config{
		Addr:    c.WsAddr,
		Headers: nil,
		DeviceConfig: conn.DeviceConfig{
			PackageId:   deviceId,
			Platform:    c.Platform,
			DeviceModel: "bot",
			OsVersion:   runtime.GOOS,
		},
		UserConfig: conn.UserConfig{
			UserId:   c.UserId,
			Password: utils.Md5(c.Password),
		},
		RsaPublicKey: c.RsaPublicKey,
	}}
}

var BotConfig *Config

func MustLoadConfig(configpath string) Config {
	// 读取 configpath 指定的配置文件
	bytes, err := os.ReadFile(configpath)
	if err != nil {
		log.Fatalf("读取配置文件失败: %s", err)
	}
	var c = &Config{}
	err = conf.LoadFromJsonBytes(bytes, c)
	if err != nil {
		log.Fatalf("加载配置文件失败: %s", err)
	}
	BotConfig = c
	return *c
}
