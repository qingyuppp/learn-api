package main

import (
	"encoding/json" // JSON 编解码
	"fmt"           // 格式化输出
	"net/http"      // HTTP 服务器和客户端
	"os"            // 读取环境变量
	"strconv"       // 字符串和数字互转
	"strings"       // 字符串处理
	"time"          // 时间处理
)

// ---------- 数据模型 ----------

// Todo 是我们管理的核心资源
// 每个字段后面的 `json:"xxx"` 叫 struct tag，控制 JSON 序列化时的字段名
// 比如 Go 里叫 CreatedAt，JSON 输出时变成 createdAt
type Todo struct {
	ID        int       `json:"id"`        // 唯一标识，自增
	Title     string    `json:"title"`     // 标题
	Status    string    `json:"status"`    // 状态：pending / doing / done
	CreatedAt time.Time `json:"createdAt"` // 创建时间
}

// ---------- 请求和响应结构体 ----------

// CreateTodoRequest 定义了创建 Todo 时客户端需要发什么
// 客户端发 {"title": "买菜"}，Decode 后 req.Title == "买菜"
type CreateTodoRequest struct {
	Title string `json:"title"`
}

// UpdateTodoRequest 定义了更新 Todo 时客户端可以发什么
// 客户端发 {"status": "doing"}，表示要把状态改成 doing
type UpdateTodoRequest struct {
	Status string `json:"status"`
}

// ErrorResponse 是统一的错误响应格式
// 所有错误都返回这个结构，客户端可以统一处理
type ErrorResponse struct {
	Code    string `json:"code"`    // 机器可读的错误码，如 "MISSING_TITLE"
	Message string `json:"message"` // 人类可读的错误描述
}

// ---------- 内存存储 ----------

// 用 map 模拟数据库，key 是 Todo ID，value 是 Todo 指针
// 真实项目这里会换成数据库（MySQL、PostgreSQL 等）
var (
	todos  = make(map[int]*Todo) // 存储所有 Todo
	nextID = 1                   // 下一个可用的 ID，每次创建后自增
)

// ---------- 工具函数 ----------

// writeJSON 把任意数据以 JSON 格式写入 HTTP 响应
// 三步必须按顺序：设 Header → 写状态码 → 写 Body
// 因为 HTTP 协议规定了 响应头 在 响应体 前面
func writeJSON(w http.ResponseWriter, code int, data any) {
	w.Header().Set("Content-Type", "application/json") // 告诉客户端返回的是 JSON
	w.WriteHeader(code)                                 // 写 HTTP 状态码（200、400、500...）
	json.NewEncoder(w).Encode(data)                     // 把 data 结构体编码成 JSON 写进响应
}

// ---------- 中间件 ----------

// requireAuth 是一个中间件，检查请求头中的 API Key
// 它接收一个 handler，返回一个新的 handler（包了一层认证检查）
//
// 使用方式：
//   http.HandleFunc("/todos", requireAuth(handleTodos))
//   原来直接注册 handleTodos，现在用 requireAuth 包一层
//
// 调用链：请求进来 → requireAuth 检查 Key → 通过 → 调用原始 handler
//                                         → 不通过 → 返回 401
func requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 从环境变量读取期望的 API Key
		// 这样 Key 不会写死在代码里（安全）
		expected := os.Getenv("API_KEY")

		// 如果没配置 API Key（环境变量为空），跳过认证
		if expected == "" {
			next(w, r)
			return
		}

		// 从请求头中取客户端传来的 Key
		provided := r.Header.Get("X-API-Key")
		if provided == "" {
			writeJSON(w, http.StatusUnauthorized, ErrorResponse{
				Code:    "MISSING_API_KEY",
				Message: "provide API key via X-API-Key header",
			})
			return
		}

		// 比较 Key 是否正确
		if provided != expected {
			writeJSON(w, http.StatusUnauthorized, ErrorResponse{
				Code:    "INVALID_API_KEY",
				Message: "invalid API key",
			})
			return
		}

		// 认证通过，调用原始 handler
		next(w, r)
	}
}

// ---------- Handler（处理函数） ----------

// handleHealthz 处理 GET /healthz，返回服务健康状态
func handleHealthz(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// handleTodos 是 /todos 路径的入口
// 根据 HTTP 方法分发到不同的处理函数：
//   GET  /todos → 列出所有 Todo
//   POST /todos → 创建一个 Todo
//   其他方法    → 返回 405 Method Not Allowed
func handleTodos(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		handleListTodos(w, r)
	case http.MethodPost:
		handleCreateTodo(w, r)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, ErrorResponse{
			Code:    "METHOD_NOT_ALLOWED",
			Message: "use GET or POST",
		})
	}
}

// handleListTodos 处理 GET /todos，返回所有 Todo 的列表
func handleListTodos(w http.ResponseWriter, r *http.Request) {
	// map 不能直接序列化成 JSON 数组，需要转成 slice
	// make([]*Todo, 0, len(todos)) 创建一个空 slice，预分配容量避免扩容
	list := make([]*Todo, 0, len(todos))
	for _, todo := range todos {
		list = append(list, todo)
	}
	// 返回 200 + JSON 数组
	writeJSON(w, http.StatusOK, list)
}

// handleTodoByID 处理 /todos/{id} 路径
// 从 URL 中提取 ID，然后根据方法分发
// 比如请求 GET /todos/3，提取出 id=3，然后调 handleGetTodo
func handleTodoByID(w http.ResponseWriter, r *http.Request) {
	// 从路径中提取 ID
	// r.URL.Path 是 "/todos/3"，去掉前缀 "/todos/" 得到 "3"
	idStr := strings.TrimPrefix(r.URL.Path, "/todos/")

	// 把字符串 "3" 转成数字 3
	// strconv.Atoi = "ASCII to Integer"
	id, err := strconv.Atoi(idStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{
			Code:    "INVALID_ID",
			Message: "id must be a number",
		})
		return
	}

	switch r.Method {
	case http.MethodGet:
		handleGetTodo(w, r, id)
	case http.MethodDelete:
		handleDeleteTodo(w, r, id)
	case http.MethodPatch:
		handleUpdateTodo(w, r, id)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, ErrorResponse{
			Code:    "METHOD_NOT_ALLOWED",
			Message: "use GET, DELETE or PATCH",
		})
	}
}

// handleGetTodo 处理 GET /todos/{id}，返回单个 Todo
func handleGetTodo(w http.ResponseWriter, r *http.Request, id int) {
	// 从 map 中查找，ok 表示是否找到
	todo, ok := todos[id]
	if !ok {
		// 找不到 → 404 Not Found
		writeJSON(w, http.StatusNotFound, ErrorResponse{
			Code:    "NOT_FOUND",
			Message: fmt.Sprintf("todo %d not found", id),
		})
		return
	}
	// 找到了 → 200 + Todo 数据
	writeJSON(w, http.StatusOK, todo)
}

// handleDeleteTodo 处理 DELETE /todos/{id}，删除一个 Todo
func handleDeleteTodo(w http.ResponseWriter, r *http.Request, id int) {
	// 先检查是否存在
	if _, ok := todos[id]; !ok {
		writeJSON(w, http.StatusNotFound, ErrorResponse{
			Code:    "NOT_FOUND",
			Message: fmt.Sprintf("todo %d not found", id),
		})
		return
	}
	// 从 map 中删除
	delete(todos, id)
	// 204 No Content：删除成功，不返回任何内容
	w.WriteHeader(http.StatusNoContent)
}

// validTransitions 定义了合法的状态转换
// key 是当前状态，value 是允许转换到的下一个状态
// pending → doing → done，不能跳跃，不能倒退
var validTransitions = map[string]string{
	"pending": "doing",
	"doing":   "done",
}

// handleUpdateTodo 处理 PATCH /todos/{id}，更新 Todo 的状态
func handleUpdateTodo(w http.ResponseWriter, r *http.Request, id int) {
	// 第一步：检查 Todo 是否存在
	todo, ok := todos[id]
	if !ok {
		writeJSON(w, http.StatusNotFound, ErrorResponse{
			Code:    "NOT_FOUND",
			Message: fmt.Sprintf("todo %d not found", id),
		})
		return
	}

	// 第二步：读取请求体
	var req UpdateTodoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{
			Code:    "INVALID_JSON",
			Message: "request body is not valid JSON",
		})
		return
	}

	// 第三步：验证状态转换是否合法
	// 比如当前是 pending，只能转到 doing；不能直接跳到 done
	nextStatus, exists := validTransitions[todo.Status]
	if !exists || nextStatus != req.Status {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{
			Code:    "INVALID_TRANSITION",
			Message: fmt.Sprintf("cannot change from %q to %q", todo.Status, req.Status),
		})
		return
	}

	// 第四步：更新状态
	todo.Status = req.Status
	writeJSON(w, http.StatusOK, todo)
}

// handleCreateTodo 处理 POST /todos，创建一个新的 Todo
func handleCreateTodo(w http.ResponseWriter, r *http.Request) {
	// 第一步：读取请求体，把 JSON 解析成 CreateTodoRequest 结构体
	// json.NewDecoder(r.Body) 从请求体创建一个 JSON 解码器
	// .Decode(&req) 把 JSON 解析到 req 变量里（注意取地址符 &）
	var req CreateTodoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// JSON 格式不对（比如客户端发了 "hello" 而不是 {"title":"..."}）
		writeJSON(w, http.StatusBadRequest, ErrorResponse{
			Code:    "INVALID_JSON",
			Message: "request body is not valid JSON",
		})
		return // early return：出错就立即返回，不往下走
	}

	// 第二步：验证参数
	if req.Title == "" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{
			Code:    "MISSING_TITLE",
			Message: "title is required",
		})
		return
	}

	// 第三步：创建 Todo 并存进 map
	// &Todo{...} 创建一个 Todo 结构体并返回它的指针
	todo := &Todo{
		ID:        nextID,
		Title:     req.Title,
		Status:    "pending",    // 新建的 Todo 默认是 pending 状态
		CreatedAt: time.Now(),   // 记录创建时间
	}
	todos[nextID] = todo // 存进 map
	nextID++             // ID 自增，下一个 Todo 用下一个数字

	// 第四步：返回创建好的 Todo
	// 201 Created 表示"资源已成功创建"，比 200 OK 更精确
	writeJSON(w, http.StatusCreated, todo)
}

// ---------- 启动 ----------

func main() {
	// 注册路由：路径 → 处理函数
	// /healthz 是健康检查接口，Kubernetes 用它判断服务是否存活
	http.HandleFunc("/healthz", handleHealthz)
	// /todos 和 /todos/{id} 用 requireAuth 包裹，需要认证才能访问
	// /healthz 不包裹，任何人都能调（Kubernetes 健康检查不带 Key）
	http.HandleFunc("/todos", requireAuth(handleTodos))
	http.HandleFunc("/todos/", requireAuth(handleTodoByID))

	// 启动 HTTP 服务器，监听 9090 端口
	// 这行会阻塞（一直运行），持续等待并处理请求
	fmt.Println("server started on :9090")
	http.ListenAndServe(":9090", nil)
}
