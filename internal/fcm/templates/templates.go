package templates

import (
	"embed"
	"encoding/json"
	"fmt"
	"math/rand"
	"sync"

	"github.com/KasumiMercury/primind-notification-invoker/internal/domain"
)

//go:embed messages.json
var messagesFS embed.FS

type Message struct {
	Title string
	Body  string
}

type TypeMessages struct {
	Title  string   `json:"title"`
	Bodies []string `json:"bodies"`
}

type MessagesConfig struct {
	Version string                  `json:"version"`
	Default TypeMessages            `json:"default"`
	Types   map[string]TypeMessages `json:"types"`
}

type Provider struct {
	config *MessagesConfig
	mu     sync.RWMutex
}

var (
	globalProvider *Provider
	once           sync.Once
	initErr        error
)

func GetProvider() (*Provider, error) {
	once.Do(func() {
		globalProvider, initErr = newProvider()
	})
	return globalProvider, initErr
}

func newProvider() (*Provider, error) {
	data, err := messagesFS.ReadFile("messages.json")
	if err != nil {
		return nil, fmt.Errorf("failed to read embedded messages.json: %w", err)
	}

	var config MessagesConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse messages.json: %w", err)
	}

	requiredTypes := []string{"short", "near", "relaxed", "scheduled"}
	for _, t := range requiredTypes {
		typeConfig, ok := config.Types[t]
		if !ok {
			return nil, fmt.Errorf("messages.json must have type: %s", t)
		}
		if typeConfig.Title == "" {
			return nil, fmt.Errorf("messages.json must have a title for type: %s", t)
		}
		if len(typeConfig.Bodies) == 0 {
			return nil, fmt.Errorf("messages.json must have at least one body for type: %s", t)
		}
	}

	if config.Default.Title == "" || len(config.Default.Bodies) == 0 {
		return nil, fmt.Errorf("messages.json must have a default title and at least one body")
	}

	return &Provider{config: &config}, nil
}

func (p *Provider) GetRandomMessage(taskType domain.Type) Message {
	p.mu.RLock()
	defer p.mu.RUnlock()

	typeKey := taskType.String()
	if typeConfig, ok := p.config.Types[typeKey]; ok && len(typeConfig.Bodies) > 0 {
		idx := rand.Intn(len(typeConfig.Bodies))
		return Message{
			Title: typeConfig.Title,
			Body:  typeConfig.Bodies[idx],
		}
	}

	// Fallback to default
	if len(p.config.Default.Bodies) > 0 {
		idx := rand.Intn(len(p.config.Default.Bodies))
		return Message{
			Title: p.config.Default.Title,
			Body:  p.config.Default.Bodies[idx],
		}
	}

	// Ultimate fallback (should never happen due to validation)
	return Message{
		Title: "お知らせ",
		Body:  "新しい通知があります",
	}
}

func (p *Provider) GetTitle(taskType domain.Type) string {
	p.mu.RLock()
	defer p.mu.RUnlock()

	typeKey := taskType.String()
	if typeConfig, ok := p.config.Types[typeKey]; ok {
		return typeConfig.Title
	}
	return p.config.Default.Title
}

func (p *Provider) GetAllBodies(taskType domain.Type) []string {
	p.mu.RLock()
	defer p.mu.RUnlock()

	typeKey := taskType.String()
	if typeConfig, ok := p.config.Types[typeKey]; ok {
		return typeConfig.Bodies
	}
	return p.config.Default.Bodies
}
