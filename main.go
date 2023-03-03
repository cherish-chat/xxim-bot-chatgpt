package main

import (
	"context"
	"flag"
	"fmt"
	chatgptpb "github.com/cherish-chat/chatgpt-server-python/pb"
	"github.com/cherish-chat/xxim-bot-chatgpt/conf"
	"github.com/cherish-chat/xxim-bot-chatgpt/localdb"
	"github.com/cherish-chat/xxim-server/common/pb"
	"github.com/cherish-chat/xxim-server/common/utils"
	"github.com/cherish-chat/xxim-server/sdk/config"
	"github.com/cherish-chat/xxim-server/sdk/svc"
	"log"
	"strings"
	"time"
)

func xxim(cfg config.Config) *eventHandler {
	svcCtx := svc.NewServiceContext(cfg)
	eh := newEventHandler(svcCtx)

	svcCtx.SetEventHandler(eh)

	err := svcCtx.Client().Connect()
	if err != nil {
		log.Fatalf("connect error: %v", err)
	}
	return eh
}

var configPath = flag.String("c", "config.json", "config file path")

func main() {
	flag.Parse()
	cfg := conf.MustLoadConfig(*configPath)
	localdb.InitSqlite3(cfg.Sqlite3Path)
	eh := xxim(cfg.GetSdkConfig(localdb.GetDeviceId()))
	chatgptpb.InitClient(cfg.ChatGptRpc)
	for {
		select {
		case msgData := <-eh.msgChan:
			go func() {
				handleMsg(eh, msgData)
			}()
		}
	}
}

func handleMsg(eh *eventHandler, data *pb.MsgData) {
	if data == nil {
		return
	}
	if data.SenderId == eh.svcCtx.Config.Client.UserConfig.UserId {
		return
	}
	// 如果没有艾特自己
	if !utils.InSlice(data.AtUsers, eh.svcCtx.Config.Client.UserConfig.UserId) {
		return
	}
	// 如果不是文本消息
	if pb.ContentType(data.ContentType) != pb.ContentType_TEXT {
		return
	}
	// 是否是空字符串
	text := string(data.Content)
	if text == "" {
		return
	}
	// 开始处理
	// 是否是命令消息
	prefix := fmt.Sprintf("@%s %s", eh.svcCtx.Config.Client.UserConfig.UserId, conf.BotConfig.CommandPrefix)
	if strings.HasPrefix(text,
		prefix,
	) {
		handleCmd(eh, data, strings.TrimPrefix(text, prefix))
		return
	}
	// 当前用户和机器人的聊天场景
	var chatScene *chatgptpb.ChatGptMessage
	chatScene = localdb.GetChatScene(data.SenderId)
	// 用户和机器人的聊天记录
	var chatLog []*chatgptpb.ChatGptMessage
	chatLog = localdb.GetChatLog(data.SenderId, conf.BotConfig.MaxChatLog)
	if chatScene != nil {
		// chatScene 插入到最前面
		chatLog = append([]*chatgptpb.ChatGptMessage{chatScene}, chatLog...)
	}
	// 用户的输入插入到最后面
	chatLog = append(chatLog, &chatgptpb.ChatGptMessage{
		Text: text,
		Role: chatgptpb.RoleEnum_User,
	})
	// 调用chatgpt
	answerResp, err := chatgptpb.Answer(context.Background(), &chatgptpb.AnswerReq{
		Messages:  chatLog,
		MaxTokens: conf.BotConfig.MaxTokens,
	})
	reply := ""
	if err != nil {
		reply = "回复失败：" + err.Error()
	} else {
		if len(answerResp.Choices) == 0 {
			reply = "回复失败：没有回复"
		} else {
			reply = answerResp.Choices[0].Message.Text
			// 插入到数据库
			localdb.InsertChatLogs(data.SenderId, []*chatgptpb.ChatGptMessage{
				{
					Text: text,
					Role: chatgptpb.RoleEnum_User,
				},
				{
					Text: reply,
					Role: chatgptpb.RoleEnum_Assistant,
				},
			})
		}
	}
	replyMsg(eh, data, reply)
}

func replyMsg(eh *eventHandler, origin *pb.MsgData, reply string) error {
	err := eh.svcCtx.Client().RequestX(
		"/v1/msg/sendMsgList",
		&pb.SendMsgListReq{
			MsgDataList:  []*pb.MsgData{getReplyMsgData(origin, reply)},
			DeliverAfter: nil,
			CommonReq:    nil,
		},
		&pb.SendMsgListResp{},
	)
	if err != nil {
		log.Printf("send reply msg error: %v", err)
	}
	return err
}

func getReplyMsgData(origin *pb.MsgData, reply string) *pb.MsgData {
	return &pb.MsgData{
		ClientMsgId: utils.GenId(),
		ClientTime:  utils.AnyToString(time.Now().UnixMilli()),
		SenderId:    conf.BotConfig.UserId,
		SenderInfo:  nil,
		ConvId:      origin.ConvId,
		AtUsers: []string{
			origin.SenderId,
		},
		ContentType: int32(pb.ContentType_TEXT),
		Content:     []byte("@" + origin.SenderId + " " + reply),
		Options: &pb.MsgData_Options{
			StorageForServer:  true,
			StorageForClient:  true,
			OfflinePush:       false,
			UpdateConvMsg:     true,
			UpdateUnreadCount: true,
		},
		Ext: utils.AnyToBytes(map[string]any{
			"replyMsgModel": utils.AnyToString(map[string]any{
				"clientMsgId": origin.ClientMsgId,
				"serverMsgId": origin.ServerMsgId,
				"clientTime":  utils.AnyToInt64(origin.ClientTime),
				"serverTime":  utils.AnyToInt64(origin.ServerTime),
				"senderId":    origin.SenderId,
				"senderInfo":  string(origin.SenderInfo),
				"seq":         utils.AnyToInt64(origin.Seq),
				"convId":      origin.ConvId,
				"atUsers":     origin.AtUsers,
				"contentType": origin.ContentType,
				"content":     string(origin.Content),
				"options":     utils.AnyToString(origin.Options),
				"offlinePush": utils.AnyToString(origin.OfflinePush),
				"ext":         string(origin.Ext),
			}),
		}),
	}
}

func handleCmd(eh *eventHandler, data *pb.MsgData, cmd string) {
	// split cmd
	cmds := strings.Split(cmd, " ")
	// 取第一个
	if len(cmds) == 0 {
		return
	}
	reply := ""
	h, ok := cmdMap[cmds[0]]
	if !ok {
		log.Printf("unknown cmd: %s", cmds[0])
		reply = defaultCmdTip
	} else {
		reply = h.do(data, cmds[1:]...)
	}
	replyMsg(eh, data, reply)
}
