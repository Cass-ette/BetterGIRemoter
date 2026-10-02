# BetterGI Remoter

自托管 BetterGI 远程控制后端，可通过网页控制远程电脑上的 BetterGI。

## 功能特性

- 🎮 远程启动/停止 BetterGI 任务
- 📊 实时查看任务状态
- 🔐 Token 认证保护
- 🌐 网页控制面板
- 🚀 单文件部署，无需外部依赖

## 快速开始

### 1. 配置 BetterGI

在 BetterGI 中启用远程控制：
1. 打开 BetterGI 设置
2. 进入"远程控制"页面
3. 启用远程控制 API
4. 生成并复制 API Token
5. 重启 BetterGI

### 2. 启动后端

#### 使用 Go 运行

```bash
# 安装依赖
go mod download

# 设置环境变量
export BETTERGI_URL="http://localhost:8080"     # BetterGI API 地址
export BETTERGI_TOKEN="你的API Token"           # BetterGI API Token
export ADMIN_TOKEN="你的管理密码"                # 可选：保护管理端点
export PORT="8000"                               # 可选：监听端口（默认8080）

# 运行
go run main.go
```

#### 使用编译后的二进制

```bash
# 编译
go build -o bettergi-remoter

# Windows
set BETTERGI_URL=http://localhost:8080
set BETTERGI_TOKEN=你的API Token
bettergi-remoter.exe

# Linux/Mac
export BETTERGI_URL=http://localhost:8080
export BETTERGI_TOKEN=你的API Token
./bettergi-remoter
```

### 3. 访问控制面板

打开浏览器访问：`http://localhost:8000`

## 远程访问

### SSH 隧道（推荐）

在你的服务器上运行后端，然后从本地电脑建立 SSH 隧道：

```bash
# 在运行 BetterGI 的电脑上
ssh -R 8080:localhost:8080 user@your-server

# 在服务器上启动 BetterGI Remoter
export BETTERGI_URL="http://localhost:8080"
export BETTERGI_TOKEN="你的Token"
./bettergi-remoter

# 从手机访问
http://your-server:8000
```

### frp 内网穿透

使用 frp 将 BetterGI API 端口暴露到公网服务器，配置与 SSH 隧道类似。

## API 端点

### 基础控制
```bash
POST /admin/start                    # 启动任务
POST /admin/stop                     # 停止任务
GET  /admin/status                   # 获取状态
GET  /admin/tasks                    # 任务列表
```

### 脚本管理
```bash
GET  /admin/scripts                  # 获取所有脚本组
POST /admin/script/{id}/start        # 启动指定脚本
```

### 一条龙流程
```bash
GET  /admin/onedragon/configs        # 获取一条龙配置列表
POST /admin/onedragon/execute/{id}   # 执行一条龙配置
```

### 截图功能
```bash
GET  /admin/screenshot               # 获取游戏截图（Base64 PNG）
```

所有端点需要在 Header 中添加：
```
Authorization: Bearer <ADMIN_TOKEN>
X-API-Token: <BETTERGI_TOKEN>
```

## 环境变量

| 变量 | 说明 | 默认值 | 必填 |
|------|------|--------|------|
| `BETTERGI_URL` | BetterGI API 地址 | http://localhost:8080 | 否 |
| `BETTERGI_TOKEN` | BetterGI API Token | - | 是 |
| `ADMIN_TOKEN` | 管理端点访问密码 | - | 否 |
| `PORT` | 监听端口 | 8080 | 否 |

## 架构

```
┌─────────┐      HTTP       ┌──────────────┐      HTTP API      ┌──────────┐
│  手机   │ ────────────> │ BetterGI     │ ──────────────> │ BetterGI │
│ 浏览器  │               │ Remoter      │   (X-API-Token) │ (PC)     │
└─────────┘               └──────────────┘                 └──────────┘
                                 │
                          SSH隧道/frp内网穿透
                                 │
                          ┌──────▼──────┐
                          │  公网服务器  │
                          └─────────────┘
```

## 技术栈

- **后端**: Go + Gin
- **前端**: 原生 HTML/CSS/JS
- **部署**: 单文件二进制

## 复用自 ArknightsMaaRemoter

本项目架构复用自 [ArknightsMaaRemoter](https://github.com/Cass-ette/ArknightsMaaRemoter-)，适配 BetterGI HTTP API。

## License

MIT
