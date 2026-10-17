package fcm

import (
	"testing"

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
