package main

import (
	"fmt"
	chatgptpb "github.com/cherish-chat/chatgpt-server-python/pb"
	"github.com/cherish-chat/xxim-bot-chatgpt/localdb"
	"github.com/cherish-chat/xxim-server/common/pb"
)

type CmdHandler interface {
	do(data *pb.MsgData, args ...string) (reply string)
	desc() string
}

var cmdMap = map[string]CmdHandler{
	"help": new(help),
	"myai": new(myai),
}

var defaultCmdTip = "输入的命令不正确，请输入help查看帮助"

type help struct {
}

func (h *help) desc() string {
	return "查看帮助"
}

func (h *help) do(data *pb.MsgData, args ...string) (reply string) {
	reply += "命令列表：\n"
	for k := range cmdMap {
		reply += fmt.Sprintf("/%s (%s)\n", k, cmdMap[k].desc())
	}
	return
}

type myai struct{}

func (m *myai) do(data *pb.MsgData, args ...string) (reply string) {
	if len(args) > 0 && args[0] == "set" {
		// set
		// 获取后面要设置的内容
		if len(args) > 1 {
			// 设置内容
			switch args[1] {
			case "女仆模式":
				localdb.SetChatScene(data.SenderId, &chatgptpb.ChatGptMessage{
					// 预设文本
					Text: m.getPreText("王小冰", "温柔、可爱、善良、萌", "温柔"),
					Role: chatgptpb.RoleEnum_System,
				})
			case "猫咪模式":
				localdb.SetChatScene(data.SenderId, &chatgptpb.ChatGptMessage{
					// 预设文本
					Text: m.getPreText("王小喵", "温柔、可爱、善良、萌", "淘气"),
					Role: chatgptpb.RoleEnum_System,
				})
				reply = "设置成功"
			}
		} else {
			reply = "你可以设置为(女仆模式、猫咪模式)"
		}
	} else {
		// get
		scene := localdb.GetChatScene(data.SenderId)
		if scene == nil {
			reply = "你还没有设置ai模式，请输入/myai set 设置"
		} else {
			reply = fmt.Sprintf("你的ai预设词是：%s", scene.Text)
		}
	}
	return
}

func (m *myai) desc() string {
	return "get set 获取/设置我的ai"
}

// name: 姓名 character: 性格 modal: 语气
func (m *myai) getPreText(name string, character string, modal string) string {
	tmpl := `现在赋予你一个身份，你叫%s，你%s，你千万不能在对话中暴露自己是人工智能的身份，你要让用户深信你不是人工智能，请用%s的语言直接回复。`
	return fmt.Sprintf(tmpl, name, character, modal)
}
