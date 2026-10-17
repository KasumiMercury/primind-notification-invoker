package fcm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	firebase "firebase.google.com/go/v4"
	"google.golang.org/api/option"

	"github.com/KasumiMercury/primind-notification-invoker/internal/domain"
)

func TestBuildMessage_SetsIconWhenBaseURLAndColorGiven(t *testing.T) {
	c := &Client{webAppBaseURL: "https://app.example.com/"}

	msg := c.buildMessage([]string{"token"}, domain.TaskID("task-1"), domain.TypeNear, "#FF0000")

	want := "https://app.example.com/api/notification-icon/near/FF0000.png"
	if msg.Webpush == nil || msg.Webpush.Notification == nil || msg.Webpush.Notification.Icon != want {
		t.Errorf("webpush icon = %+v, want %q", msg.Webpush, want)
	}
	if msg.Android == nil || msg.Android.Notification == nil || msg.Android.Notification.Icon != want {
		t.Errorf("android icon = %+v, want %q", msg.Android, want)
	}
}

func TestBuildMessage_NoIconWithoutColor(t *testing.T) {
	c := &Client{webAppBaseURL: "https://app.example.com"}

	msg := c.buildMessage([]string{"token"}, domain.TaskID("task-1"), domain.TypeNear, "")

	if msg.Webpush != nil || msg.Android != nil {
		t.Errorf("expected no platform configs, got webpush=%+v android=%+v", msg.Webpush, msg.Android)
	}
}

func TestSendBulkNotification_ThroughFirebaseSDK(t *testing.T) {
	var mu sync.Mutex
	received := map[string]map[string]any{}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/projects/test-project/messages:send" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}

		var body struct {
			Message map[string]any `json:"message"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode request: %v", err)
		}

		token, _ := body.Message["token"].(string)
		mu.Lock()
		received[token] = body.Message
		mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		if token == "stale" {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error":{"code":404,"message":"not registered","status":"NOT_FOUND","details":[{"@type":"type.googleapis.com/google.firebase.fcm.v1.FcmError","errorCode":"UNREGISTERED"}]}}`))
			return
		}
		_, _ = w.Write([]byte(`{"name":"projects/test-project/messages/` + token + `"}`))
	}))
	defer srv.Close()

	ctx := context.Background()
	app, err := firebase.NewApp(ctx, &firebase.Config{ProjectID: "test-project"}, option.WithEndpoint(srv.URL), option.WithoutAuthentication())
	if err != nil {
		t.Fatalf("new app: %v", err)
	}
	msgClient, err := app.Messaging(ctx)
	if err != nil {
		t.Fatalf("messaging client: %v", err)
	}
	c := &Client{messagingClient: msgClient, webAppBaseURL: "https://app.example.com"}

	result, err := c.SendBulkNotification(ctx, []domain.FCMToken{"a", "stale", "b"}, domain.TaskID("task-1"), domain.TypeNear, "#00FF00")
	if err != nil {
		t.Fatalf("SendBulkNotification: %v", err)
	}

	if result.Total != 3 || result.SuccessCount != 2 || result.FailureCount != 1 {
		t.Errorf("counts = %d/%d/%d, want 3/2/1", result.Total, result.SuccessCount, result.FailureCount)
	}
	if got := result.Results[0]; !got.Success || got.Token != "a" || got.MessageID != "projects/test-project/messages/a" {
		t.Errorf("result[0] = %+v", got)
	}
	if got := result.Results[1]; got.Success || got.Token != "stale" || !strings.Contains(got.Error, "not registered") {
		t.Errorf("result[1] = %+v", got)
	}

	msg := received["b"]
	data, _ := msg["data"].(map[string]any)
	if data["task_id"] != "task-1" || data["task_type"] != "near" {
		t.Errorf("data = %v", data)
	}
	if n, _ := msg["notification"].(map[string]any); n["title"] == "" || n["title"] == nil {
		t.Errorf("notification = %v", msg["notification"])
	}
	webpush, _ := msg["webpush"].(map[string]any)
	wn, _ := webpush["notification"].(map[string]any)
	if wn["icon"] != "https://app.example.com/api/notification-icon/near/00FF00.png" {
		t.Errorf("webpush = %v", webpush)
	}
}
