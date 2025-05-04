package gateway

import (
	"context"
	"github.com/JunBSer/FileManager/pkg/logger"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"net/http"
)

func CorsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
func LoggerMiddleware(l logger.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id, err := uuid.NewUUID()
			if err != nil {
				l.Error(r.Context(), "Error to create uuid for http request")
				next.ServeHTTP(w, r)
				return
			}

			requestLogger := l.CreateChildLogger(zap.String("requestID", id.String()))

			requestLogger.Info(r.Context(), "Request started",
				zap.String("method", r.Method),
				zap.String("url", r.RequestURI),
			)

			ctx := r.Context()
			ctx = context.WithValue(ctx, logger.RequestID, id.String())
			ctx = context.WithValue(ctx, logger.Key, requestLogger)
			r = r.WithContext(ctx)

			next.ServeHTTP(w, r)

			requestLogger.Info(r.Context(), "Request completed")
		})
	}
}

func TransportFromServMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Transfer-Encoding", "binary")

		next.ServeHTTP(w, r)
	})
}
