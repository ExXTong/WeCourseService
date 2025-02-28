package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// 课程持续时间，周几第几节
type CourseTime struct {
	DayOfTheWeek int
	TimeOfTheDay int
}

// 课程信息
type Course struct {
	CourseID    string
	CourseName  string
	RoomID      string
	RoomName    string
	Weeks       string
	CourseTimes []CourseTime
}

var USERNAME, PASSWORD string
var myCourses []Course
var teachers []TeacherStruct
var myTeacher TeacherStruct
var myAllCourseResult CourseResult

// 实现简单的内存缓存，替代 github.com/patrickmn/go-cache
type SimpleCache struct {
	items map[string]cacheItem
}

type cacheItem struct {
	value      string
	expiration int64
}

func NewSimpleCache() *SimpleCache {
	return &SimpleCache{
		items: make(map[string]cacheItem),
	}
}

var c = NewSimpleCache()

func (c *SimpleCache) Set(key string, value string, duration time.Duration) {
	c.items[key] = cacheItem{
		value:      value,
		expiration: time.Now().Add(duration).UnixNano(),
	}
}

func (c *SimpleCache) Get(key string) (string, bool) {
	item, found := c.items[key]
	if !found {
		return "", false
	}
	if time.Now().UnixNano() > item.expiration {
		delete(c.items, key)
		return "", false
	}
	return item.value, true
}

// 删除重复定义的B2S函数 - 使用utils.go中的定义

func GetTeacherObj() []TeacherStruct {
	return teachers
}

func GetCourse(cookieStr string) string {
	// 使用 cookie 作为缓存键
	value, found := c.Get(cookieStr)
	if found && value != "" {
		//fmt.Print("Using Cache")
		return value
	}

	// 配置读取
	conf := ReadConfig()
	myCourses = nil
	teachers = nil

	myAllCourseResult.Type = "allcourse"

	// 解析 cookie 字符串
	cookies := parseCookieString(cookieStr)
	if len(cookies) == 0 {
		return getErrorResponse("allcourse", "Cookie无效")
	}

	// 创建 HTTP 客户端
	var client http.Client

	// 设置 cookie jar
	jar, err := cookiejar.New(nil)
	if err != nil {
		fmt.Println("ERROR_0: ", err.Error())
		return getErrorResponse("allcourse", "创建Cookie容器失败")
	}

	// 将解析的cookies添加到jar中
	u, _ := url.Parse(conf.MangerURL)
	jar.SetCookies(u, cookies)
	client.Jar = jar

	// 直接请求课表页面
	req, err := http.NewRequest(http.MethodGet, conf.MangerURL+"eams/courseTableForStd.action", nil)
	if err != nil {
		fmt.Println("ERROR_9: ", err.Error())
		return getErrorResponse("allcourse", "创建请求失败")
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64; rv:66.0) Gecko/20100101 Firefox/66.0")

	resp3, err := client.Do(req)
	if err != nil {
		fmt.Println("ERROR_10: ", err.Error())
		return getErrorResponse("allcourse", "网络请求失败")
	}
	defer resp3.Body.Close()

	// 检查是否重定向到登录页面(表示session已过期)
	if strings.Contains(resp3.Request.URL.Path, "login.action") {
		return getErrorResponse("allcourse", "登录已过期，请重新登录")
	}

	content, err := io.ReadAll(resp3.Body)
	if err != nil {
		fmt.Println("ERROR_11: ", err.Error())
		return getErrorResponse("allcourse", "读取响应失败")
	}

	temp := string(content)
	if !strings.Contains(temp, "bg.form.addInput(form,\"ids\",\"") {
		fmt.Println("ERROR_12: GET ids Failed")
		return getErrorResponse("allcourse", "获取课程ID失败")
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
		return getErrorResponse("allcourse", "创建课表请求失败")
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64; rv:66.0) Gecko/20100101 Firefox/66.0")

	resp4, err := client.Do(req)
	if err != nil {
		fmt.Println("ERROR_14: ", err.Error())
		return getErrorResponse("allcourse", "获取课表失败")
	}
	defer resp4.Body.Close()

	content, err = io.ReadAll(resp4.Body)
	if err != nil {
		fmt.Println("ERROR_15: ", err.Error())
		return getErrorResponse("allcourse", "读取课表内容失败")
	}

	temp = string(content)
	if !strings.Contains(temp, "课表格式说明") {
		fmt.Println("ERROR_16: Get Courses Failed")
		return getErrorResponse("allcourse", "解析课表格式失败")
	}

	// 以下是原有的课程数据解析逻辑，保持不变
	reg1 := regexp.MustCompile(`TaskActivity\(actTeacherId.join\(','\),actTeacherName.join\(','\),"(.*)","(.*)\(.*\)","(.*)","(.*)","(.*)",null,null,assistantName,"",""\);((?:\s*index =\d+\*unitCount\+\d+;\s*.*\s)+)`)
	reg2 := regexp.MustCompile(`\s*index =(\d+)\*unitCount\+(\d+);\s*`)
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

	coursesStr := reg1.FindAllStringSubmatch(temp, -1)
	for _, courseStr := range coursesStr {
		var course Course
		course.CourseID = courseStr[1]
		course.CourseName = courseStr[2]
		course.RoomID = courseStr[3]
		course.RoomName = courseStr[4]
		course.Weeks = courseStr[5]
		for _, indexStr := range strings.Split(courseStr[6], "table0.activities[index][table0.activities[index].length]=activity;") {
			if !strings.Contains(indexStr, "unitCount") {
				continue
			}
			var courseTime CourseTime
			courseTime.DayOfTheWeek, _ = strconv.Atoi(reg2.FindStringSubmatch(indexStr)[1])
			courseTime.TimeOfTheDay, _ = strconv.Atoi(reg2.FindStringSubmatch(indexStr)[2])
			course.CourseTimes = append(course.CourseTimes, courseTime)
		}
		myCourses = append(myCourses, course)
	}
	req, err = http.NewRequest(http.MethodGet, conf.MangerURL+"eams/logout.action", nil)
	if err != nil {
		fmt.Println("ERROR_17: ", err.Error())
		return getErrorResponse("allcourse", "登出失败")
	}

	resp5, err := client.Do(req)
	if err != nil {
		fmt.Println("ERROR_18: ", err.Error())
		return getErrorResponse("allcourse", "登出请求失败")
	}
	defer resp5.Body.Close()
	myAllCourseResult.Data = myCourses
	js, err := json.MarshalIndent(myAllCourseResult, "", "\t")
	if err != nil {
		// 根据您的应用程序需求处理错误
		fmt.Println("JSON序列化失败:", err.Error())
		return getErrorResponse("courses", "生成JSON失败")
	}
	cachestr := B2S(js)
	c.Set(cookieStr, cachestr, time.Hour) // 缓存1小时
	value_check, found_check := c.Get(cookieStr)
	if found_check {
		//fmt.Print("Using Cache")
		if value_check == "" {
			c.Set(cookieStr, cachestr, time.Hour) // 缓存1小时
		}
	}
	return cachestr
}
