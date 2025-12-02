package main

import (
	"context"
	"log"
	"log/slog"
	"os"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"

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

	httpRequestTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "http_request_total",
		Help: "Total number of http requests",
	}, []string{"method", "path"})

	httpRequestDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name: "http_request_duration_seconds",
		Help: "HTTP request duration in seconds",
	}, []string{"method", "path"})

	tracer trace.Tracer
)

func init() {
	prometheus.MustRegister(onlineUsers, requestCount, httpRequestTotal, httpRequestDuration)
}

func initTracer() {
	exporter, err := otlptracehttp.New(context.Background(), otlptracehttp.WithEndpoint("tempo:4317"))
	if err != nil {
		log.Fatal(err)
	}

	res := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceName("go-app"),
	)

	provider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)

	otel.SetTracerProvider(provider)
	tracer = provider.Tracer("http-handler")

}

func main() {

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		AddSource: false,
		Level:     slog.LevelDebug,
	}))
	slog.SetDefault(logger)
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
	tracer := otel.Tracer("go-app")

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
		MetricsMiddleware(),
	)

	r.GET("/hello", func(c *gin.Context) {
		requestCount.Inc()
		c.JSON(200, gin.H{"msg": "ok"})
	})

	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	r.GET("/api/users/:id", func(c *gin.Context) {
		c.JSON(200, gin.H{"user_id": c.Param("id")})
	})

	r.GET("/error", func(c *gin.Context) {
		start := time.Now()
		duration := time.Since(start)
		_, span := tracer.Start(c.Request.Context(), "handleRequest")
		if c.Query("error") == "1" {
			slog.Error("http request",
				"method", c.Request.Method,
				"path", c.Request.URL.Path,
				"user_agent", c.Request.UserAgent(),
				"status", c.Writer.Status(),
				"duration_ms", duration.Milliseconds(),
				"trace_id", span.SpanContext().TraceID().String(),
			)
			c.JSON(500, gin.H{"msg": "error"})
			return
		}

		slog.Info("http request",
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"user_agent", c.Request.UserAgent(),
			"status", c.Writer.Status(),
			"duration_ms", duration.Milliseconds(),
			"trace_id", span.SpanContext().TraceID().String(),
		)

		c.JSON(200, gin.H{"msg": "ok"})
	})

	// 启动服务器
	r.Run(":8080")
}
