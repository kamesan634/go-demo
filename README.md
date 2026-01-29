# 龜三的即時聊天室 Demo - Go WebSocket 專案

![CI](https://github.com/kamesan634/go-demo/actions/workflows/ci.yml/badge.svg)

基於 Go 1.21 + Gin + WebSocket 的即時聊天室系統，展示高併發處理能力。

## 技能樹 請點以下技能

| 技能 | 版本 | 說明 |
|------|------|------|
| Go | 1.21 | 程式語言 |
| Gin | 1.9 | Web 框架 |
| gorilla/websocket | 1.5 | WebSocket 實作 |
| PostgreSQL | 15 | 資料庫 |
| Redis | 7 | 快取 / Pub-Sub |
| sqlx | 1.3 | SQL 工具（非 ORM） |
| JWT | 5.2 | Token 認證 |
| Zap | 1.26 | 結構化日誌 |
| Viper | 1.18 | 設定管理 |
| Swagger | 1.16 | API 文件 |
| Docker | - | 容器化佈署 |

## 功能模組

- **auth** - 認證管理（註冊、登入、JWT Token）
- **users** - 用戶管理（個人資料、好友、封鎖）
- **rooms** - 聊天室管理（公開/私人房間、成員管理）
- **messages** - 訊息管理（發送、歷史記錄、搜尋）
- **dm** - 私訊管理（一對一私人訊息）
- **ws** - WebSocket 核心（即時推送、狀態同步）

## 快速開始

### 環境需求

- Docker & Docker Compose
- 或 Go 1.21 + PostgreSQL 15 + Redis 7

### 使用 Docker 佈署（推薦）

```bash
# 啟動所有服務
docker-compose up -d

# 查看服務狀態
docker-compose ps

# 查看日誌
docker-compose logs -f app

# 停止服務
docker-compose down
```

### 本地開發

```bash
# 啟動資料庫和 Redis
docker-compose up -d postgres redis

# 執行資料庫遷移
make migrate-up

# 執行 seed 資料
make seed

# 啟動服務
make run
```

## Port

| 服務 | Port | 說明 |
|------|------|------|
| Go Server | 8080 | API / WebSocket |
| PostgreSQL | 5432 | 資料庫 |
| Redis | 6379 | 快取 / Pub-Sub |

## API 端點

啟動服務後，訪問 Swagger UI：http://localhost:8080/swagger/index.html

### 主要端點

| 端點 | 方法 | 說明 |
|------|------|------|
| /api/v1/auth/register | POST | 用戶註冊 |
| /api/v1/auth/login | POST | 用戶登入 |
| /api/v1/auth/logout | POST | 用戶登出 |
| /api/v1/auth/me | GET | 取得當前用戶 |
| /api/v1/rooms | GET | 聊天室列表 |
| /api/v1/rooms | POST | 建立聊天室 |
| /api/v1/rooms/:id/join | POST | 加入聊天室 |
| /api/v1/rooms/:id/messages | GET | 取得訊息歷史 |
| /api/v1/dm | GET | 私訊對話列表 |
| /api/v1/dm/:user_id | POST | 發送私訊 |
| /api/v1/users/search | GET | 搜尋用戶 |
| /api/v1/users/friends | GET | 好友列表 |
| /ws | GET | WebSocket 連線 |

## 測試資訊

### 測試帳號

所有帳號的密碼都是：`password123`

| 帳號 | 說明 |
|------|------|
| alice | 測試用戶 1 |
| bob | 測試用戶 2 |
| charlie | 測試用戶 3 |
| diana | 測試用戶 4 |
| evan | 測試用戶 5 |

### 測試資料

執行 `make seed` 後會建立以下種子資料：

| 資料類型 | 數量 | 說明 |
|----------|------|------|
| 用戶 | 5 | alice, bob, charlie, diana, evan |
| 聊天室 | 3 | 公開聊天室 |
| 訊息 | 10 | 測試訊息 |
| 好友關係 | 4 | 用戶間的好友關係 |

## 專案結構

```
go-demo/
├── docker-compose.yml          # Docker Compose 配置
├── Dockerfile                  # Docker 映像配置
├── Makefile                    # 常用指令
├── migrations/                 # 資料庫遷移腳本
├── scripts/                    # 工具腳本
├── cmd/
│   └── server/main.go          # 應用程式進入點
├── internal/
│   ├── config/                 # 設定管理
│   ├── model/                  # 資料模型
│   ├── repository/             # 資料存取層
│   ├── service/                # 業務邏輯層
│   ├── handler/                # HTTP 處理器
│   ├── ws/                     # WebSocket 核心
│   ├── middleware/             # 中間件
│   ├── dto/                    # 資料傳輸物件
│   └── pkg/                    # 工具套件
```

## 資料庫連線

### Docker 環境

- Host: `localhost`
- Port: `5432`
- Database: `chat`
- Username: `postgres`
- Password: `postgres`

```bash
# 使用 psql 連線
psql -h localhost -p 5432 -U postgres -d chat

# 或進入 Docker 容器
docker exec -it chat-postgres psql -U postgres -d chat
```

## 健康檢查

```bash
# 檢查 API 健康狀態
curl http://localhost:8080/health

# 檢查 WebSocket 連線
wscat -c "ws://localhost:8080/ws?token=YOUR_TOKEN"
```

## WebSocket 訊息格式

### 客戶端 -> 伺服器

```json
// 加入聊天室
{"type": "join_room", "payload": {"room_id": "xxx"}}

// 發送訊息
{"type": "send_message", "payload": {"room_id": "xxx", "content": "Hello!"}}

// 發送私訊
{"type": "send_dm", "payload": {"receiver_id": "xxx", "content": "Hi!"}}
```

### 伺服器 -> 客戶端

```json
// 新訊息通知
{"type": "new_message", "payload": {...}}

// 用戶上線通知
{"type": "user_online", "payload": {"user_id": "xxx", "username": "alice"}}

// 新私訊通知
{"type": "new_dm", "payload": {...}}
```

## License

MIT License
