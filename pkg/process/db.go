package process

import (
	"fmt"
	"github.com/eryajf/chatgpt-dingtalk/pkg/db"
	"github.com/eryajf/chatgpt-dingtalk/pkg/dingbot"
	"github.com/eryajf/chatgpt-dingtalk/pkg/logger"
	"github.com/eryajf/chatgpt-dingtalk/public"
	"strings"
)

// 与数据库交互的请求处理在此

// SelectHistory 查询会话历史
func SelectHistory(rmsg *dingbot.ReceiveMsg) error {
	name := strings.TrimSpace(strings.Split(rmsg.Text.Content, ":")[1])
	// 获取数据列表
	var chat db.Chat
	if !chat.Exist(map[string]interface{}{"username": name}) {
		_, err := rmsg.ReplyToDingtalk(string(dingbot.TEXT), "用户名错误，这个用户不存在，请核实之后再进行查询")
		if err != nil {
			logger.Error(fmt.Errorf("send message error: %v", err))
			return err
		}
		return fmt.Errorf("用户名错误，这个用户不存在，请核实之后重新查询")
	}

	// 回复@我的用户
	reply := fmt.Sprintf("- 在线查看: [点我](%s)\n- 下载文件: [点我](%s)\n- 在线预览请安装插件:[Markdown Preview Plus](https://chrome.google.com/webstore/detail/markdown-preview-plus/febilkbfcbhebfnokafefeacimjdckgl)", public.Config.ServiceURL+":"+public.Config.Port+"/history/"+name+".md", public.Config.ServiceURL+":"+public.Config.Port+"/download/"+name+".md")

	_, err := rmsg.ReplyToDingtalk(string(dingbot.MARKDOWN), reply)
	if err != nil {
		logger.Error(fmt.Errorf("send message error: %v", err))
		return err
	}
	return nil
}

func OutPutHistory(name string) string {
	names := strings.Split(name, ".")
	if len(names) == 2 {
		name = names[0]
	}
	var chat db.Chat
	chats, err := chat.List(db.ChatListReq{
		Username: name,
	})
	if err != nil {
		return ""
	}
	var rst string
	for _, chatTmp := range chats {
		ctime := chatTmp.CreatedAt.Format("2006-01-02 15:04:05")
		if chatTmp.ChatType == 1 {
			rst += fmt.Sprintf("## 🙋 %s 问\n\n**时间:** %v\n\n**问题为:** %s\n\n", chatTmp.Username, ctime, chatTmp.Content)
		} else {
			rst += fmt.Sprintf("## 🤖 机器人 答\n\n**时间:** %v\n\n**回答如下：** \n\n%s\n\n", ctime, chatTmp.Content)
		}
		// TODO: 答案应该严格放在问题之后，目前只根据ID排序进行的陈列，当一个用户同时提出多个问题时，最终展示的可能会有点问题
	}
	return rst
}

func ClearHistory(rmsg *dingbot.ReceiveMsg) error {
	name := strings.TrimSpace(strings.Split(rmsg.Text.Content, ":")[1])
	// 获取数据列表
	var chat db.Chat
	if !chat.Exist(map[string]interface{}{"username": name}) {
		_, err := rmsg.ReplyToDingtalk(string(dingbot.TEXT), "用户名错误，这个用户不存在，请核实之后再进行查询")
		if err != nil {
			logger.Error(fmt.Errorf("send message error: %v", err))
			return err
		}
		return fmt.Errorf("用户名错误，这个用户不存在，请核实之后重新查询")
	}

	err := chat.Delete(name)
	_, err = rmsg.ReplyToDingtalk(string(dingbot.MARKDOWN), fmt.Sprintf("用户：%s对话清理成功🔜🗑️！", name))
	if err != nil {
		logger.Error(fmt.Errorf("send message error: %v", err))
		return err
	}
	return nil
}
