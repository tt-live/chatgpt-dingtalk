package main

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/eryajf/chatgpt-dingtalk/pkg/dingbot"
	"github.com/eryajf/chatgpt-dingtalk/pkg/logger"
	"github.com/eryajf/chatgpt-dingtalk/pkg/process"
	"github.com/eryajf/chatgpt-dingtalk/public"
	"github.com/xgfone/ship/v5"
)

func init() {
	public.InitSvc()
	logger.InitLogger(public.Config.LogLevel)
}
func main() {
	StartCmd()
}

func StartCmd() {

	go router()

	port := ":" + public.Config.Port
	logger.Info("🚀 The HTTP Server is running on", port)

	// Create a channel to receive the os.Signal exit signal
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	scanner := bufio.NewScanner(os.Stdin)
	for {
		select {
		case <-sigs:
			logger.Info("Shutting down server input...")
			return
		default:
			fmt.Print("\n请输入：")
			scanner.Scan()
			input := scanner.Text()
			msg := dingbot.ReceiveMsg{
				ConversationID: public.UserService.GetUserName(),
				ChatbotUserID:  public.UserService.GetUserName(),
				SenderNick:     public.UserService.GetUserName(),
				SenderStaffId:  public.UserService.GetUserName(),
				Text:           dingbot.Text{Content: input},
			}
			if strings.TrimSpace(input) == "" {
				_, err := msg.ReplyToDingtalk(string(dingbot.MARKDOWN), "输入问题为空，请重新输入！")
				if err != nil {
					logger.Warning(fmt.Errorf("input message error: %v", err))
					return
				}
			} else {
				ProcessRequest(msg)
			}
		}
	}
}

func ProcessRequest(msgObj dingbot.ReceiveMsg) {
	// 去除问题的前后空格
	msgObj.Text.Content = strings.TrimSpace(msgObj.Text.Content)

	if len(msgObj.Text.Content) == 1 || msgObj.Text.Content == "帮助" {
		// 欢迎信息
		_, err := msgObj.ReplyToDingtalk(string(dingbot.MARKDOWN), public.Config.Help)
		if err != nil {
			logger.Warning(fmt.Errorf("send message error: %v", err))
			return
		}
	} else {
		// 除去帮助之外的逻辑分流在这里处理
		switch {
		case strings.HasPrefix(msgObj.Text.Content, "#查对话"):
			process.SelectHistory(&msgObj)
		case strings.HasPrefix(msgObj.Text.Content, "#标题设置"):
			process.SetTitle(&msgObj)
		default:
			msgObj.Text.Content, _ = process.GeneratePrompt(msgObj.Text.Content)
			process.ProcessRequest(&msgObj)
		}
	}
}

func router() {
	app := ship.Default()
	// 解析生成后的历史聊天
	app.Route("/history/:username").GET(func(c *ship.Context) error {
		username := c.Param("username")
		resp := process.OutPutHistory(username)
		return c.Blob(http.StatusOK, "text/markdown;charset=utf-8", []byte(resp))
	})
	// 直接下载文件
	app.Route("/download/:filename").GET(func(c *ship.Context) error {
		filename := c.Param("filename")
		root := "./data/chatHistory/"
		return c.Attachment(filepath.Join(root, filename), "")
	})
	// 服务器健康检测
	app.Route("/").GET(func(c *ship.Context) error {
		//返回消息优雅一点，告诉用户欢迎使用ding ding机器人服务 服务状态oK
		return c.JSON(http.StatusOK, map[string]interface{}{
			"status": "ok",
			"msg":    "欢迎使用chat-gpt机器人",
		})
	})
	port := ":" + public.Config.Port
	srv := &http.Server{
		Addr:    port,
		Handler: app,
	}

	// Initializing the server in a goroutine so that
	// it won't block the graceful shutdown handling below
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("listen: %s\n", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server with
	// a timeout of 5 seconds.
	quit := make(chan os.Signal, 1)
	// kill (no param) default send syscall.SIGTERM
	// kill -2 is syscall.SIGINT
	// kill -9 is syscall.SIGKILL but can't be catch, so don't need add it
	// signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	signal.Notify(quit, os.Interrupt)
	<-quit
	logger.Info("Shutting down server...")

	// 5秒后强制退出
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatal("Server forced to shutdown:", err)
	}
	logger.Info("Server exiting!")
}

func Start() {
	app := ship.Default()
	app.Route("/").POST(func(c *ship.Context) error {
		var msgObj dingbot.ReceiveMsg
		err := c.Bind(&msgObj)
		if err != nil {
			return ship.ErrBadRequest.New(fmt.Errorf("bind to receivemsg failed : %v", err))
		}
		// 先校验回调是否合法
		if !public.CheckRequest(c.GetReqHeader("timestamp"), c.GetReqHeader("sign")) && msgObj.SenderStaffId != "" {
			logger.Warning("该请求不合法，可能是其他企业或者未经允许的应用调用所致，请知悉！")
			return nil
		} else if !public.JudgeOutgoingGroup(msgObj.ConversationID) && msgObj.SenderStaffId == "" {
			logger.Warning("该请求不合法，可能是未经允许的普通群outgoing机器人调用所致，请知悉！")
			return nil
		}
		// 再校验回调参数是否有价值
		if msgObj.Text.Content == "" || msgObj.ChatbotUserID == "" {
			logger.Warning("从钉钉回调过来的内容为空，根据过往的经验，或许重新创建一下机器人，能解决这个问题")
			return ship.ErrBadRequest.New(fmt.Errorf("从钉钉回调过来的内容为空，根据过往的经验，或许重新创建一下机器人，能解决这个问题"))
		}
		// 去除问题的前后空格
		msgObj.Text.Content = strings.TrimSpace(msgObj.Text.Content)
		if public.JudgeSensitiveWord(msgObj.Text.Content) {
			_, err = msgObj.ReplyToDingtalk(string(dingbot.MARKDOWN), "**🤷 抱歉，您提问的问题中包含敏感词汇，请审核自己的对话内容之后再进行！**")
			if err != nil {
				logger.Warning(fmt.Errorf("send message error: %v", err))
				return err
			}
			return nil
		}
		// 打印钉钉回调过来的请求明细，调试时打开
		logger.Debug(fmt.Sprintf("dingtalk callback parameters: %#v", msgObj))

		if public.Config.ChatType != "0" && msgObj.ConversationType != public.Config.ChatType {
			logger.Info(fmt.Sprintf("🙋 %s使用了禁用的聊天方式", msgObj.SenderNick))
			_, err = msgObj.ReplyToDingtalk(string(dingbot.MARKDOWN), "**🤷 抱歉，管理员禁用了这种聊天方式，请选择其他聊天方式与机器人对话！**")
			if err != nil {
				logger.Warning(fmt.Errorf("send message error: %v", err))
				return err
			}
			return nil
		}

		// 查询群ID，发送指令后，可通过查看日志来获取
		if msgObj.ConversationType == "2" && msgObj.Text.Content == "群ID" {
			if msgObj.RobotCode == "normal" {
				logger.Info(fmt.Sprintf("🙋 outgoing机器人 在『%s』群的ConversationID为: %#v", msgObj.ConversationTitle, msgObj.ConversationID))
			} else {
				logger.Info(fmt.Sprintf("🙋 企业内部机器人 在『%s』群的ConversationID为: %#v", msgObj.ConversationTitle, msgObj.ConversationID))
			}
			//_, err = msgObj.ReplyToDingtalk(string(dingbot.MARKDOWN), msgObj.ConversationID)
			if err != nil {
				logger.Warning(fmt.Errorf("send message error: %v", err))
				return err
			}
			return nil
		}

		// 不在允许群组，不在允许用户（包括在黑名单），满足任一条件，拒绝会话；管理员不受限制
		if msgObj.ConversationType == "2" && !public.JudgeGroup(msgObj.ConversationID) && !public.JudgeAdminUsers(msgObj.SenderStaffId) && msgObj.SenderStaffId != "" {
			logger.Info(fmt.Sprintf("🙋『%s』群组未被验证通过，群ID: %#v，userid：%#v, 昵称: %#v，消息: %#v", msgObj.ConversationTitle, msgObj.ConversationID, msgObj.SenderStaffId, msgObj.SenderNick, msgObj.Text.Content))
			_, err = msgObj.ReplyToDingtalk(string(dingbot.MARKDOWN), "**🤷 抱歉，该群组未被认证通过，无法使用机器人对话功能。**\n>如需继续使用，请联系管理员申请访问权限。")
			if err != nil {
				logger.Warning(fmt.Errorf("send message error: %v", err))
				return err
			}
			return nil
		} else if !public.JudgeUsers(msgObj.SenderStaffId) && !public.JudgeAdminUsers(msgObj.SenderStaffId) && msgObj.SenderStaffId != "" {
			logger.Info(fmt.Sprintf("🙋 %s身份信息未被验证通过，userid：%#v，消息: %#v", msgObj.SenderNick, msgObj.SenderStaffId, msgObj.Text.Content))
			_, err = msgObj.ReplyToDingtalk(string(dingbot.MARKDOWN), "**🤷 抱歉，您的身份信息未被认证通过，无法使用机器人对话功能。**\n>如需继续使用，请联系管理员申请访问权限。")
			if err != nil {
				logger.Warning(fmt.Errorf("send message error: %v", err))
				return err
			}
			return nil
		}
		if len(msgObj.Text.Content) == 0 || msgObj.Text.Content == "帮助" {
			// 欢迎信息
			_, err := msgObj.ReplyToDingtalk(string(dingbot.MARKDOWN), public.Config.Help)
			if err != nil {
				logger.Warning(fmt.Errorf("send message error: %v", err))
				return ship.ErrBadRequest.New(fmt.Errorf("send message error: %v", err))
			}
		} else {
			logger.Info(fmt.Sprintf("🙋 %s发起的问题: %#v", msgObj.SenderNick, msgObj.Text.Content))
			// 除去帮助之外的逻辑分流在这里处理
			switch {
			case strings.HasPrefix(msgObj.Text.Content, "#图片"):
				return process.ImageGenerate(&msgObj)
			case strings.HasPrefix(msgObj.Text.Content, "#查对话"):
				return process.SelectHistory(&msgObj)
			case strings.HasPrefix(msgObj.Text.Content, "#域名"):
				return process.DomainMsg(&msgObj)
			case strings.HasPrefix(msgObj.Text.Content, "#证书"):
				return process.DomainCertMsg(&msgObj)
			default:
				msgObj.Text.Content, err = process.GeneratePrompt(msgObj.Text.Content)
				// err不为空：提示词之后没有文本 -> 直接返回提示词所代表的内容
				if err != nil {
					_, err = msgObj.ReplyToDingtalk(string(dingbot.TEXT), msgObj.Text.Content)
					if err != nil {
						logger.Warning(fmt.Errorf("send message error: %v", err))
						return err
					}
					return nil
				}
				return process.ProcessRequest(&msgObj)
			}
		}
		return nil
	})
	// 解析生成后的图片
	app.Route("/images/:filename").GET(func(c *ship.Context) error {
		filename := c.Param("filename")
		root := "./data/images/"
		return c.File(filepath.Join(root, filename))
	})
	// 解析生成后的历史聊天
	app.Route("/history/:filename").GET(func(c *ship.Context) error {
		filename := c.Param("filename")
		root := "./data/chatHistory/"
		return c.File(filepath.Join(root, filename))
	})
	// 直接下载文件
	app.Route("/download/:filename").GET(func(c *ship.Context) error {
		filename := c.Param("filename")
		root := "./data/chatHistory/"
		return c.Attachment(filepath.Join(root, filename), "")
	})
	// 服务器健康检测
	app.Route("/").GET(func(c *ship.Context) error {
		//返回消息优雅一点，告诉用户欢迎使用ding ding机器人服务 服务状态oK
		return c.JSON(http.StatusOK, map[string]interface{}{
			"status": "ok",
			"msg":    "欢迎使用钉钉机器人",
		})
	})
	port := ":" + public.Config.Port
	srv := &http.Server{
		Addr:    port,
		Handler: app,
	}

	// Initializing the server in a goroutine so that
	// it won't block the graceful shutdown handling below
	go func() {
		logger.Info("🚀 The HTTP Server is running on", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("listen: %s\n", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server with
	// a timeout of 5 seconds.
	quit := make(chan os.Signal, 1)
	// kill (no param) default send syscall.SIGTERM
	// kill -2 is syscall.SIGINT
	// kill -9 is syscall.SIGKILL but can't be catch, so don't need add it
	// signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	signal.Notify(quit, os.Interrupt)
	<-quit
	logger.Info("Shutting down server...")

	// 5秒后强制退出
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatal("Server forced to shutdown:", err)
	}
	logger.Info("Server exiting!")
}
