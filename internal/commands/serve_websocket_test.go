package commands

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/99designs/gqlgen/graphql/handler/transport"
	coderws "github.com/coder/websocket"

	"github.com/xRiErOS/beans/internal/cors"
)

// TestGqlWebsocketImplementationRejectsDisallowedOrigin pins the origin check
// that replaced gorilla/websocket's Upgrader.CheckOrigin when gqlgen moved to
// the pluggable WebsocketImplementation interface (v0.17.9x). A regression
// here would let any origin open a GraphQL websocket (CSWSH).
func TestGqlWebsocketImplementationRejectsDisallowedOrigin(t *testing.T) {
	checker := cors.NewChecker([]string{"http://allowed.example"})
	impl := gqlWebsocketImplementation{checkOrigin: checker.CheckOriginFunc()}

	req := httptest.NewRequest(http.MethodGet, "/api/graphql", nil)
	req.Header.Set("Origin", "http://evil.example")
	rec := httptest.NewRecorder()

	conn, err := impl.Accept(rec, req, transport.WebsocketAcceptOptions{})
	if err == nil {
		if conn != nil {
			conn.Close()
		}
		t.Fatal("expected Accept to reject a disallowed origin, got nil error")
	}
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for a disallowed origin, got %d", rec.Code)
	}
}

// TestGqlWebsocketImplementationAcceptsAllowedOrigin proves the allowed path
// still completes a real websocket handshake through coder/websocket, the
// implementation gqlgen now defaults to.
func TestGqlWebsocketImplementationAcceptsAllowedOrigin(t *testing.T) {
	checker := cors.NewChecker([]string{"http://allowed.example"})
	impl := gqlWebsocketImplementation{checkOrigin: checker.CheckOriginFunc()}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := impl.Accept(w, r, transport.WebsocketAcceptOptions{})
		if err != nil {
			return
		}
		defer conn.Close()
		conn.NextReader() //nolint:errcheck // block until the client closes
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	url := "ws" + srv.URL[len("http"):]
	c, _, err := coderws.Dial(ctx, url, &coderws.DialOptions{
		HTTPHeader: http.Header{"Origin": []string{"http://allowed.example"}},
	})
	if err != nil {
		t.Fatalf("expected the allowed origin to complete the handshake, got: %v", err)
	}
	c.Close(coderws.StatusNormalClosure, "")
}
