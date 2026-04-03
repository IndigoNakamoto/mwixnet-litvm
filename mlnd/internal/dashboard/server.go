package dashboard

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/IndigoNakamoto/mwixnet-litvm/mlnd/internal/opslog"
)

//go:embed web/dashboard/*
var webFS embed.FS

// Server is the optional loopback HTTP control plane for operators.
type Server struct {
	addr   string
	token  string
	ops    *opslog.Log
	deps   StatusDeps
	logger *log.Logger
}

// NewServer validates listen addr (loopback by default) and returns a configured Server.
func NewServer(addr, token string, allowLAN string, ops *opslog.Log, deps StatusDeps, lg *log.Logger) (*Server, error) {
	if strings.TrimSpace(addr) == "" {
		return nil, fmt.Errorf("empty dashboard listen addr")
	}
	if err := validateDashboardAddr(addr, allowLAN); err != nil {
		return nil, err
	}
	if lg == nil {
		lg = log.Default()
	}
	return &Server{addr: addr, token: strings.TrimSpace(token), ops: ops, deps: deps, logger: lg}, nil
}

func validateDashboardAddr(addr, allowLAN string) error {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return fmt.Errorf("MLND_DASHBOARD_ADDR must be host:port: %w", err)
	}
	if envTruthy(allowLAN) {
		return nil
	}
	if host == "localhost" || host == "127.0.0.1" || host == "::1" {
		return nil
	}
	if ip := net.ParseIP(host); ip != nil && ip.IsLoopback() {
		return nil
	}
	return fmt.Errorf("dashboard bind host %q must be loopback (use MLND_DASHBOARD_ALLOW_LAN=1 to bind elsewhere)", host)
}

func envTruthy(s string) bool {
	s = strings.TrimSpace(strings.ToLower(s))
	return s == "1" || s == "true" || s == "yes"
}

// Run serves until ctx is cancelled.
func (s *Server) Run(ctx context.Context) error {
	sub, err := fs.Sub(webFS, "web/dashboard")
	if err != nil {
		return err
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/status", s.withAuthFunc(s.handleStatus))
	mux.HandleFunc("/api/v1/events", s.withAuthFunc(s.handleEvents))
	mux.Handle("/", s.withAuthHandler(http.FileServer(http.FS(sub))))

	srv := &http.Server{
		Addr:              s.addr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		s.logger.Printf("mlnd dashboard: listening on http://%s/", s.addr)
		errCh <- srv.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		shCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shCtx)
		return nil
	case err := <-errCh:
		if err != nil && err != http.ErrServerClosed {
			return err
		}
		return nil
	}
}

func (s *Server) checkAuth(w http.ResponseWriter, r *http.Request) bool {
	if s.token == "" {
		return true
	}
	q := r.URL.Query().Get("token")
	if q == s.token || r.Header.Get("X-MLND-Token") == s.token {
		return true
	}
	auth := r.Header.Get("Authorization")
	const p = "Bearer "
	if strings.HasPrefix(auth, p) && strings.TrimPrefix(auth, p) == s.token {
		return true
	}
	http.Error(w, "unauthorized", http.StatusUnauthorized)
	return false
}

func (s *Server) withAuthFunc(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !s.checkAuth(w, r) {
			return
		}
		next(w, r)
	}
}

func (s *Server) withAuthHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !s.checkAuth(w, r) {
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 25*time.Second)
	defer cancel()
	st := BuildStatus(ctx, s.deps)
	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(true)
	if err := enc.Encode(st); err != nil {
		s.logger.Printf("mlnd dashboard: encode status: %v", err)
	}
}

func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if s.ops == nil {
		http.Error(w, "ops log unavailable", http.StatusInternalServerError)
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	for _, e := range s.ops.Snapshot() {
		b, err := json.Marshal(e)
		if err != nil {
			continue
		}
		fmt.Fprintf(w, "data: %s\n\n", b)
		flusher.Flush()
	}

	ch := s.ops.Subscribe(r.Context())
	for ev := range ch {
		b, err := json.Marshal(ev)
		if err != nil {
			continue
		}
		fmt.Fprintf(w, "data: %s\n\n", b)
		flusher.Flush()
	}
}
