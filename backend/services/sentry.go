package services

import (
	"log"
	"time"

	"kikundibora/config"

	"github.com/getsentry/sentry-go"
)

// Enabled reports whether Sentry was initialized with a DSN.
var Enabled = false

// InitSentry wires the Sentry SDK. A missing SENTRY_DSN disables Sentry
// silently (local-dev default) — the app runs exactly as before.
func InitSentry() {
	dsn := config.AppConfig.SentryDSN
	if dsn == "" {
		log.Println("Sentry disabled (SENTRY_DSN not set)")
		return
	}
	if err := sentry.Init(sentry.ClientOptions{
		Dsn:              dsn,
		Environment:      config.AppConfig.Environment,
		TracesSampleRate: config.AppConfig.SentryTracesSampleRate,
		AttachStacktrace: true,
	}); err != nil {
		log.Printf("Sentry init failed (continuing without Sentry): %s", err)
		return
	}
	Enabled = true
	log.Printf("Sentry enabled (env=%s traces=%.2f)",
		config.AppConfig.Environment, config.AppConfig.SentryTracesSampleRate)
}

// FlushSentry blocks until buffered events are delivered (call on shutdown).
func FlushSentry() {
	if !Enabled {
		return
	}
	sentry.Flush(2 * time.Second)
}

// CaptureError reports a 5xx-class error with request context. Callers must
// check Enabled first (main.errorHandler does) — this keeps sentry-go out
// of files that only need the flag.
func CaptureError(err error, method, path string) {
	if !Enabled || err == nil {
		return
	}
	sentry.WithScope(func(scope *sentry.Scope) {
		scope.SetTag("method", method)
		scope.SetContext("request", sentry.Context{"path": path})
		sentry.CaptureException(err)
	})
}
