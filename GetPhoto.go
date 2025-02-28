package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strings"
)

func GetPhoto(cookieStr string) string {
	conf := ReadConfig()
	var myPhotoResult PhotoResult
	myPhotoResult.Type = "photo"

	// 从字符串解析 cookie
	cookies := parseCookieString(cookieStr)
	if len(cookies) == 0 {
		return getErrorResponse("photo", "Cookie无效")
	}

	// 创建 HTTP 客户端
	var client http.Client

	// 设置 cookie jar
	jar, err := cookiejar.New(nil)
	if err != nil {
		fmt.Println("ERROR_0: ", err.Error())
		return getErrorResponse("photo", "创建Cookie容器失败")
	}

	u, _ := url.Parse(conf.MangerURL)
	jar.SetCookies(u, cookies)
	client.Jar = jar

	// 获取用户名信息（从 cookie 中或请求个人信息页面）
	userName := getUserNameFromCookie(client, conf)
	if userName == "" {
		return getErrorResponse("photo", "无法获取用户信息，Cookie可能已过期")
	}

	// 请求用户照片
	req, err := http.NewRequest(http.MethodGet, conf.MangerURL+"eams/showSelfAvatar.action?user.name="+userName, nil)
	if err != nil {
		fmt.Println("ERROR_1: ", err.Error())
		return getErrorResponse("photo", "创建请求失败")
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64; rv:66.0) Gecko/20100101 Firefox/66.0")

	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("ERROR_2: ", err.Error())
		return getErrorResponse("photo", "网络连接错误")
	}
	defer resp.Body.Close()

	// 检查是否重定向到登录页面(表示session已过期)
	if strings.Contains(resp.Request.URL.Path, "login.action") {
		return getErrorResponse("photo", "登录已过期，请重新登录")
	}

	// 检查请求是否成功
	if resp.StatusCode != http.StatusOK {
		return getErrorResponse("photo", fmt.Sprintf("图像请求返回错误状态: %d", resp.StatusCode))
	}

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("ERROR_3: ", err.Error())
		return getErrorResponse("photo", "读取数据错误")
	}

	// 检查内容是否为图像
	contentType := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "image/") && len(content) < 100 {
		return getErrorResponse("photo", "返回内容不是有效图像")
	}

	// 将照片编码为 base64
	temp := base64.StdEncoding.EncodeToString(content)
	myPhotoResult.Data = "data:image/jpg;base64," + temp
	js, err := json.MarshalIndent(myPhotoResult, "", "\t")
	if err != nil {
		return getErrorResponse("photo", "生成JSON失败")
	}

	return B2S(js)
}

// 从 cookie 或个人页面获取用户名
func getUserNameFromCookie(client http.Client, conf Config) string {
	// 尝试请求个人信息页面获取用户名
	req, err := http.NewRequest(http.MethodGet, conf.MangerURL+"eams/security/my.action", nil)
	if err != nil {
		fmt.Println("ERROR_4: ", err.Error())
		return ""
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64; rv:66.0) Gecko/20100101 Firefox/66.0")

	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("ERROR_5: ", err.Error())
		return ""
	}
	defer resp.Body.Close()

	// 检查是否重定向到登录页面(表示session已过期)
	if strings.Contains(resp.Request.URL.Path, "login.action") {
		fmt.Println("ERROR_6: Session expired")
		return ""
	}

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("ERROR_7: ", err.Error())
		return ""
	}

	// 从个人页面提取用户名
	tempStr := string(content)

	// 通常用户名会显示在个人主页上，这里通过正则表达式提取
	// 示例模式，可能需要根据实际页面结构调整
	patterns := []string{
		`用户名：\s*(\w+)`,
		`username['"]*>\s*([^<]+)`,
		`user\.name['"]*>\s*([^<]+)`,
		`user\.name=([^&"]+)`,
		`userName['"]*>\s*([^<]+)`,
	}

	for _, pattern := range patterns {
		reg := regexp.MustCompile(pattern)
		matches := reg.FindStringSubmatch(tempStr)
		if len(matches) >= 2 && matches[1] != "" {
			return matches[1]
		}
	}

	fmt.Println("ERROR_8: Cannot extract username")
	return ""
}

// 删除重复定义的辅助函数
// 这些函数已经在utils.go中定义
