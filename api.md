# WeCourseService API 文档

## 概述

WeCourseService 是一个基于 WebSocket 的服务，用于获取学校教务系统的各种信息，包括课程表、成绩、教师信息、个人信息和照片等。本服务针对只能创建单个 WebSocket 连接的平台（如微信小程序和uni-app）进行了优化设计。

当前版本：202502281030-SecurityEnhanced

## 连接信息

- WebSocket 端点：`ws://[服务器IP]:[端口]/`
- 端口在配置文件中设置（默认为config.json中的SocketPort值）

## 认证方式

所有请求都需要通过提供有效的教务系统 Cookie 进行认证。Cookie 在初次登录时获取，之后的所有请求都将使用这个 Cookie 进行身份验证。

> 注意：出于安全考虑，本版本不再支持直接发送用户名和密码的认证方式。

## 请求格式

所有 WebSocket 请求使用 JSON 格式，基本结构如下：

```json
{
    "Type": "请求类型",
    "Cookie": "教务系统Cookie",
    "Week": 周数（可选，仅特定请求需要）
}
```

## API 端点

### 1. 获取课程信息

**请求：**
```json
{
    "Type": "allcourse",
    "Cookie": "你的教务系统Cookie"
}
```

**响应：**
```json
{
    "Type": "allcourse",
    "Data": [
        {
            "CourseID": "课程ID",
            "CourseName": "课程名称",
            "RoomID": "教室ID",
            "RoomName": "教室名称",
            "Weeks": "课程周数",
            "CourseTimes": [
                {
                    "DayOfTheWeek": 星期几(0-6),
                    "TimeOfTheDay": 第几节课(0-n)
                }
            ]
        }
    ]
}
```

### 2. 获取个人账户信息

**请求：**
```json
{
    "Type": "account",
    "Cookie": "你的教务系统Cookie"
}
```

**响应：**
```json
{
    "Type": "account",
    "Data": {
        "FullName": "姓名",
        "EnglishName": "英文名",
        "Sex": "性别",
        "StartTime": "入学时间",
        "EndTime": "毕业时间",
        "SchoolYear": "学年",
        "Type": "学生类型",
        "System": "学院",
        "Specialty": "专业",
        "Class": "班级"
    }
}
```

### 3. 获取当前教学周

**请求：**
```json
{
    "Type": "week",
    "Cookie": "你的教务系统Cookie"
}
```

**响应：**
```json
{
    "Type": "week",
    "Data": "当前周数"
}
```

### 4. 获取教师信息

**请求：**
```json
{
    "Type": "teacher",
    "Cookie": "你的教务系统Cookie"
}
```

**响应：**
```json
{
    "Type": "teacher",
    "Data": [
        {
            "CourseID": "课程ID",
            "CourseName": "课程名称",
            "CourseCredit": "学分",
            "CourseTeacher": "授课教师"
        }
    ]
}
```

### 5. 获取用户照片

**请求：**
```json
{
    "Type": "photo",
    "Cookie": "你的教务系统Cookie"
}
```

**响应：**
```json
{
    "Type": "photo",
    "Data": "base64编码的图片数据，格式为data:image/jpg;base64,..."
}
```

### 6. 获取成绩信息

**请求：**
```json
{
    "Type": "grade",
    "Cookie": "你的教务系统Cookie"
}
```

**响应：**
```json
{
    "Type": "grade",
    "Data": [
        {
            "CourseID": "课程ID",
            "CourseName": "课程名称",
            "CourseTerm": "学期",
            "CourseCredit": "学分",
            "CourseGrade": "成绩",
            "GradePoint": "绩点"
        }
    ]
}
```

## 错误处理

当请求失败时，API 会返回一个包含错误信息的 JSON 对象：

```json
{
    "Type": "请求的类型",
    "Data": "错误信息"
}
```

常见错误：
- "Cookie无效" - 提供的Cookie不正确或已过期
- "登录已过期，请重新登录" - 会话已过期，需要重新获取Cookie
- "网络连接错误" - 连接到教务系统时出现网络问题
- "解析数据失败" - 服务器返回的数据格式不正确

## 缓存机制

为了提高性能并减少对教务系统的请求频率，某些请求（如课程信息）会被缓存一小时。如果需要获取最新数据，请确保Cookie已更新。

## 配置文件

服务使用config.json配置文件，包含以下关键设置：
- SchoolName: 学校名称
- MangerType: 教务系统类型 (目前支持"supwisdom")
- MangerURL: 教务系统URL
- SocketPort: WebSocket服务端口
- CalendarFirst: 学期开始日期（用于计算当前教学周）