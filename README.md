# Go 即時聊天室系統

一個展示 Go 語言高併發處理能力的即時聊天室系統，使用 WebSocket 實現即時通訊。

## 技術棧

| 技術 | 版本 | 用途 |
|------|------|------|
| Go | 1.21+ | 程式語言 |
| Gin | 1.9+ | Web 框架 |
| gorilla/websocket | 1.5+ | WebSocket |
| PostgreSQL | 15+ | 資料庫 |
| sqlx | 1.3+ | SQL 工具 |
| golang-migrate | 4.17+ | Migration |
| Redis | 7+ | 快取/Pub-Sub |
| JWT | 5.2+ | 認證 |
| Swagger | 1.16+ | API 文件 |
| Viper | 1.18+ | 設定管理 |
| Zap | 1.26+ | 日誌 |
| Docker | - | 容器化 |

## 專案結構

```
go-demo/
├── cmd/server/main.go          # 應用進入點
├── internal/
│   ├── config/                 # 設定管理
│   ├── model/                  # 資料模型
│   ├── repository/             # 資料存取層
│   ├── service/                # 業務邏輯層
│   ├── handler/                # HTTP 處理器
│   ├── ws/                     # WebSocket 核心
│   ├── middleware/             # 中間件
│   ├── dto/                    # 資料傳輸物件
│   └── pkg/                    # 工具包
├── migrations/                 # 資料庫遷移
├── docs/                       # Swagger 文件
├── scripts/                    # 腳本
├── .github/workflows/ci.yml    # CI 配置
├── Dockerfile                  # Docker 配置
├── docker-compose.yml          # Docker Compose
├── Makefile                    # 常用指令
└── README.md
```

## 功能特點

### 核心功能
- 用戶註冊、登入、登出
- JWT Token 認證
- 公開/私人聊天室
- 即時訊息 (WebSocket)
- 私人訊息 (Direct Message)
- 好友系統
- 用戶封鎖
- 訊息搜尋

### 技術亮點
- **高併發處理**: 使用 goroutine 和 channel 實現
- **WebSocket Hub**: 支援多房間訂閱和用戶狀態追蹤
- **水平擴展**: 透過 Redis Pub/Sub 支援多實例部署
- **乾淨架構**: Repository -> Service -> Handler 分層
- **完整測試**: 單元測試覆蓋率 80%+

## 快速開始

### 環境需求
- Go 1.21+
- Docker & Docker Compose
- Make (可選)

### 使用 Docker Compose

```bash
# 啟動所有服務
docker-compose up -d

# 執行資料庫遷移
docker-compose run --rm migrate

# 查看日誌
docker-compose logs -f app
```

### 本地開發

```bash
# 安裝依賴
go mod download

# 啟動資料庫和 Redis
docker-compose up -d postgres redis

# 執行資料庫遷移
make migrate-up

# 執行 seed 資料
make seed

# 啟動服務
make run
```

## API 文件

啟動服務後，訪問 Swagger UI：
```
http://localhost:8080/swagger/index.html
```

### 主要 API 端點

#### 認證
- `POST /api/v1/auth/register` - 用戶註冊
- `POST /api/v1/auth/login` - 用戶登入
- `POST /api/v1/auth/logout` - 用戶登出
- `POST /api/v1/auth/refresh` - 刷新 Token
- `GET /api/v1/auth/me` - 獲取當前用戶

#### 用戶
- `GET /api/v1/users/search` - 搜尋用戶
- `GET /api/v1/users/:id` - 獲取用戶資料
- `POST /api/v1/users/:id/friend-request` - 發送好友請求
- `GET /api/v1/users/friends` - 好友列表

#### 聊天室
- `GET /api/v1/rooms` - 公開聊天室列表
- `POST /api/v1/rooms` - 創建聊天室
- `POST /api/v1/rooms/:id/join` - 加入聊天室
- `GET /api/v1/rooms/:id/messages` - 獲取訊息

#### 私訊
- `GET /api/v1/dm` - 對話列表
- `POST /api/v1/dm/:user_id` - 發送私訊
- `GET /api/v1/dm/:user_id` - 獲取對話

## WebSocket 使用

### 連線

```javascript
const ws = new WebSocket('ws://localhost:8080/ws?token=YOUR_JWT_TOKEN');
```

### 訊息類型

#### 客戶端 -> 伺服器

```json
// 加入聊天室
{"type": "join_room", "payload": {"room_id": "xxx"}}

// 發送訊息
{"type": "send_message", "payload": {"room_id": "xxx", "content": "Hello!"}}

// 發送私訊
{"type": "send_dm", "payload": {"receiver_id": "xxx", "content": "Hi!"}}

// 輸入中
{"type": "typing", "payload": {"room_id": "xxx"}}
```

#### 伺服器 -> 客戶端

```json
// 新訊息
{"type": "new_message", "payload": {...}}

// 用戶上線
{"type": "user_online", "payload": {"user_id": "xxx", "username": "alice"}}

// 私訊
{"type": "new_dm", "payload": {...}}
```

## 測試

```bash
# 執行所有測試
make test

# 執行測試並生成覆蓋率報告
make test-coverage

# 查看覆蓋率報告
go tool cover -html=coverage.out
```

## 開發指令

```bash
make help          # 顯示所有可用指令
make build         # 編譯專案
make run           # 執行專案
make test          # 執行測試
make lint          # 程式碼檢查
make swagger       # 生成 Swagger 文件
make migrate-up    # 執行資料庫遷移
make migrate-down  # 回滾資料庫遷移
make docker-up     # 啟動 Docker 容器
make docker-down   # 停止 Docker 容器
```

## 測試帳號

執行 `make seed` 後可使用以下測試帳號：

| 帳號 | 密碼 |
|------|------|
| alice | password123 |
| bob | password123 |
| charlie | password123 |
| diana | password123 |
| evan | password123 |

## 環境變數

複製 `.env.example` 為 `.env` 並根據需要修改：

```bash
cp .env.example .env
```

| 變數 | 說明 | 預設值 |
|------|------|--------|
| SERVER_PORT | 伺服器埠號 | 8080 |
| SERVER_MODE | 運行模式 (debug/release) | debug |
| DB_HOST | PostgreSQL 主機 | localhost |
| DB_PORT | PostgreSQL 埠號 | 5432 |
| DB_USER | PostgreSQL 用戶 | postgres |
| DB_PASSWORD | PostgreSQL 密碼 | postgres |
| DB_NAME | 資料庫名稱 | chat |
| REDIS_HOST | Redis 主機 | localhost |
| REDIS_PORT | Redis 埠號 | 6379 |
| JWT_SECRET | JWT 密鑰 | (請更改) |

## 部署

### Docker 部署

```bash
# 構建映像
docker build -t chat-server .

# 執行容器
docker run -d \
  -p 8080:8080 \
  -e DB_HOST=your-db-host \
  -e REDIS_HOST=your-redis-host \
  -e JWT_SECRET=your-secret \
  chat-server
```

### 生產環境建議

1. 使用強密鑰：更改 JWT_SECRET 為安全的隨機字串
2. 啟用 HTTPS：使用反向代理 (Nginx/Traefik)
3. 設定 CORS：限制允許的來源
4. 啟用限流：防止濫用
5. 監控日誌：收集並分析日誌

## 授權

MIT License
