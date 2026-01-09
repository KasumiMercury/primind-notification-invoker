package fcm

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"

	"github.com/KasumiMercury/primind-notification-invoker/internal/domain"
	"github.com/KasumiMercury/primind-notification-invoker/internal/model"
)

const maxTokensPerBatch = 500

type NotificationTemplate struct {
	Title string
	Body  string
}

var notificationTemplates = map[domain.Type]NotificationTemplate{
	domain.TypeShort:     {Title: "Urgency Task", Body: "Urgency Task Notification"},
	domain.TypeNear:      {Title: "Normal Task", Body: "Task Notification"},
	domain.TypeRelaxed:   {Title: "Low Task", Body: "Low Priority Task Notification"},
	domain.TypeScheduled: {Title: "Scheduled Task", Body: "Scheduled Task Notification"},
}

var defaultTemplate = NotificationTemplate{
	Title: "Notification",
	Body:  "New notification",
}

type Client struct {
	messagingClient *messaging.Client
	webAppBaseURL   string
}

func NewClient(ctx context.Context, projectID, webAppBaseURL string) (*Client, error) {
	config := &firebase.Config{}
	if projectID != "" {
		config.ProjectID = projectID
	}

	app, err := firebase.NewApp(ctx, config)
	if err != nil {
		return nil, err
	}

	msgClient, err := app.Messaging(ctx)
	if err != nil {
		return nil, err
	}

	return &Client{
		messagingClient: msgClient,
		webAppBaseURL:   webAppBaseURL,
	}, nil
}

type BulkResult struct {
	Total        int
	SuccessCount int
	FailureCount int
	Results      []model.TokenResult
}

func (c *Client) SendBulkNotification(ctx context.Context, tokens []domain.FCMToken, taskID domain.TaskID, taskType domain.Type, color string) (*BulkResult, error) {
	if len(tokens) <= maxTokensPerBatch {
		return c.sendBatch(ctx, tokens, taskID, taskType, color)
	}

	slog.Debug("splitting tokens into batches",
		"total_tokens", len(tokens),
		"max_per_batch", maxTokensPerBatch,
	)

	var allResults []model.TokenResult
	successCount, failureCount := 0, 0

	for i := 0; i < len(tokens); i += maxTokensPerBatch {
		end := i + maxTokensPerBatch
		if end > len(tokens) {
			end = len(tokens)
		}
		batch := tokens[i:end]
		batchNum := (i / maxTokensPerBatch) + 1

		slog.Debug("sending batch", "batch_number", batchNum, "batch_size", len(batch))

		result, err := c.sendBatch(ctx, batch, taskID, taskType, color)
		if err != nil {
			slog.Error("batch send failed", "batch_number", batchNum, "error", err)
			return nil, err
		}

		allResults = append(allResults, result.Results...)
		successCount += result.SuccessCount
		failureCount += result.FailureCount

		slog.Debug("batch completed",
			"batch_number", batchNum,
			"success_count", result.SuccessCount,
			"failure_count", result.FailureCount,
		)
	}

	return &BulkResult{
		Total:        len(tokens),
		SuccessCount: successCount,
		FailureCount: failureCount,
		Results:      allResults,
	}, nil
}

func (c *Client) sendBatch(ctx context.Context, tokens []domain.FCMToken, taskID domain.TaskID, taskType domain.Type, color string) (*BulkResult, error) {
	template := getTemplate(taskType)
	tokenStrings := domain.ToStrings(tokens)

	notification := &messaging.Notification{
		Title: template.Title,
		Body:  template.Body,
	}

	message := &messaging.MulticastMessage{
		Data: map[string]string{
			"task_id":   taskID.String(),
			"task_type": taskType.String(),
		},
		Notification: notification,
		Tokens:       tokenStrings,
	}

	// Add icon URL if web app base URL is configured and color is provided
	if c.webAppBaseURL != "" && color != "" {
		iconURL := buildIconURL(c.webAppBaseURL, taskType, color)
		message.Webpush.Notification.Icon = iconURL
		message.Android.Notification.Icon = iconURL
		slog.Debug("notification icon URL set", "icon_url", iconURL)
	}

	response, err := c.messagingClient.SendEachForMulticast(ctx, message)
	if err != nil {
		slog.Error("FCM multicast send failed", "error", err, "token_count", len(tokens))
		return nil, err
	}

	results := make([]model.TokenResult, len(tokens))
	for i, resp := range response.Responses {
		results[i] = model.TokenResult{
			Token:     tokenStrings[i],
			Success:   resp.Success,
			MessageID: resp.MessageID,
		}
		if resp.Error != nil {
			results[i].Error = resp.Error.Error()
			slog.Warn("FCM send failed for token",
				"token_index", i,
				"error", resp.Error.Error(),
			)
		}
	}

	return &BulkResult{
		Total:        len(tokens),
		SuccessCount: response.SuccessCount,
		FailureCount: response.FailureCount,
		Results:      results,
	}, nil
}

func getTemplate(taskType domain.Type) NotificationTemplate {
	if template, ok := notificationTemplates[taskType]; ok {
		return template
	}
	return defaultTemplate
}

func buildIconURL(baseURL string, taskType domain.Type, color string) string {
	colorHex := strings.TrimPrefix(color, "#")
	return fmt.Sprintf("%s/api/notification-icon/%s/%s.png",
		strings.TrimSuffix(baseURL, "/"),
		strings.ToLower(taskType.String()),
		colorHex,
	)
}
