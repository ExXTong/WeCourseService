package main

import (
	"fmt"
	"strconv"
)

func main() {
	conf := ReadConfig() // 使用统一的ReadConfig函数
	fmt.Println("学校名称：" + conf.SchoolName)
	switch conf.MangerType {
	case "supwisdom":
		fmt.Println("教务系统：树维教务系统")
	}
	fmt.Println("绑定端口：" + strconv.Itoa(conf.SocketPort))
	StartWebSocket()
}
