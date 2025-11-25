package main

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"time"
)

var (
	onlineUsers = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "online_users",
		Help: "Current number of online users",
	})
	requestCount = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "request_count",
		Help: "Number of requests received",
	})
)

func init() {
	prometheus.MustRegister(onlineUsers)
	prometheus.MustRegister(requestCount)
}

func main() {
	go func() {
		users := 100
		for {
			onlineUsers.Set(float64(users))
			users += 10
			if users > 1000 {
				users = 100
			}
			time.Sleep(10 * time.Second)
		}
	}()
	// 初始化 Gin 路由
	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},                                       // 前端地址，必须写完整
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}, // 允许 POST 和 OPTIONS 方法
		AllowHeaders:     []string{"*"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,           // 允许前端传递 Content-Type 头（JSON 格式需要）
		MaxAge:           12 * time.Hour, // 预检请求有效期，避免频繁发 OPTIONS
	}))

	r.GET("/hello", func(c *gin.Context) {
		requestCount.Inc()
		c.JSON(200, gin.H{"msg": "ok"})
	})

	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// 启动服务器
	r.Run(":8080")
}
