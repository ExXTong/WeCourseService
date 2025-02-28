package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"unsafe"
)

// 配置结构
type Config struct {
	SchoolName    string
	MangerType    string
	MangerURL     string
	SocketPort    int
	CalendarFirst string // 学期开始日期
}

// 将bytes转换为string，无需内存分配
func B2S(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}

// 从cookie字符串解析出http.Cookie对象数组
func parseCookieString(cookieStr string) []*http.Cookie {
	// 确保有效的cookie字符串
	if cookieStr == "" {
		return nil
	}

	var cookies []*http.Cookie
	parts := strings.Split(cookieStr, ";")

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		// 分割键值对
		pair := strings.SplitN(part, "=", 2)
		if len(pair) != 2 {
			continue
		}

		cookie := &http.Cookie{
			Name:  strings.TrimSpace(pair[0]),
			Value: strings.TrimSpace(pair[1]),
		}
		cookies = append(cookies, cookie)
	}

	return cookies
}

// 生成通用错误响应JSON
func getErrorResponse(errorType string, message string) string {
	errorData := map[string]interface{}{
		"Type": errorType,
		"Data": message,
	}

	js, err := json.MarshalIndent(errorData, "", "\t")
	if err != nil {
		// 如果JSON序列化失败，返回简单字符串
		return `{"Type":"` + errorType + `","Data":"内部错误"}`
	}

	return B2S(js)
}
