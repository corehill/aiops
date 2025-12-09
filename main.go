package main

import (
	"context"
	"log"

	"github.com/corehill/aiops/logx"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel/trace"

	"time"
)

var (
	//onlineUsers = prometheus.NewGauge(prometheus.GaugeOpts{
	//	Name: "online_users",
	//	Help: "Current number of online users",
	//})
	//requestCount = prometheus.NewCounter(prometheus.CounterOpts{
	//	Name: "request_count",
	//	Help: "Number of requests received",
	//})
	//
	//httpRequestTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
	//	Name: "http_request_total",
	//	Help: "Total number of http requests",
	//}, []string{"method", "path", "traceID"})
	//
	//httpRequestDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
	//	Name: "http_request_duration_seconds",
	//	Help: "HTTP request duration in seconds",
	//}, []string{"method", "path"})

	tracer trace.Tracer
)

//func init() {
//	prometheus.MustRegister(onlineUsers, requestCount, httpRequestTotal, httpRequestDuration)
//}

// 初始化tracer

func main() {

	logx.InitTraceIDHandler()

	initTracer()

	err := initMetricSDK(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	// 初始化 Gin 路由
	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},                                       // 前端地址，必须写完整
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}, // 允许 POST 和 OPTIONS 方法
		AllowHeaders:     []string{"*"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,           // 允许前端传递 Content-Type 头（JSON 格式需要）
		MaxAge:           12 * time.Hour, // 预检请求有效期，避免频繁发 OPTIONS
	}),
		TracingMiddleware(),
		MetricsMiddleware(),
	)

	r.GET("/hello", func(c *gin.Context) {
		c.JSON(200, gin.H{"msg": "ok"})
	})

	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	r.GET("/api/users/:id", func(c *gin.Context) {
		c.JSON(200, gin.H{"user_id": c.Param("id")})
	})

	r.GET("/error", func(c *gin.Context) {
		start := time.Now()
		duration := time.Since(start)
		if c.Query("error") == "1" {
			logx.Error(c, "no", "duration_ms", duration.Milliseconds())
			c.JSON(500, gin.H{"msg": "error"})
			return
		}

		logx.Info(c, "ok", "duration_ms", duration.Milliseconds())

		c.JSON(200, gin.H{"msg": "ok"})
	})

	// 启动服务器
	r.Run(":8080")
}
