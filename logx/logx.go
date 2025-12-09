package logx

import (
	"context"
	"log/slog"
	"os"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/trace"
)

type traceIDHandler struct {
	slog.Handler
}

func (h *traceIDHandler) Handle(ctx context.Context, r slog.Record) error {

	if spanCtx := trace.SpanContextFromContext(ctx); spanCtx.HasTraceID() {

		r.Add("traceID", spanCtx.TraceID().String())
	} else {
		r.Add("traceID", "no trace")
	}

	return h.Handler.Handle(ctx, r)
}

func InitTraceIDHandler() {

	jsonHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		AddSource: false,
		Level:     slog.LevelDebug,
	})

	customHandler := &traceIDHandler{jsonHandler}

	slog.SetDefault(slog.New(customHandler))
}

func Info(c *gin.Context, msg string, args ...any) {

	logArgs := append([]any{"http request",
		"method", c.Request.Method,
		"path", c.Request.URL.Path,
		"user_agent", c.Request.UserAgent(),
		"status", c.Writer.Status()}, args...)

	slog.InfoContext(c.Request.Context(), msg, logArgs...)
}

func Error(c *gin.Context, msg string, args ...any) {

	logArgs := append([]any{"http request",
		"method", c.Request.Method,
		"path", c.Request.URL.Path,
		"user_agent", c.Request.UserAgent(),
		"status", c.Writer.Status()}, args...)

	slog.ErrorContext(c.Request.Context(), msg, logArgs...)
}
