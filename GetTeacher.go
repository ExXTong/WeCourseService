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

type TeacherStruct struct {
	CourseID      string
	CourseName    string
	CourseCredit  string
	CourseTeacher string
}

func GetTeacher(cookieStr string) string {
	// 配置读取
	conf := ReadConfig()
	var teachers []TeacherStruct
	var myTeacher TeacherStruct
	var teacherResult TeacherResult

	teacherResult.Type = "teacher"

	// 从字符串解析 cookie
	cookies := parseCookieString(cookieStr)
	if len(cookies) == 0 {
		return getErrorResponse("teacher", "Cookie无效")
	}

	// 创建 HTTP 客户端
	var client http.Client

	// 设置 cookie jar
	jar, err := cookiejar.New(nil)
	if err != nil {
		fmt.Println("ERROR_0: ", err.Error())
		return getErrorResponse("teacher", "创建Cookie容器失败")
	}

	// 将解析的cookies添加到jar中
	u, _ := url.Parse(conf.MangerURL)
	jar.SetCookies(u, cookies)
	client.Jar = jar

	// 直接请求课表页面
	req, err := http.NewRequest(http.MethodGet, conf.MangerURL+"eams/courseTableForStd.action", nil)
	if err != nil {
		fmt.Println("ERROR_9: ", err.Error())
		return getErrorResponse("teacher", "创建请求失败")
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64; rv:66.0) Gecko/20100101 Firefox/66.0")

	resp3, err := client.Do(req)
	if err != nil {
		fmt.Println("ERROR_10: ", err.Error())
		return getErrorResponse("teacher", "网络请求失败")
	}
	defer resp3.Body.Close()

	// 检查是否重定向到登录页面(表示session已过期)
	if strings.Contains(resp3.Request.URL.Path, "login.action") {
		return getErrorResponse("teacher", "登录已过期，请重新登录")
	}

	content, err := io.ReadAll(resp3.Body)
	if err != nil {
		fmt.Println("ERROR_11: ", err.Error())
		return getErrorResponse("teacher", "读取响应失败")
	}

	temp := string(content)
	if !strings.Contains(temp, "bg.form.addInput(form,\"ids\",\"") {
		fmt.Println("ERROR_12: GET ids Failed")
		return getErrorResponse("teacher", "获取课程ID失败")
	}

	temp = temp[strings.Index(temp, "bg.form.addInput(form,\"ids\",\"")+29 : strings.Index(temp, "bg.form.addInput(form,\"ids\",\"")+50]
	ids := temp[:strings.Index(temp, "\");")]

	formValues := make(url.Values)
	formValues.Set("ignoreHead", "1")
	formValues.Set("showPrintAndExport", "1")
	formValues.Set("setting.kind", "std")
	formValues.Set("startWeek", "")
	formValues.Set("semester.id", "30")
	formValues.Set("ids", ids)

	req, err = http.NewRequest(http.MethodPost, conf.MangerURL+"eams/courseTableForStd!courseTable.action", strings.NewReader(formValues.Encode()))
	if err != nil {
		fmt.Println("ERROR_13: ", err.Error())
		return getErrorResponse("teacher", "创建请求失败")
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64; rv:66.0) Gecko/20100101 Firefox/66.0")
	resp4, err := client.Do(req)
	if err != nil {
		fmt.Println("ERROR_14: ", err.Error())
		return getErrorResponse("teacher", "网络请求失败")
	}
	defer resp4.Body.Close()

	content, err = io.ReadAll(resp4.Body)
	if err != nil {
		fmt.Println("ERROR_15: ", err.Error())
		return getErrorResponse("teacher", "读取响应失败")
	}

	temp = string(content)
	if !strings.Contains(temp, "课表格式说明") {
		fmt.Println("ERROR_16: Get Courses Failed")
		return getErrorResponse("teacher", "获取课程失败")
	}
	reg3 := regexp.MustCompile(`(?i)<td>(\d)</td>\s*<td>([:alpha:].+)</td>\s*<td>(.+)</td>\s*<td>((\d)|(\d\.\d))</td>\s*<td>\s*<a href=.*\s.*\s.*\s.*>.*</a>\s*</td>\s*<td>(.*)</td>`)
	reg4 := regexp.MustCompile(`(?i)<td>([^>]*)</td>`)
	reg5 := regexp.MustCompile(`(?i)>([^>]*)</a>`)
	teanchersStr := reg3.FindAllStringSubmatch(temp, -1)
	for _, teacherStr := range teanchersStr {
		teacher := reg4.FindAllStringSubmatch(teacherStr[0], -1)
		courseid := reg5.FindAllStringSubmatch(teacherStr[0], -1)
		myTeacher.CourseID = courseid[0][1]
		myTeacher.CourseName = teacher[2][1]
		myTeacher.CourseCredit = teacher[3][1]
		myTeacher.CourseTeacher = teacher[4][1]
		teachers = append(teachers, myTeacher)
	}
	req, err = http.NewRequest(http.MethodGet, conf.MangerURL+"eams/logout.action", nil)
	if err != nil {
		fmt.Println("ERROR_17: ", err.Error())
		return getErrorResponse("teacher", "创建请求失败")
	}

	resp5, err := client.Do(req)
	if err != nil {
		fmt.Println("ERROR_18: ", err.Error())
		return getErrorResponse("teacher", "网络请求失败")
	}
	defer resp5.Body.Close()
	teacherResult.Data = teachers
	js, err := json.MarshalIndent(teacherResult, "", "\t")
	if err != nil {
		fmt.Println("JSON序列化失败:", err.Error())
		return getErrorResponse("teacher", "生成JSON失败")
	}
	return B2S(js)
}
