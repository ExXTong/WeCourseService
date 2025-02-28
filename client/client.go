package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"

	"github.com/gorilla/websocket"
)

// 请求结构体
type Request struct {
	Type string `json:"Type"`
	Week int    `json:"Week,omitempty"`
}

// 响应结构体
type Response struct {
	Type string      `json:"Type"`
	Data interface{} `json:"Data"`
}

func main() {
	// 命令行参数
	serverAddr := flag.String("server", "localhost:25565", "服务器地址 (host:port)")
	flag.Parse()

	// 连接WebSocket服务器
	url := fmt.Sprintf("ws://%s", *serverAddr)
	fmt.Printf("正在连接到 %s...\n", url)

	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		log.Fatalf("连接失败: %v", err)
	}
	defer conn.Close()

	fmt.Println("连接成功！")

	// 处理系统中断信号
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	// 输出接收的消息
	go func() {
		for {
			var resp Response
			err := conn.ReadJSON(&resp)
			if err != nil {
				log.Printf("读取响应错误: %v", err)
				return
			}
			// 美化输出JSON
			jsonData, err := json.MarshalIndent(resp.Data, "", "    ")
			if err != nil {
				log.Printf("解析响应数据错误: %v", err)
				continue
			}
			fmt.Printf("\n收到响应 (类型: %s):\n%s\n\n> ", resp.Type, string(jsonData))
		}
	}()

	// 命令行交互
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("\n可用命令:")
	fmt.Println("1. week - 获取当前教学周")
	fmt.Println("2. teacher - 获取教师列表")
	fmt.Println("3. account - 获取学籍信息")
	fmt.Println("4. course [周数] - 获取课程表 (0表示获取全部课程表)")
	fmt.Println("5. photo - 获取学籍照片")
	fmt.Println("6. grade - 获取成绩")
	fmt.Println("7. exit - 退出程序")
	fmt.Println("\n示例: course 10")

	// 存储Cookie以便重复使用
	var userCookie string

	for {
		fmt.Print("> ")
		input, err := reader.ReadString('\n')
		if err != nil {
			log.Printf("读取输入错误: %v", err)
			continue
		}

		input = strings.TrimSpace(input)
		args := strings.Fields(input)

		if len(args) == 0 {
			continue
		}

		if args[0] == "exit" {
			break
		}

		req := Request{Type: args[0]}

		switch args[0] {
		case "week":
			// 不需要额外参数

		case "login", "teacher", "account", "photo", "grade":
			// 需要Cookie认证
			if userCookie == "" {
				fmt.Print("请输入Cookie: ")
				cookie, err := reader.ReadString('\n')
				if err != nil {
					log.Printf("读取Cookie错误: %v", err)
					continue
				}
				userCookie = strings.TrimSpace(cookie)
			}

			// 重新连接WebSocket并发送Cookie头
			if conn != nil {
				conn.Close()
			}

			header := http.Header{}
			header.Add("Cookie", userCookie)
			conn, _, err = websocket.DefaultDialer.Dial(url, header)
			if err != nil {
				log.Printf("连接失败: %v", err)
				// 清除Cookie并重试
				userCookie = ""
				continue
			}

		case "course":
			if len(args) < 2 {
				fmt.Println("请指定周数，例如: course 10 或 course 0 (获取全部课程表)")
				continue
			}
			week, err := strconv.Atoi(args[1])
			if err != nil {
				fmt.Println("错误: 周数必须是数字")
				continue
			}
			req.Week = week

			// 需要Cookie认证
			if userCookie == "" {
				fmt.Print("请输入Cookie: ")
				cookie, err := reader.ReadString('\n')
				if err != nil {
					log.Printf("读取Cookie错误: %v", err)
					continue
				}
				userCookie = strings.TrimSpace(cookie)
			}

			// 重新连接WebSocket并发送Cookie头
			if conn != nil {
				conn.Close()
			}

			header := http.Header{}
			header.Add("Cookie", userCookie)
			conn, _, err = websocket.DefaultDialer.Dial(url, header)
			if err != nil {
				log.Printf("连接失败: %v", err)
				// 清除Cookie并重试
				userCookie = ""
				continue
			}

		default:
			fmt.Println("未知命令，请重试")
			continue
		}

		// 发送请求
		err = conn.WriteJSON(req)
		if err != nil {
			log.Printf("发送请求错误: %v", err)
			continue
		}

		fmt.Printf("已发送 %s 请求...\n", req.Type)
	}

	fmt.Println("退出程序")
}
