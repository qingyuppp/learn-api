package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// ---------- 测试工具 ----------

// resetStore 清空数据，每个测试开始前调用，保证测试之间互不影响
func resetStore() {
	todos = make(map[int]*Todo)
	nextID = 1
}

// ---------- 测试 healthz ----------

// 测试函数必须以 Test 开头，参数是 *testing.T
func TestHealthz(t *testing.T) {
	// httptest.NewRequest 创建一个假的 HTTP 请求（不走网络）
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	// httptest.NewRecorder 创建一个假的 ResponseWriter，记录 handler 写了什么
	rec := httptest.NewRecorder()

	// 直接调用 handler 函数，不需要启动真正的服务器
	handleHealthz(rec, req)

	// 检查状态码是否是 200
	if rec.Code != http.StatusOK {
		// t.Errorf 报告测试失败，但继续执行后面的检查
		t.Errorf("healthz status = %d, want %d", rec.Code, http.StatusOK)
	}
}

// ---------- 测试创建 Todo ----------

func TestCreateTodo(t *testing.T) {
	resetStore()

	// strings.NewReader 把字符串变成 io.Reader，模拟请求体
	body := strings.NewReader(`{"title":"写测试"}`)
	req := httptest.NewRequest(http.MethodPost, "/todos", body)
	rec := httptest.NewRecorder()

	handleTodos(rec, req)

	// 创建成功应该返回 201
	if rec.Code != http.StatusCreated {
		t.Errorf("create status = %d, want %d", rec.Code, http.StatusCreated)
	}

	// 检查内存中确实存了一条
	if len(todos) != 1 {
		t.Errorf("todos count = %d, want 1", len(todos))
	}
}

func TestCreateTodoMissingTitle(t *testing.T) {
	resetStore()

	body := strings.NewReader(`{"title":""}`)
	req := httptest.NewRequest(http.MethodPost, "/todos", body)
	rec := httptest.NewRecorder()

	handleTodos(rec, req)

	// 缺少 title 应该返回 400
	if rec.Code != http.StatusBadRequest {
		t.Errorf("create empty title status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

// ---------- 测试列出 Todo ----------

func TestListTodos(t *testing.T) {
	resetStore()

	// 先创建两个
	for _, title := range []string{"任务A", "任务B"} {
		body := strings.NewReader(`{"title":"` + title + `"}`)
		req := httptest.NewRequest(http.MethodPost, "/todos", body)
		handleTodos(httptest.NewRecorder(), req)
	}

	// 再列出
	req := httptest.NewRequest(http.MethodGet, "/todos", nil)
	rec := httptest.NewRecorder()
	handleTodos(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("list status = %d, want %d", rec.Code, http.StatusOK)
	}
}

// ---------- 测试获取单个 Todo ----------

func TestGetTodoNotFound(t *testing.T) {
	resetStore()

	req := httptest.NewRequest(http.MethodGet, "/todos/999", nil)
	rec := httptest.NewRecorder()

	// 直接调用，传入 id
	handleGetTodo(rec, req, 999)

	// 不存在应该返回 404
	if rec.Code != http.StatusNotFound {
		t.Errorf("get missing todo status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

// ---------- 测试状态流转 ----------

func TestUpdateTodoStatus(t *testing.T) {
	resetStore()

	// 先创建一个 todo
	body := strings.NewReader(`{"title":"状态测试"}`)
	req := httptest.NewRequest(http.MethodPost, "/todos", body)
	handleTodos(httptest.NewRecorder(), req)

	// pending → doing（合法）
	body = strings.NewReader(`{"status":"doing"}`)
	req = httptest.NewRequest(http.MethodPatch, "/todos/1", body)
	rec := httptest.NewRecorder()
	handleUpdateTodo(rec, req, 1)

	if rec.Code != http.StatusOK {
		t.Errorf("pending→doing status = %d, want %d", rec.Code, http.StatusOK)
	}

	// doing → pending（非法，不能倒退）
	body = strings.NewReader(`{"status":"pending"}`)
	req = httptest.NewRequest(http.MethodPatch, "/todos/1", body)
	rec = httptest.NewRecorder()
	handleUpdateTodo(rec, req, 1)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("doing→pending status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

// ---------- 测试删除 ----------

func TestDeleteTodo(t *testing.T) {
	resetStore()

	// 先创建
	body := strings.NewReader(`{"title":"删掉我"}`)
	req := httptest.NewRequest(http.MethodPost, "/todos", body)
	handleTodos(httptest.NewRecorder(), req)

	// 删除
	req = httptest.NewRequest(http.MethodDelete, "/todos/1", nil)
	rec := httptest.NewRecorder()
	handleDeleteTodo(rec, req, 1)

	if rec.Code != http.StatusNoContent {
		t.Errorf("delete status = %d, want %d", rec.Code, http.StatusNoContent)
	}

	// 确认已删除
	if len(todos) != 0 {
		t.Errorf("todos count after delete = %d, want 0", len(todos))
	}
}
