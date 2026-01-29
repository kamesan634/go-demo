# CLAUDE.md - Go 即時聊天室系統

## 專案概述

這是一個使用 Go 語言開發的即時聊天室系統，展示高併發處理能力和 WebSocket 即時通訊。

## 技術棧

- **Go 1.21+**: 主要程式語言
- **Gin**: Web 框架
- **gorilla/websocket**: WebSocket 實現
- **PostgreSQL**: 主要資料庫
- **Redis**: 快取和 Pub/Sub
- **sqlx**: SQL 工具 (非 ORM)
- **JWT**: 認證機制
- **Zap**: 結構化日誌
- **Viper**: 設定管理

## 專案結構

```
go-demo/
├── cmd/server/main.go          # 進入點和路由設定
├── internal/
│   ├── config/config.go        # Viper 設定載入
│   ├── model/                  # 資料模型 (User, Room, Message 等)
│   ├── repository/             # 資料存取層 (sqlx SQL)
│   ├── service/                # 業務邏輯層
│   ├── handler/                # HTTP 處理器
│   ├── ws/                     # WebSocket 核心 (Hub, Client)
│   ├── middleware/             # 認證、CORS、限流、日誌
│   ├── dto/                    # Request/Response DTO
│   └── pkg/                    # 工具包 (password, jwt, validator)
├── migrations/                 # PostgreSQL 遷移檔
└── scripts/seed.go             # 測試資料
```

## 常用開發指令

```bash
# 啟動開發環境
docker-compose up -d postgres redis
make migrate-up
make run

# 執行測試
make test

# 程式碼檢查
make lint

# 生成 Swagger
make swagger
```

## 重要架構決策

### 1. WebSocket Hub 設計
- 使用 channel 避免鎖競爭
- 每個 Client 使用 readPump + writePump 兩個 goroutine
- 支援多房間訂閱和多設備連線
- Redis Pub/Sub 支援水平擴展

### 2. 資料存取
- 使用 sqlx 直接寫 SQL，不使用 ORM
- Repository 模式分離資料存取邏輯
- 使用 sql.NullString 處理可空欄位

### 3. 認證
- JWT Access Token (15分鐘) + Refresh Token (7天)
- bcrypt 密碼雜湊
- 中間件驗證 Token

## API 端點摘要

- `POST /api/v1/auth/register` - 註冊
- `POST /api/v1/auth/login` - 登入
- `GET /api/v1/rooms` - 聊天室列表
- `POST /api/v1/rooms/:id/messages` - 發送訊息
- `GET /ws?token=xxx` - WebSocket 連線

## WebSocket 訊息類型

```go
// 客戶端 -> 伺服器
MessageTypeJoinRoom    = "join_room"
MessageTypeSendMessage = "send_message"
MessageTypeSendDM      = "send_dm"
MessageTypeTyping      = "typing"

// 伺服器 -> 客戶端
MessageTypeNewMessage  = "new_message"
MessageTypeUserOnline  = "user_online"
MessageTypeNewDM       = "new_dm"
```

## 測試帳號

所有帳號密碼: `password123`
- alice, bob, charlie, diana, evan

## 注意事項

1. JWT_SECRET 在生產環境必須更改
2. 資料庫連線池設定在 config/config.go
3. WebSocket 心跳間隔為 54 秒
4. Rate limit: API 100/分鐘, Auth 10/分鐘, Message 60/分鐘
