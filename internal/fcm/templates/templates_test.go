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
			if len(bodies) == 0 {
				t.Errorf("expected at least one body for %s", taskType.String())
			}
			// Verify we have 5 bodies per type as defined in messages.json
			if len(bodies) != 5 {
				t.Errorf("expected 5 bodies for %s, got %d", taskType.String(), len(bodies))
			}
		})
	}
}

func TestGetTitle(t *testing.T) {
	provider, err := GetProvider()
	if err != nil {
		t.Fatalf("failed to get provider: %v", err)
	}

	expectedTitles := map[domain.Type]string{
		domain.TypeShort:     "「すぐ」のタスクがあります",
		domain.TypeNear:      "そろそろ取りかかりませんか？",
		domain.TypeRelaxed:   "のんびりでも大丈夫です",
		domain.TypeScheduled: "予定の時間です",
	}

	for taskType, expectedTitle := range expectedTitles {
		t.Run(taskType.String(), func(t *testing.T) {
			title := provider.GetTitle(taskType)
			if title != expectedTitle {
				t.Errorf("expected title %q for %s, got %q", expectedTitle, taskType.String(), title)
			}
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
