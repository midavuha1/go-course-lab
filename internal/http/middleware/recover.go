package middleware

import (
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/midavuha1/go-course-lab/internal/http/domainerror"
)

func Recover(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				rec := recover()
				if rec == nil {
					return
				}
				if rec == http.ErrAbortHandler {
					panic(rec)
				}
				logger.ErrorContext(r.Context(), "panic recovered",
					slog.Any("panic", rec), slog.String("stack", string(debug.Stack())))
				domainerror.Respond(w, r, fmt.Errorf("panic: %v", rec))
			}()
			next.ServeHTTP(w, r)
		})
	}
}
