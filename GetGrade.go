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
	"time"
)

type GradeStruct struct {
	CourseID     string
	CourseName   string
	CourseTerm   string
	CourseCredit string
	CourseGrade  string
	GradePoint   string
}

func GetGrade(cookieStr string) string {
	// 获取配置
	conf := ReadConfig()

	// 初始化结果变量
	var grades []GradeStruct
	var myGrade GradeStruct
	var gradeResult GradeResult
	gradeResult.Type = "grade"

	// 解析 cookie 字符串
	cookies := parseCookieString(cookieStr)
	if len(cookies) == 0 {
		fmt.Println("没有解析到Cookie，原始Cookie字符串:", cookieStr)
		return getErrorResponse("grade", "Cookie无效")
	}

	// 输出解析后的Cookie，用于调试
	fmt.Println("解析到", len(cookies), "个Cookie:")
	for i, cookie := range cookies {
		fmt.Printf("  Cookie %d: %s=%s\n", i, cookie.Name, cookie.Value)
	}

	// 创建 HTTP 客户端
	var client http.Client

	// 设置 cookie jar
	jar, err := cookiejar.New(nil)
	if err != nil {
		fmt.Println("ERROR_0: ", err.Error())
		return getErrorResponse("grade", "创建Cookie容器失败")
	}

	// 将解析的cookies添加到jar中
	u, _ := url.Parse(conf.MangerURL)
	jar.SetCookies(u, cookies)
	client.Jar = jar

	// 先访问主页面来初始化会话
	mainReq, _ := http.NewRequest(http.MethodGet, conf.MangerURL+"/eams/teach/grade/course/person.action", nil)
	mainReq.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/133.0.0.0 Safari/537.36")
	mainReq.Header.Set("Referer", conf.MangerURL)
	mainReq.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7")
	mainReq.Header.Set("Accept-Language", "zh-CN,zh;q=0.9")
	mainReq.Header.Set("Upgrade-Insecure-Requests", "1")

	mainResp, err := client.Do(mainReq)
	if err != nil {
		fmt.Println("主页面请求失败:", err.Error())
		return getErrorResponse("grade", "主页面请求失败")
	}
	defer mainResp.Body.Close()

	// 检查主页面是否返回登录页
	if strings.Contains(mainResp.Request.URL.Path, "login") {
		fmt.Println("主页面请求被重定向到登录页")
		return getErrorResponse("grade", "登录已过期，请重新登录")
	}

	mainContent, _ := io.ReadAll(mainResp.Body)
	mainHtml := string(mainContent)
	if strings.Contains(mainHtml, "统一身份认证平台") {
		fmt.Println("主页面返回了登录页HTML")
		return getErrorResponse("grade", "登录已过期，请重新登录")
	}

	// 直接请求成绩数据的AJAX接口，与浏览器使用相同的查询参数
	timestamp := fmt.Sprintf("%d", time.Now().UnixNano()/1000000)
	req, err := http.NewRequest(http.MethodGet, conf.MangerURL+"/eams/teach/grade/course/person!search.action?semesterId=173&projectType=&_="+timestamp, nil)
	if err != nil {
		fmt.Println("ERROR_13: ", err.Error())
		return getErrorResponse("grade", "创建请求失败")
	}

	// 设置头信息，完全匹配浏览器AJAX请求
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/133.0.0.0 Safari/537.36")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Header.Set("Accept", "text/html, */*; q=0.01")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9")
	req.Header.Set("Referer", conf.MangerURL+"/eams/teach/grade/course/person.action")
	req.Header.Set("Sec-Ch-Ua", `"Chromium";v="133", "Not(A:Brand";v="99"`)
	req.Header.Set("Sec-Ch-Ua-Mobile", "?0")
	req.Header.Set("Sec-Ch-Ua-Platform", "Windows")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("Sec-Fetch-Mode", "cors")
	req.Header.Set("Sec-Fetch-Dest", "empty")

	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("ERROR_14: ", err.Error())
		return getErrorResponse("grade", "网络请求失败")
	}
	defer resp.Body.Close()

	// 检查是否重定向到登录页面(表示session已过期)
	if strings.Contains(resp.Request.URL.Path, "login") {
		fmt.Println("成绩请求被重定向到:", resp.Request.URL.Path)
		return getErrorResponse("grade", "登录已过期，请重新登录")
	}

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("ERROR_15: ", err.Error())
		return getErrorResponse("grade", "读取响应失败")
	}
	temp := string(content)

	// 检查响应内容是否为登录页
	if strings.Contains(temp, "统一身份认证平台") {
		fmt.Println("响应内容包含登录页标识")
		return getErrorResponse("grade", "登录已过期，请重新登录")
	}

	// 检查是否为404页面
	if strings.Contains(temp, "404") && strings.Contains(temp, "无法找到页面") {
		fmt.Println("响应内容是404页面")
		return getErrorResponse("grade", "页面不存在，请检查URL")
	}

	// 使用更准确的正则表达式匹配表格中的行
	reg3 := regexp.MustCompile(`<tr>\s*<td[^>]*>(.*?)</td>\s*<td[^>]*>(.*?)</td>\s*<td[^>]*>(.*?)</td>\s*<td[^>]*>(.*?)</td>\s*<td[^>]*>(.*?)</td>\s*<td[^>]*>(.*?)</td>\s*<td[^>]*>\s*(.*?)\s*</td>\s*<td[^>]*>\s*(.*?)\s*</td>\s*</tr>`)

	// 找到所有匹配的行
	matches := reg3.FindAllStringSubmatch(temp, -1)

	if len(matches) == 0 {
		// 输出前200个字符，帮助调试
		preview := temp
		if len(temp) > 200 {
			preview = temp[:200]
		}
		fmt.Println("未匹配到数据，响应前200字符:", preview)

		// 尝试使用更宽松的正则表达式
		reg4 := regexp.MustCompile(`<tr[^>]*>[\s\S]*?</tr>`)
		rows := reg4.FindAllString(temp, -1)
		if len(rows) > 0 {
			fmt.Println("找到", len(rows), "行数据，但无法使用主正则表达式解析")
			fmt.Println("第一行数据:", rows[0])
		}

		return getErrorResponse("grade", "没有找到成绩数据")
	}

	// 解析每行数据
	for _, match := range matches {
		if len(match) < 9 {
			continue
		}

		myGrade = GradeStruct{
			CourseTerm:   strings.TrimSpace(match[1]),
			CourseID:     strings.TrimSpace(match[2]),
			CourseName:   strings.TrimSpace(match[4]),
			CourseCredit: strings.TrimSpace(match[6]),
			CourseGrade:  strings.TrimSpace(match[7]),
			GradePoint:   strings.TrimSpace(match[8]),
		}
		grades = append(grades, myGrade)
	}

	// 将结果包装为 JSON
	gradeResult.Data = grades
	js, err := json.MarshalIndent(gradeResult, "", "\t")
	if err != nil {
		return getErrorResponse("grade", "生成JSON失败")
	}

	return B2S(js)
}
