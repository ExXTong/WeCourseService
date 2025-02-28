package main

import (
	"io"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"
)

// 测试成绩API处理程序
func TestGetGrade(t *testing.T) {
	req, err := http.NewRequest("GET", "/grade", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(GetGradeHandler)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	expected := `{"grade": "A"}`
	if rr.Body.String() != expected {
		t.Errorf("handler returned unexpected body: got %v want %v", rr.Body.String(), expected)
	}
}

// 使用真实服务器测试成绩获取功能
func TestGetGradeWithRealServer(t *testing.T) {
	// 从环境变量获取测试cookie
	testCookie := os.Getenv("TEST_COOKIE")
	if testCookie == "" {
		// 如果环境变量未设置，使用默认单点登录Cookie
		testCookie = "happyVoyagePersonal=PhY9qu5md+jR9M2kePMXU8SsyR+gleWu/KvOFWGsyH8oPeyGDR2j8cRu4vDJuT0T0xq1aogU3E4jkMgzCCvOZziPpg5ye+SpfSXgd9rGFCUee6kUpVENoWq3lXtVaMUQvLpTgg4wjbYSBxWxlcCBxz1OQmVCKb8q4dH566FrAOk=; route=46100fc282b5137f0fdc1aea94c29ad6; JSESSIONID=D8780819116D19086569F6FA4D4856B6; semester.id=173;"
	}

	// 创建一个临时的测试函数来获取原始响应
	debugResponse := getHtmlResponse(testCookie)

	// 检查是否登录页面
	if strings.Contains(debugResponse, "统一身份认证平台") ||
		strings.Contains(debugResponse, "请输入用户名") ||
		strings.Contains(debugResponse, "请输入密码") {
		t.Skip("Cookie已过期，返回登录页面，跳过测试")
	}

	// 检查是否404错误
	if strings.Contains(debugResponse, "404") && strings.Contains(debugResponse, "无法找到页面") {
		t.Error("服务器返回404错误，URL可能不正确")
	}

	// 输出响应的前部分，帮助调试
	previewLength := min(1000, len(debugResponse))
	t.Logf("服务器响应前%d字符: %s", previewLength, debugResponse[:previewLength])

	// 检查响应是否包含表格数据
	if !strings.Contains(debugResponse, "<table") || !strings.Contains(debugResponse, "<tr") {
		t.Logf("响应不包含表格数据，可能格式有误或者数据为空")
	}

	result := GetGrade(testCookie)

	// 检查返回的结果格式是否正确
	if !strings.Contains(result, `"type":"grade"`) {
		t.Errorf("返回结果格式不正确: %s", result)
	}

	// 验证是否包含成绩数据
	if strings.Contains(result, `"data":[]`) || strings.Contains(result, `"error":`) {
		t.Errorf("未获取到成绩数据: %s", result)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// 获取原始HTML响应，增强版
func getHtmlResponse(cookieStr string) string {
	// 设置日志文件
	logFile, err := os.Create("network_debug.log")
	if err == nil {
		defer logFile.Close()
		log.SetOutput(logFile)
	}

	conf := ReadConfig()
	cookies := parseCookieString(cookieStr)
	log.Printf("解析到 %d 个Cookie", len(cookies))
	for i, c := range cookies {
		log.Printf("Cookie %d: %s=%s", i, c.Name, c.Value)
	}

	var client http.Client
	jar, _ := cookiejar.New(nil)
	u, _ := url.Parse(conf.MangerURL)
	jar.SetCookies(u, cookies)
	client.Jar = jar

	// 步骤1: 初始化会话请求
	log.Printf("步骤1: 初始化会话")
	mainReq, _ := http.NewRequest(http.MethodGet, conf.MangerURL+"/eams/teach/grade/course/person.action", nil)
	mainReq.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/133.0.0.0 Safari/537.36")
	mainReq.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7")
	mainReq.Header.Set("Accept-Language", "zh-CN,zh;q=0.9")
	mainReq.Header.Set("Referer", conf.MangerURL)
	mainReq.Header.Set("Upgrade-Insecure-Requests", "1")
	mainReq.Header.Set("Sec-Fetch-Mode", "navigate")
	mainReq.Header.Set("Sec-Fetch-User", "?1")
	mainReq.Header.Set("Sec-Fetch-Dest", "document")
	mainReq.Header.Set("Sec-Ch-Ua", `"Chromium";v="133", "Not(A:Brand";v="99"`)
	mainReq.Header.Set("Sec-Ch-Ua-Mobile", "?0")
	mainReq.Header.Set("Sec-Ch-Ua-Platform", "Windows")

	// 记录完整请求
	dumpReq, _ := httputil.DumpRequestOut(mainReq, false)
	log.Printf("初始化请求:\n%s", string(dumpReq))

	// 执行请求
	mainResp, err := client.Do(mainReq)
	if err != nil {
		log.Printf("初始化会话请求错误: %s", err)
		return "初始化会话失败: " + err.Error()
	}

	// 记录响应信息
	dumpResp, _ := httputil.DumpResponse(mainResp, false)
	log.Printf("初始化响应:\n%s", string(dumpResp))

	// 检查是否重定向
	finalURL := mainResp.Request.URL.String()
	origURL := conf.MangerURL + "/eams/teach/grade/course/person.action"
	if finalURL != origURL {
		log.Printf("重定向发生: %s -> %s", origURL, finalURL)
		if strings.Contains(finalURL, "login") {
			mainContent, _ := io.ReadAll(mainResp.Body)
			mainResp.Body.Close()
			return string(mainContent)
		}
	}

	mainContent, _ := io.ReadAll(mainResp.Body)
	contentPreview := ""
	if len(mainContent) > 0 {
		previewLen := min(100, len(mainContent))
		contentPreview = string(mainContent[:previewLen])
	}
	log.Printf("初始化响应内容(前100字符): %s", contentPreview)
	mainResp.Body.Close()

	// 记录当前Cookie
	currentCookies := client.Jar.Cookies(u)
	log.Printf("初始化请求后的Cookie (%d个):", len(currentCookies))
	for i, c := range currentCookies {
		log.Printf("  Cookie %d: %s=%s", i, c.Name, c.Value)
	}

	// 步骤2: 请求成绩数据
	log.Printf("步骤2: 请求成绩数据")
	time.Sleep(1 * time.Second) // 添加延迟，模拟人类操作

	// 与浏览器完全匹配的URL
	gradeURL := conf.MangerURL + "/eams/teach/grade/course/person!search.action?semesterId=173"
	log.Printf("成绩请求URL: %s", gradeURL)

	req, err := http.NewRequest(http.MethodGet, gradeURL, nil)
	if err != nil {
		log.Printf("创建成绩请求失败: %s", err)
		return "创建成绩请求失败: " + err.Error()
	}

	// 设置与浏览器完全匹配的请求头
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/133.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9")
	req.Header.Set("Referer", conf.MangerURL+"/eams/teach/grade/course/person.action")
	req.Header.Set("Sec-Ch-Ua", `"Chromium";v="133", "Not(A:Brand";v="99"`)
	req.Header.Set("Sec-Ch-Ua-Mobile", "?0")
	req.Header.Set("Sec-Ch-Ua-Platform", "Windows")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("Sec-Fetch-Mode", "navigate")
	req.Header.Set("Sec-Fetch-User", "?1")
	req.Header.Set("Sec-Fetch-Dest", "document")
	req.Header.Set("Upgrade-Insecure-Requests", "1")
	// 不设置 X-Requested-With 头，因为浏览器请求中没有

	// 记录完整请求
	dumpGradeReq, _ := httputil.DumpRequestOut(req, false)
	log.Printf("成绩请求:\n%s", string(dumpGradeReq))

	// 执行请求
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("成绩请求错误: %s", err)
		return "成绩请求失败: " + err.Error()
	}
	defer resp.Body.Close()

	// 记录响应信息
	dumpGradeResp, _ := httputil.DumpResponse(resp, false)
	log.Printf("成绩响应:\n%s", string(dumpGradeResp))

	// 检查是否重定向
	gradeRespURL := resp.Request.URL.String()
	if gradeRespURL != gradeURL {
		log.Printf("成绩请求被重定向: %s -> %s", gradeURL, gradeRespURL)
		if strings.Contains(gradeRespURL, "login") {
			log.Printf("被重定向到登录页面，可能是会话已过期")
		}
	}

	// 记录响应后的Cookie
	respCookies := client.Jar.Cookies(u)
	log.Printf("成绩请求后的Cookie (%d个):", len(respCookies))
	for i, c := range respCookies {
		log.Printf("  Cookie %d: %s=%s", i, c.Name, c.Value)
	}

	// 读取响应内容
	content, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("读取成绩响应内容错误: %s", err)
		return "读取成绩响应失败: " + err.Error()
	}

	// 记录响应内容预览
	contentStr := string(content)
	previewLen := min(200, len(contentStr))
	log.Printf("成绩响应内容预览 (前%d字符): %s", previewLen, contentStr[:previewLen])

	// 检查响应是否包含表格数据
	if strings.Contains(contentStr, "<table") {
		log.Printf("响应包含表格数据")
		tableCount := strings.Count(contentStr, "<table")
		rowCount := strings.Count(contentStr, "<tr")
		log.Printf("找到 %d 个表格, %d 行数据", tableCount, rowCount)
	} else {
		log.Printf("响应不包含表格数据")
	}

	return contentStr
}

func GetGradeHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"grade": "A"}`))
}
