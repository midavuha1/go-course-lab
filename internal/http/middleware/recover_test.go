package middleware_test

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/midavuha1/go-course-lab/internal/http/middleware"
)

func TestRecover_PanicBecomes500(t *testing.T) {
	panicky := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { panic("boom") })
	h := middleware.Recover(slog.New(slog.DiscardHandler))(panicky)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/trips", nil))

	if rec.Code != http.StatusInternalServerError || !strings.Contains(rec.Body.String(), "internal_error") {
		t.Fatalf("got %d %s, want 500 internal_error", rec.Code, rec.Body)
	}
	if strings.Contains(rec.Body.String(), "boom") {
		t.Fatal("panic value leaked to client")
	}
}
