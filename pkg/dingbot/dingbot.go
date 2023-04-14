package dingbot

import (
	"fmt"
)

// 接收的消息体
type ReceiveMsg struct {
	ConversationID string `json:"conversationId"`
	AtUsers        []struct {
		DingtalkID string `json:"dingtalkId"`
	} `json:"atUsers"`
	ChatbotUserID             string  `json:"chatbotUserId"`
	MsgID                     string  `json:"msgId"`
	SenderNick                string  `json:"senderNick"`
	IsAdmin                   bool    `json:"isAdmin"`
	SenderStaffId             string  `json:"senderStaffId"`
	SessionWebhookExpiredTime int64   `json:"sessionWebhookExpiredTime"`
	CreateAt                  int64   `json:"createAt"`
	ConversationType          string  `json:"conversationType"`
	SenderID                  string  `json:"senderId"`
	ConversationTitle         string  `json:"conversationTitle"`
	IsInAtList                bool    `json:"isInAtList"`
	SessionWebhook            string  `json:"sessionWebhook"`
	Text                      Text    `json:"text"`
	RobotCode                 string  `json:"robotCode"`
	Msgtype                   MsgType `json:"msgtype"`
}

// 消息类型
type MsgType string

const TEXT MsgType = "text"
const MARKDOWN MsgType = "markdown"

// Text 消息
type TextMessage struct {
	MsgType MsgType `json:"msgtype"`
	At      *At     `json:"at"`
	Text    *Text   `json:"text"`
}

// Text 消息内容
type Text struct {
	Content string `json:"content"`
}

// MarkDown 消息
type MarkDownMessage struct {
	MsgType  MsgType   `json:"msgtype"`
	At       *At       `json:"at"`
	MarkDown *MarkDown `json:"markdown"`
}

// MarkDown 消息内容
type MarkDown struct {
	Title string `json:"title"`
	Text  string `json:"text"`
}

// at 内容
type At struct {
	AtUserIds []string `json:"atUserIds"`
	AtMobiles []string `json:"atMobiles"`
	IsAtAll   bool     `json:"isAtAll"`
}

// 获取用户标识，兼容当 SenderStaffId 字段为空的场景，此处提供给发送消息是艾特使用
func (r ReceiveMsg) GetSenderIdentifier() (uid string) {
	uid = r.SenderStaffId
	if uid == "" {
		uid = r.SenderNick
	}
	return
}

// GetChatTitle 获取聊天的群名字，如果是私聊，则命名为 昵称_私聊
func (r ReceiveMsg) GetChatTitle() (chatType string) {
	chatType = r.ConversationTitle
	if chatType == "" {
		chatType = r.SenderNick + "_私聊"
	}
	return
}

// 发消息给钉钉
func (r ReceiveMsg) ReplyToDingtalk(msgType, msg string) (statuscode int, err error) {
	atUser := r.SenderStaffId
	if atUser == "" {
		msg = fmt.Sprintf("%s\n\n@%s", msg, r.SenderNick)
	}

	fmt.Printf("\n响应内容:\n%v\n", msg)

	return
}
