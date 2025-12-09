package main

import (
	"context"
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/metric"
	metric_sdk "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
)

var (
	meter        metric.Meter
	reqCounter   metric.Int64Counter     // 计数器（核心 API）
	reqHistogram metric.Float64Histogram // 直方图（核心 API）
	onlineUsers  metric.Int64UpDownCounter
)

// 初始化 Metric SDK（1.38 版本，使用重命名的 SDK 包）
func initMetricSDK(ctx context.Context) error {
	// 1. 创建服务资源（通用资源，追踪/指标共用）
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName("go-app"), // 服务名必须一致
		),
	)
	if err != nil {
		return err
	}

	// 创建 Prometheus 导出器（核心：将 OTel 指标转换为 Prometheus 格式）
	exporter, err := prometheus.New(
		prometheus.WithNamespace("go_app"), // 可选：添加指标前缀避免冲突
		prometheus.WithoutScopeInfo(),
		prometheus.WithoutUnits(),
	)
	if err != nil {
		return err
	}

	// 2. 创建 MeterProvider（关键：使用 metric_sdk 而非 metric）
	mp := metric_sdk.NewMeterProvider(
		metric_sdk.WithResource(res), // 这里是 metric_sdk.WithResource，不是 metric.WithResource
		metric_sdk.WithReader(exporter),
		// 1.38 中 Exemplar 自动采集 trace 上下文，无需额外配置
	)
	otel.SetMeterProvider(mp)

	// 3. 创建 Meter 实例（核心 API）
	meter = otel.Meter("go-app/metrics")

	// 4. 初始化计数器（http_request_total）
	reqCounter, err = meter.Int64Counter(
		"http_request_total",
		metric.WithDescription("Total number of HTTP requests"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return err
	}

	// 5. 初始化直方图（http_request_duration_seconds）
	reqHistogram, err = meter.Float64Histogram(
		"http_request_duration_seconds",
		metric.WithDescription("HTTP request duration in seconds"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return err
	}

	// 6. 初始化在线用户指标数
	onlineUsers, err = meter.Int64UpDownCounter(
		"http_online_users",
		metric.WithDescription("Number of online users"),
		metric.WithUnit("1"))

	return nil
}

// MetricsMiddleware
func MetricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. 获取 trace 上下文（TracingMiddleware 已注入）
		ctx := c.Request.Context()
		spanCtx := trace.SpanContextFromContext(ctx)

		// 2. 构建 traceID 标签（非空，避免 Prometheus 省略）
		traceID := "unknown"
		if spanCtx.IsValid() {
			traceID = spanCtx.TraceID().String()
		}
		attrs := []attribute.KeyValue{
			attribute.String("method", c.Request.Method),
			attribute.String("path", c.FullPath()),
			attribute.String("traceID", traceID),
		}

		// 3. 记录请求开始时间
		start := time.Now()

		// 4. 执行后续中间件/接口逻辑
		c.Next()

		// 5. 计算响应时间
		duration := time.Since(start).Seconds()

		// 6. 记录指标（传入 trace 上下文，自动生成 Exemplar）
		reqCounter.Add(
			ctx,
			1,
			metric.WithAttributes(attrs...),
		)
		reqHistogram.Record(
			ctx,
			duration,
			metric.WithAttributes(attrs...),
		)

	}
}

func TracingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {

		//tracer := otel.Tracer("go-app/trace")
		// 从请求上下文创建span
		var ctx, span = tracer.Start(
			c.Request.Context(), c.FullPath())
		defer span.End() // 确保span结束时被记录

		// 将新上下文传递给后续处理
		c.Request = c.Request.WithContext(ctx)

		// 执行后续的handler
		c.Next()
	}
}

func initTracer() {
	exporter, err := otlptracehttp.New(context.Background(),
		otlptracehttp.WithEndpoint("tempo:4318"),
		otlptracehttp.WithInsecure())
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
