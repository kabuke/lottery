# 解析結果 (Analysis.md)

## 專案概觀 (Project Overview)
這是一個基於 Go 語言與 Gin 框架開發的網頁版抽獎應用程式。該程式設計為多租戶 (Multi-tenant) 模式，允許不同使用者 (以 Tenant ID 區分) 擁有獨立的抽獎活動階段 (Session)。主要功能包括獎項管理、參與者管理、抽獎執行、即時結果顯示以及 CSV 報表匯出。

根據 `GEMINI.md` 的需求與 `main.go`, `lottery_service.go` 的實作，此專案完全符合所列出的功能需求，並且採用全記憶體 (In-Memory) 的資料儲存方式，不依賴外部資料庫。

## 技術堆疊 (Data Stack)
- **程式語言:** Go 1.24.1
- **網頁框架:** Gin (github.com/gin-gonic/gin)
- **模板引擎:** Go `html/template` (搭配 `embed` 套件進行靜態資源與模板打包)
- **資料儲存:** In-Memory (記憶體儲存)，無資料庫。
- **依賴管理:** Go Modules (`go.mod`)

## 專案結構 (Project Structure)
```
.
├── cmd/
│   ├── main.go               # 程式進入點，負責初始化服務、路由與啟動 Server
│   ├── templates/            # HTML 頁面模板
│   └── assets/               # 靜態資源 (CSS, JS, 圖片等)
├── internal/
│   ├── handlers/             # HTTP 請求處理 (Controllers)
│   ├── models/               # 資料結構定義 (Prize, Participant, Result)
│   └── services/             # 核心商業邏輯 (抽獎演算法、Session 管理)
├── go.mod                    # 專案依賴定義
├── GEMINI.md                 # 專案需求文件
└── README.md                 # 使用說明書
```

## 功能與實作細節 (Functionality & Implementation)

### 1. 資料模型 (Models)
- **獎項 (Prize):** 包含名稱 (`Name`)、獎品內容 (`Item`)、數量 (`Quantity`) 以及是否從全體抽取 (`DrawFromAll`)。
- **參與者 (Participant):** 包含編號 (`ID`) 與姓名 (`Name`)。
- **抽獎結果 (LotteryResult):** 記錄具體的得獎資訊 (獎項、獎品、得獎者)。

### 2. 核心邏輯 (Lottery Logic) - `internal/services/lottery_service.go`
- **Session 管理:**
    - 使用 `map[string]*LotterySession` 儲存不同 Tenant 的狀態。
    - 實作了 `CleanUpInactiveSessions`，會定期清除閒置超過 1 小時的 Session。
- **抽獎規則:**
    - **一般獎項 (`DrawFromAll = false`):** 僅限「尚未中過任何獎」的參與者參加 (`len(wins) == 0`)。
    - **加碼/特殊獎項 (`DrawFromAll = true`):** 全體參與者皆可參加，但「已中過該特定獎項」者除外 (`!wins[prizeName]`)。這意味著一個人可以多次中獎，只要是不同的「全體抽取」獎項。
- **隨機性:** 使用 `math/rand` 進行隨機抽取。

### 3. API 與路由 (API & Routing)
- **Public Routes:** 公開頁面 (如首頁、登入頁)。
- **Tenant Routes:** 需要 Session/Cookie 驗證的路由 (如設定頁、抽獎 API、報表下載)。
- **Middleware:** `TenantMiddleware` 用於識別與隔離不同使用者的操作。

## 待優化或注意事項 (Observations)
1. **資料持久性:** 目前程式重啟後所有資料會消失 (因記憶體儲存)。這符合 `README` 中「程式不會存儲任何資料」的聲明。
2. **併發處理:** `LotteryService` 使用 `sync.RWMutex` 保護 Shared State，確保多使用者同時操作時的 Thread Safety。
3. **隨機亂數:** 使用全域的 `math/rand`，在 Go 1.20+ 若未 Seed 可能會有固定序列問題，但在較新版本已自動 Seed。專案使用 Go 1.24.1，應無此問題。

## 結論
此專案結構清晰，職責分離 (Handlers, Services, Models)，且完整實作了需求文件中的商業邏輯。
