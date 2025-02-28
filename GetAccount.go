package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strings"
)

type StudentStruct struct {
	FullName    string
	EnglishName string
	Sex         string
	StartTime   string
	EndTime     string
	SchoolYear  string
	Type        string
	System      string
	Specialty   string
	Class       string
}

func GetAccount(cookieStr string) string {
	// 初始化结果变量
	var myStudent StudentStruct
	var myAccountResult AccountResult
	myAccountResult.Type = "account"
	conf := ReadConfig()

	// 从字符串解析 cookie
	cookies := parseCookieString(cookieStr)
	if len(cookies) == 0 {
		return getErrorResponse("account", "Cookie无效")
	}

	// 创建 HTTP 客户端
	var client http.Client

	// 设置 cookie jar
	jar, err := cookiejar.New(nil)
	if err != nil {
		fmt.Println("ERROR_0: ", err.Error())
		return getErrorResponse("account", "创建Cookie容器失败")
	}

	// 将解析的cookies添加到jar中
	u, _ := url.Parse(conf.MangerURL)
	jar.SetCookies(u, cookies)
	client.Jar = jar

	// 直接请求个人详情页面
	req, err := http.NewRequest(http.MethodGet, conf.MangerURL+"eams/stdDetail.action", nil)
	if err != nil {
		fmt.Println("ERROR_1: ", err.Error())
		return getErrorResponse("account", "创建请求失败")
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64; rv:66.0) Gecko/20100101 Firefox/66.0")

	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("ERROR_2: ", err.Error())
		return getErrorResponse("account", "网络请求失败")
	}
	defer resp.Body.Close()

	// 检查是否重定向到登录页面(表示session已过期)
	if strings.Contains(resp.Request.URL.Path, "login.action") {
		return getErrorResponse("account", "登录已过期，请重新登录")
	}

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("ERROR_3: ", err.Error())
		return getErrorResponse("account", "读取响应失败")
	}

	temp := string(content)

	// 解析学生信息
	reg := regexp.MustCompile(`(?i)<td>([^>]*)</td>`)
	stuinfo := reg.FindAllStringSubmatch(temp, -1)

	if len(stuinfo) < 19 { // 确保获取了足够的信息
		return getErrorResponse("account", "解析个人信息失败")
	}

	// 提取各项信息
	myStudent.FullName = stuinfo[0][1]
	myStudent.EnglishName = stuinfo[1][1]
	myStudent.Sex = stuinfo[2][1]
	myStudent.SchoolYear = stuinfo[4][1]
	myStudent.Type = stuinfo[5][1]
	if len(stuinfo) > 14 {
		myStudent.Type += "(" + stuinfo[14][1] + ")"
	}
	myStudent.StartTime = stuinfo[11][1]
	myStudent.EndTime = stuinfo[12][1]
	myStudent.System = stuinfo[8][1]
	myStudent.Specialty = stuinfo[9][1]
	if len(stuinfo) > 18 {
		myStudent.Class = stuinfo[18][1]
	}

	// 将结果打包为JSON返回
	myAccountResult.Data = myStudent
	js, err := json.MarshalIndent(myAccountResult, "", "\t")
	if err != nil {
		return getErrorResponse("account", "生成JSON失败")
	}

	return B2S(js)
}

// 删除重复定义的辅助函数
// 这些函数已经在utils.go中定义
