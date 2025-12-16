package fcm

import (
	"context"

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
	domain.TypeUrgent:    {Title: "Urgency Task", Body: "Urgency Task Notification"},
	domain.TypeNormal:    {Title: "Normal Task", Body: "Task Notification"},
	domain.TypeLow:       {Title: "Low Task", Body: "Low Priority Task Notification"},
	domain.TypeScheduled: {Title: "Scheduled Task", Body: "Scheduled Task Notification"},
}

var defaultTemplate = NotificationTemplate{
	Title: "Notification",
	Body:  "New notification",
}

type Client struct {
	messagingClient *messaging.Client
}

func NewClient(ctx context.Context, projectID string) (*Client, error) {
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

	return &Client{messagingClient: msgClient}, nil
}

type BulkResult struct {
	Total        int
	SuccessCount int
	FailureCount int
	Results      []model.TokenResult
}

func (c *Client) SendBulkNotification(ctx context.Context, tokens []domain.FCMToken, taskID domain.TaskID, taskType domain.Type) (*BulkResult, error) {
	if len(tokens) <= maxTokensPerBatch {
		return c.sendBatch(ctx, tokens, taskID, taskType)
	}

	var allResults []model.TokenResult
	successCount, failureCount := 0, 0

	for i := 0; i < len(tokens); i += maxTokensPerBatch {
		end := i + maxTokensPerBatch
		if end > len(tokens) {
			end = len(tokens)
		}
		batch := tokens[i:end]

		result, err := c.sendBatch(ctx, batch, taskID, taskType)
		if err != nil {
			return nil, err
		}

		allResults = append(allResults, result.Results...)
		successCount += result.SuccessCount
		failureCount += result.FailureCount
	}

	return &BulkResult{
		Total:        len(tokens),
		SuccessCount: successCount,
		FailureCount: failureCount,
		Results:      allResults,
	}, nil
}

func (c *Client) sendBatch(ctx context.Context, tokens []domain.FCMToken, taskID domain.TaskID, taskType domain.Type) (*BulkResult, error) {
	template := getTemplate(taskType)
	tokenStrings := domain.ToStrings(tokens)

	message := &messaging.MulticastMessage{
		Data: map[string]string{
			"task_id":   taskID.String(),
			"task_type": taskType.String(),
		},
		Notification: &messaging.Notification{
			Title: template.Title,
			Body:  template.Body,
		},
		Tokens: tokenStrings,
	}

	response, err := c.messagingClient.SendEachForMulticast(ctx, message)
	if err != nil {
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
