package main

import (
	"bytes"
	"context"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/KasumiMercury/primind-notification-invoker/internal/fcm"
	"github.com/KasumiMercury/primind-notification-invoker/internal/observability/metrics"
)

func TestServerSmoke(t *testing.T) {
	adc := filepath.Join(t.TempDir(), "adc.json")
	if err := os.WriteFile(adc, []byte(`{"type":"authorized_user","client_id":"x","client_secret":"y","refresh_token":"z"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GOOGLE_APPLICATION_CREDENTIALS", adc)

	ctx := context.Background()
	fcmClient, err := fcm.NewClient(ctx, "test-project", "")
	if err != nil {
		t.Fatalf("fcm client: %v", err)
	}
	httpMetrics, err := metrics.NewHTTPMetrics()
	if err != nil {
		t.Fatalf("http metrics: %v", err)
	}

	srv := newServer("0", fcmClient, httpMetrics)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	go func() { _ = srv.Serve(ln) }()
	t.Cleanup(func() { _ = srv.Shutdown(context.Background()) })
	base := "http://" + ln.Addr().String()

	h2cProtocols := new(http.Protocols)
	h2cProtocols.SetUnencryptedHTTP2(true)
	h2c := &http.Client{Transport: &http.Transport{Protocols: h2cProtocols}}

	t.Run("grpc health check over h2c", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, base+"/grpc.health.v1.Health/Check", bytes.NewReader([]byte{0, 0, 0, 0, 0}))
		req.Header.Set("Content-Type", "application/grpc")
		req.Header.Set("TE", "trailers")
		resp, err := h2c.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)

		if resp.ProtoMajor != 2 {
			t.Errorf("proto = %s, want HTTP/2", resp.Proto)
		}
		if got := resp.Trailer.Get("Grpc-Status"); got != "0" {
			t.Errorf("grpc-status = %q, message = %q", got, resp.Trailer.Get("Grpc-Message"))
		}
		if want := []byte{0, 0, 0, 0, 2, 0x08, 0x01}; !bytes.Equal(body, want) {
			t.Errorf("body = %x, want %x (SERVING)", body, want)
		}
	})

	t.Run("readiness", func(t *testing.T) {
		resp, err := http.Get(base + "/health/ready")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != http.StatusOK || !strings.Contains(string(body), `"status":"healthy"`) {
			t.Errorf("status = %d, body = %s", resp.StatusCode, body)
		}
	})

	t.Run("notify rejects GET", func(t *testing.T) {
		resp, err := http.Get(base + "/notify")
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusMethodNotAllowed {
			t.Errorf("status = %d, want 405", resp.StatusCode)
		}
	})

	t.Run("notify rejects invalid request", func(t *testing.T) {
		for body, want := range map[string]string{
			`{`: "invalid JSON",
			`{"tokens":[],"taskId":"task-1","taskType":"TASK_TYPE_NEAR"}`: "validation error",
		} {
			resp, err := http.Post(base+"/notify", "application/json", strings.NewReader(body))
			if err != nil {
				t.Fatal(err)
			}
			got, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			if resp.StatusCode != http.StatusBadRequest || !strings.Contains(string(got), want) {
				t.Errorf("body %s: status = %d, response = %s", body, resp.StatusCode, got)
			}
		}
	})
}
