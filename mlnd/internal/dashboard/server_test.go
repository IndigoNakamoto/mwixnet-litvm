package dashboard

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestServer_checkAuth_noToken(t *testing.T) {
	s := &Server{}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/status", nil)
	rec := httptest.NewRecorder()
	if !s.checkAuth(rec, req) {
		t.Fatal("expected ok")
	}
}

func TestServer_checkAuth_query(t *testing.T) {
	s := &Server{token: "sekret"}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/status?token=sekret", nil)
	rec := httptest.NewRecorder()
	if !s.checkAuth(rec, req) {
		t.Fatal("expected ok")
	}
}

func TestServer_checkAuth_header(t *testing.T) {
	s := &Server{token: "sekret"}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/status", nil)
	req.Header.Set("X-MLND-Token", "sekret")
	rec := httptest.NewRecorder()
	if !s.checkAuth(rec, req) {
		t.Fatal("expected ok")
	}
}

func TestServer_checkAuth_bearer(t *testing.T) {
	s := &Server{token: "sekret"}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/status", nil)
	req.Header.Set("Authorization", "Bearer sekret")
	rec := httptest.NewRecorder()
	if !s.checkAuth(rec, req) {
		t.Fatal("expected ok")
	}
}

func TestServer_checkAuth_reject(t *testing.T) {
	s := &Server{token: "sekret"}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/status", nil)
	rec := httptest.NewRecorder()
	if s.checkAuth(rec, req) {
		t.Fatal("expected fail")
	}
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("code %d", rec.Code)
	}
}

func TestValidateDashboardAddr_loopback(t *testing.T) {
	if err := validateDashboardAddr("127.0.0.1:9842", ""); err != nil {
		t.Fatal(err)
	}
	if err := validateDashboardAddr("192.168.1.1:80", ""); err == nil {
		t.Fatal("expected error")
	}
	if err := validateDashboardAddr("192.168.1.1:80", "1"); err != nil {
		t.Fatal(err)
	}
}
