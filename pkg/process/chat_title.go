package process

import (
	"fmt"
	"github.com/eryajf/chatgpt-dingtalk/pkg/dingbot"
	"github.com/eryajf/chatgpt-dingtalk/pkg/logger"
	"github.com/eryajf/chatgpt-dingtalk/public"
	"strings"
)

func SetTitle(rmsg *dingbot.ReceiveMsg) error {
	name := strings.TrimSpace(strings.Split(rmsg.Text.Content, ":")[1])

	if name == "" {
		name = "chat-gpt"
	}

	public.UserService.SetUserName(name)

	_, err := rmsg.ReplyToDingtalk(string(dingbot.TEXT), "用户名设置成功，对话昵称为："+name)
	if err != nil {
		logger.Error(fmt.Errorf("seting user name error: %v", err))
		return err
	}
	return nil
}
