package gateway

import (
	"context"
	"github.com/JunBSer/FileManager/pkg/logger"
	uuid2 "github.com/google/uuid"
	"go.uber.org/zap"
	"net/http"
)

func CorsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding")
		if r.Method == "OPTIONS" {
			return
		}
		next.ServeHTTP(w, r)
	})
}

func LoggerMiddleware(ctx context.Context, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		uuid, err := uuid2.NewUUID()
		if err != nil {
			logger.GetLoggerFromContext(ctx).Error(ctx, "Error to create uuid http request")
			next.ServeHTTP(w, r)
			return
		}

		logger.GetLoggerFromContext(ctx).Info(ctx, "Requested: ", zap.String("Request method", r.Method), zap.String("Request URL", r.RequestURI), zap.String("UUID", uuid.String()))

		r.WithContext(context.WithValue(r.Context(), logger.RequestID, uuid))
		r.WithContext(context.WithValue(r.Context(), logger.Key, logger.GetLoggerFromContext(ctx)))

		next.ServeHTTP(w, r)
		return
	})
}

func TransportFromServMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Transfer-Encoding", "binary")

		next.ServeHTTP(w, r)
	})
}
