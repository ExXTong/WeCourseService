package main

import (
	"encoding/json"
	"fmt"
	"os"
)

// ReadConfig 从config.json文件读取配置
func ReadConfig() Config {
	var config Config

	// 读取配置文件
	file, err := os.Open("config.json")
	if err != nil {
		fmt.Println("警告: 无法打开配置文件，使用默认配置:", err)
		return getDefaultConfig()
	}
	defer file.Close()

	// 解析JSON
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&config)
	if err != nil {
		fmt.Println("警告: 解析配置文件失败，使用默认配置:", err)
		return getDefaultConfig()
	}

	// 验证必要的配置值
	if config.MangerURL == "" || config.SocketPort == 0 {
		fmt.Println("警告: 配置文件缺少必要参数，使用默认值补充")
		defaultConfig := getDefaultConfig()

		if config.MangerURL == "" {
			config.MangerURL = defaultConfig.MangerURL
		}
		if config.SocketPort == 0 {
			config.SocketPort = defaultConfig.SocketPort
		}
		if config.CalendarFirst == "" {
			config.CalendarFirst = defaultConfig.CalendarFirst
		}
	}

	return config
}

// 获取默认配置
func getDefaultConfig() Config {
	return Config{
		SchoolName:    "某某大学",
		MangerType:    "supwisdom",
		MangerURL:     "http://jwxt.example.edu/",
		SocketPort:    8080,
		CalendarFirst: "2023-09-01",
	}
}
