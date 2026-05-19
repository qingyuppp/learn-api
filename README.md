# learn-api

从零学习后端接口开发的练手项目。用 Go 标准库实现一个完整的 Todo RESTful API，每个 commit 对应一个学习步骤。

## 学习路径

查看 [commit 历史](../../commits/main) 按顺序学习：

1. **Hello World** — 最小 HTTP 服务器
2. **返回 JSON** — 结构体 + writeJSON 工具函数
3. **创建 Todo（POST）** — 读取请求体、JSON 解析、参数验证、early return
4. **列出 Todo（GET）** — 路由分发模式，同一路径不同方法
5. **获取单个 Todo（GET /todos/{id}）** — 路径参数提取
6. **删除 Todo（DELETE）** — delete 操作 + 204 No Content
7. **更新状态（PATCH）** — 状态机验证 pending → doing → done
8. **API Key 认证** — 中间件模式
9. **测试 + README** — httptest 包、测试隔离

## 接口一览

| 操作 | 方法 | 路径 | 说明 |
|------|------|------|------|
| 健康检查 | GET | /healthz | 不需要认证 |
| 创建 | POST | /todos | 需要认证 |
| 列出 | GET | /todos | 需要认证 |
| 获取 | GET | /todos/{id} | 需要认证 |
| 更新状态 | PATCH | /todos/{id} | 需要认证 |
| 删除 | DELETE | /todos/{id} | 需要认证 |

## 运行

```bash
# 不需要认证
go run main.go

# 开启 API Key 认证
API_KEY=my-secret-key go run main.go
```

## 测试

```bash
go test -v
```

## 试用

```bash
# 创建
curl -X POST http://localhost:9090/todos -d '{"title":"买菜"}' -H "X-API-Key: my-secret-key"

# 列出
curl http://localhost:9090/todos -H "X-API-Key: my-secret-key"

# 获取
curl http://localhost:9090/todos/1 -H "X-API-Key: my-secret-key"

# 更新状态
curl -X PATCH http://localhost:9090/todos/1 -d '{"status":"doing"}' -H "X-API-Key: my-secret-key"

# 删除
curl -X DELETE http://localhost:9090/todos/1 -H "X-API-Key: my-secret-key"
```
