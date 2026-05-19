package main

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// ---------- 指标定义 ----------

// Counter：只增不减的计数器
// 每来一个请求就 +1，按 method 和 path 区分
// 比如 GET /todos 和 POST /todos 分开统计
var httpRequestsTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "http_requests_total",            // 指标名（Prometheus 中查询用）
		Help: "Total number of HTTP requests.", // 指标描述
	},
	[]string{"method", "path"}, // 标签：按这两个维度分类统计
)

// Histogram：记录值的分布
// 不只记录平均值，而是记录耗时落在哪些区间（桶）里
// 比如 0-10ms 有多少请求，10-50ms 有多少请求
// 这样可以算出 P50、P95、P99 等分位数
var httpRequestDuration = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Name:    "http_request_duration_ms",
		Help:    "HTTP request duration in milliseconds.",
		Buckets: []float64{1, 5, 10, 50, 100, 500, 1000}, // 桶的边界（毫秒）
	},
	[]string{"method", "path"},
)

// Gauge：可增可减的仪表盘
// 表示当前状态，比如当前有多少个 Todo
var todosTotal = prometheus.NewGauge(
	prometheus.GaugeOpts{
		Name: "todos_total",
		Help: "Current number of todos.",
	},
)

// init 在程序启动时自动执行，注册所有指标
func init() {
	prometheus.MustRegister(httpRequestsTotal)
	prometheus.MustRegister(httpRequestDuration)
	prometheus.MustRegister(todosTotal)
}

// ---------- 指标中间件 ----------

// metricsMiddleware 记录每个请求的计数和耗时
// 和 requireAuth 一样的中间件模式：包装 handler，在前后加逻辑
func metricsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now() // 记录开始时间

		next(w, r) // 调用原始 handler

		duration := float64(time.Since(start).Milliseconds()) // 计算耗时
		path := r.URL.Path

		httpRequestsTotal.WithLabelValues(r.Method, path).Inc()        // Counter +1
		httpRequestDuration.WithLabelValues(r.Method, path).Observe(duration) // 记录耗时
	}
}

// metricsHandler 返回 Prometheus 标准的 /metrics 端点
// Prometheus 服务器会定时请求这个端点来采集指标
func metricsHandler() http.Handler {
	return promhttp.Handler()
}
