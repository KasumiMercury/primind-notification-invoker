package model

import (
	"github.com/KasumiMercury/primind-notification-invoker/internal/domain"
)

type NotificationRequest struct {
	Tokens   []string `json:"tokens"`
	TaskID   string   `json:"task_id"`
	TaskType string   `json:"task_type"`
}

type NotificationParams struct {
	Tokens   []domain.FCMToken
	TaskID   domain.TaskID
	TaskType domain.Type
}

func (r *NotificationRequest) ToDomain() (*NotificationParams, error) {
	tokens, err := domain.NewFCMTokens(r.Tokens)
	if err != nil {
		return nil, err
	}

	taskID, err := domain.NewTaskID(r.TaskID)
	if err != nil {
		return nil, err
	}

	taskType, err := domain.NewType(r.TaskType)
	if err != nil {
		return nil, err
	}

	return &NotificationParams{
		Tokens:   tokens,
		TaskID:   taskID,
		TaskType: taskType,
	}, nil
}

type NotificationResponse struct {
	Success      bool          `json:"success"`
	Total        int           `json:"total"`
	SuccessCount int           `json:"success_count"`
	FailureCount int           `json:"failure_count"`
	Results      []TokenResult `json:"results"`
}

type TokenResult struct {
	Token     string `json:"token"`
	Success   bool   `json:"success"`
	MessageID string `json:"message_id,omitempty"`
	Error     string `json:"error,omitempty"`
}

type ErrorResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
}

type HealthResponse struct {
	Status string `json:"status"`
}
