package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// ---------- 数据模型 ----------

// Todo 是我们管理的资源，类似 OpenSandbox 里的 Sandbox
type Todo struct {
	ID        int       `json:"id"`
	Title     string    `json:"title"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"createdAt"`
}

// ---------- 请求和响应 ----------

type CreateTodoRequest struct {
	Title string `json:"title"`
}

type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// ---------- 内存存储 ----------

var (
	todos  = make(map[int]*Todo)
	nextID = 1
)

// ---------- 工具函数 ----------

func writeJSON(w http.ResponseWriter, code int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(data)
}

// ---------- Handler ----------

func handleCreateTodo(w http.ResponseWriter, r *http.Request) {
	// 第一步：只允许 POST 方法
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, ErrorResponse{
			Code:    "METHOD_NOT_ALLOWED",
			Message: "use POST",
		})
		return
	}

	// 第二步：读取请求体，把 JSON 解析成结构体
	var req CreateTodoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{
			Code:    "INVALID_JSON",
			Message: "request body is not valid JSON",
		})
		return
	}

	// 第三步：验证参数
	if req.Title == "" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{
			Code:    "MISSING_TITLE",
			Message: "title is required",
		})
		return
	}

	// 第四步：创建 Todo，存进 map
	todo := &Todo{
		ID:        nextID,
		Title:     req.Title,
		Status:    "pending",
		CreatedAt: time.Now(),
	}
	todos[nextID] = todo
	nextID++

	// 第五步：返回创建好的 Todo（201 = Created）
	writeJSON(w, http.StatusCreated, todo)
}

// ---------- 启动 ----------

func main() {
	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	http.HandleFunc("/todos", handleCreateTodo)

	fmt.Println("server started on :9090")
	http.ListenAndServe(":9090", nil)
}
