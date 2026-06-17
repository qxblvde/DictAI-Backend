package transcriber

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestTranscript_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"success":true,"language":"en","segments":[{"start":0.0,"end":1.23,"text":"hello"},{"start":1.24,"end":2.00,"text":"world"}]}`)
	}))
	defer srv.Close()

	w := NewWhisper(srv.URL)
	r := io.NopCloser(strings.NewReader("raw-audio"))
	out, err := w.Transcript(context.Background(), r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "0.00-1.23: hello") || !strings.Contains(out, "1.24-2.00: world") {
		t.Fatalf("unexpected output: %q", out)
	}
}

func TestTranscript_Non200(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(w, `bad request`)
	}))
	defer srv.Close()

	w := NewWhisper(srv.URL)
	r := io.NopCloser(strings.NewReader("x"))
	_, err := w.Transcript(context.Background(), r)
	if err == nil {
		t.Fatal("expected error for non-200 response")
	}
}

func TestTranscript_BadJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, `invalid-json`)
	}))
	defer srv.Close()

	w := NewWhisper(srv.URL)
	r := io.NopCloser(strings.NewReader("x"))
	_, err := w.Transcript(context.Background(), r)
	if err == nil {
		t.Fatal("expected JSON decode error")
	}
}

func TestTranscript_SuccessFalse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, `{"success":false}`)
	}))
	defer srv.Close()

	w := NewWhisper(srv.URL)
	r := io.NopCloser(strings.NewReader("x"))
	_, err := w.Transcript(context.Background(), r)
	if err == nil {
		t.Fatal("expected error for success=false")
	}
}
