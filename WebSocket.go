package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync"

	"github.com/gorilla/websocket"
)

type userlogin struct {
	Type   string
	Cookie string // 替换 UserName 和 PassWord，直接使用 Cookie
	Week   int
}

var build string = "202502281030-SecurityEnhanced"
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// 添加连接管理
var (
	clients = make(map[*websocket.Conn]bool)
	mutex   = &sync.Mutex{}
)

func StartWebSocket() {
	fmt.Println("Websocket服务开始运行")
	fmt.Println("固件版本：" + build)
	conf := ReadConfig()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// 正确处理WebSocket升级错误
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println("WebSocket升级失败:", err)
			return
		}
		defer func() {
			// 安全关闭连接
			mutex.Lock()
			delete(clients, conn)
			mutex.Unlock()
			conn.Close()
			log.Println("WebSocket连接已关闭")
		}()

		// 注册客户端
		mutex.Lock()
		clients[conn] = true
		mutex.Unlock()

		log.Println("新的WebSocket连接建立")

		for {
			// 正确处理消息读取错误
			msgType, msg, err := conn.ReadMessage()
			if err != nil {
				log.Println("读取消息错误:", err)
				break // 退出循环，连接将被defer关闭
			}

			// 处理空消息
			if len(msg) == 0 {
				continue
			}

			var u userlogin
			if err := json.Unmarshal(msg, &u); err != nil {
				log.Println("JSON解析失败:", err)
				continue
			}

			// 使用安全的响应写入方法
			sendResponse := func(data string) {
				if err := conn.WriteMessage(msgType, []byte(data)); err != nil {
					log.Println("消息发送失败:", err)
				}
			}

			switch u.Type {
			case "allcourse":
				sendResponse(GetCourse(u.Cookie))
			case "account":
				sendResponse(GetAccount(u.Cookie))
			case "week":
				sendResponse(GetWeekTime(conf.CalendarFirst))
			case "teacher":
				sendResponse(GetTeacher(u.Cookie))
			case "photo":
				sendResponse(GetPhoto(u.Cookie))
			case "grade":
				sendResponse(GetGrade(u.Cookie))
			default:
				log.Println("未知的消息类型:", u.Type)
			}
		}
	})

	// 添加日志记录服务启动信息
	log.Println("WebSocket服务器正在启动，端口:", conf.SocketPort)
	if err := http.ListenAndServe(":"+strconv.Itoa(conf.SocketPort), nil); err != nil {
		log.Fatal("ListenAndServe失败:", err)
	}
}
