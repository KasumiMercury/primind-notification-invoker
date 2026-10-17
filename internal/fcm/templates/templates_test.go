package templates

import (
	"testing"

	"github.com/KasumiMercury/primind-notification-invoker/internal/domain"
)

func TestGetProvider(t *testing.T) {
	provider, err := GetProvider()
	if err != nil {
		t.Fatalf("failed to get provider: %v", err)
	}
	if provider == nil {
		t.Fatal("provider is nil")
	}
}

func TestGetRandomMessage_AllTypes(t *testing.T) {
	provider, err := GetProvider()
	if err != nil {
		t.Fatalf("failed to get provider: %v", err)
	}

	types := []domain.Type{
		domain.TypeShort,
		domain.TypeNear,
		domain.TypeRelaxed,
		domain.TypeScheduled,
	}

	for _, taskType := range types {
		t.Run(taskType.String(), func(t *testing.T) {
			msg := provider.GetRandomMessage(taskType)
			if msg.Title == "" {
				t.Error("title should not be empty")
			}
			if msg.Body == "" {
				t.Error("body should not be empty")
			}
		})
	}
}

func TestGetRandomMessage_TitleIsFixed(t *testing.T) {
	provider, err := GetProvider()
	if err != nil {
		t.Fatalf("failed to get provider: %v", err)
	}

	types := []domain.Type{
		domain.TypeShort,
		domain.TypeNear,
		domain.TypeRelaxed,
		domain.TypeScheduled,
	}

	for _, taskType := range types {
		t.Run(taskType.String(), func(t *testing.T) {
			expectedTitle := provider.GetTitle(taskType)

			// Call multiple times and verify title is always the same
			for i := 0; i < 20; i++ {
				msg := provider.GetRandomMessage(taskType)
				if msg.Title != expectedTitle {
					t.Errorf("title should be fixed, expected %q but got %q", expectedTitle, msg.Title)
				}
			}
		})
	}
}

func TestGetRandomMessage_BodyRandomness(t *testing.T) {
	provider, err := GetProvider()
	if err != nil {
		t.Fatalf("failed to get provider: %v", err)
	}

	// Call 100 times and verify at least 2 different bodies are returned
	seen := make(map[string]bool)
	for i := 0; i < 100; i++ {
		msg := provider.GetRandomMessage(domain.TypeShort)
		seen[msg.Body] = true
	}

	if len(seen) < 2 {
		t.Errorf("expected at least 2 different bodies, got %d", len(seen))
	}
}

func TestGetAllBodies(t *testing.T) {
	provider, err := GetProvider()
	if err != nil {
		t.Fatalf("failed to get provider: %v", err)
	}

	types := []domain.Type{
		domain.TypeShort,
		domain.TypeNear,
		domain.TypeRelaxed,
		domain.TypeScheduled,
	}

	for _, taskType := range types {
		t.Run(taskType.String(), func(t *testing.T) {
			bodies := provider.GetAllBodies(taskType)
			if len(bodies) < 2 {
				t.Errorf("expected at least 2 bodies for %s, got %d", taskType.String(), len(bodies))
			}
			for i, body := range bodies {
				if body == "" {
					t.Errorf("body %d for %s is empty", i, taskType.String())
				}
			}
		})
	}
}

func TestGetTitle(t *testing.T) {
	provider, err := GetProvider()
	if err != nil {
		t.Fatalf("failed to get provider: %v", err)
	}

	defaultTitle := provider.GetTitle(domain.Type("unknown"))
	types := []domain.Type{
		domain.TypeShort,
		domain.TypeNear,
		domain.TypeRelaxed,
		domain.TypeScheduled,
	}

	seen := make(map[string]domain.Type)
	for _, taskType := range types {
		t.Run(taskType.String(), func(t *testing.T) {
			title := provider.GetTitle(taskType)
			if title == "" {
				t.Fatalf("expected non-empty title for %s", taskType.String())
			}
			if title == defaultTitle {
				t.Errorf("expected type-specific title for %s, got default %q", taskType.String(), title)
			}
			if other, ok := seen[title]; ok {
				t.Errorf("title %q for %s duplicates %s", title, taskType.String(), other.String())
			}
			seen[title] = taskType
		})
	}
}

func TestGetRandomMessage_UnknownType(t *testing.T) {
	provider, err := GetProvider()
	if err != nil {
		t.Fatalf("failed to get provider: %v", err)
	}

	// Test with an unknown type - should fallback to default
	unknownType := domain.Type("unknown")
	msg := provider.GetRandomMessage(unknownType)

	// Should return the default message
	if msg.Title == "" {
		t.Error("title should not be empty for unknown type")
	}
	if msg.Body == "" {
		t.Error("body should not be empty for unknown type")
	}
	// Default title should be "リマインド"
	if msg.Title != "リマインド" {
		t.Errorf("expected default title 'リマインド', got %q", msg.Title)
	}
}
