package py_embedding_service

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPyService_Get_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"embedding": [` + generate192ZerosJSON() + `]}`))
	}))
	defer srv.Close()

	svc := NewPyService(srv.URL + "/embed")

	audio := io.NopCloser(bytes.NewReader([]byte("dummy")))
	defer func() {
		_ = audio.Close()
	}()

	emb, err := svc.Get(audio)
	if err != nil {
		t.Fatalf("Get error: %v", err)
	}
	for i := 0; i < 192; i++ {
		if emb[i] != 0.0 {
			t.Fatalf("expected zero embedding at %d, got %v", i, emb[i])
		}
	}
}

func TestPyService_Get_BadResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		_, _ = w.Write([]byte(`{"error":"server"}`))
	}))
	defer srv.Close()

	svc := NewPyService(srv.URL + "/embed")

	audio := io.NopCloser(bytes.NewReader([]byte("dummy")))
	defer func() {
		_ = audio.Close()
	}()

	_, err := svc.Get(audio)
	if err == nil {
		t.Fatalf("expected error on 500 response")
	}
}

func generate192ZerosJSON() string {
	s := ""
	for i := 0; i < 192; i++ {
		if i > 0 {
			s += ","
		}
		s += "0.0"
	}
	return s
}
