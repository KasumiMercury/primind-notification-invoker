package handler

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"

	"github.com/KasumiMercury/primind-notification-invoker/internal/domain"
	"github.com/KasumiMercury/primind-notification-invoker/internal/fcm"
	notifyv1 "github.com/KasumiMercury/primind-notification-invoker/internal/gen/notify/v1"
	"github.com/KasumiMercury/primind-notification-invoker/internal/model"
	pjson "github.com/KasumiMercury/primind-notification-invoker/internal/proto"
)

type NotificationHandler struct {
	fcmClient *fcm.Client
}

func NewNotificationHandler(client *fcm.Client) *NotificationHandler {
	return &NotificationHandler{fcmClient: client}
}

func (h *NotificationHandler) SendNotification(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		slog.Warn("method not allowed", "method", r.Method, "path", r.URL.Path)
		respondProtoError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		slog.Error("failed to read request body", "error", err)
		respondProtoError(w, http.StatusBadRequest, "failed to read request body")
		return
	}

	var req notifyv1.NotificationRequest
	if err := pjson.Unmarshal(body, &req); err != nil {
		slog.Error("failed to decode request body", "error", err)
		respondProtoError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	if err := pjson.Validate(&req); err != nil {
		slog.Error("validation error", "error", err)
		respondProtoError(w, http.StatusBadRequest, "validation error: "+err.Error())
		return
	}

	taskType, err := domain.ProtoTaskTypeToDomain(req.TaskType)
	if err != nil {
		slog.Error("invalid task type", "error", err)
		respondProtoError(w, http.StatusBadRequest, err.Error())
		return
	}

	modelReq := model.NotificationRequest{
		Tokens: req.Tokens,
		TaskID: req.TaskId,
	}

	params, err := modelReq.ToDomain(taskType)
	if err != nil {
		slog.Error("invalid request parameters", "error", err)
		respondProtoError(w, http.StatusBadRequest, err.Error())
		return
	}

	slog.Info("sending notification",
		"task_id", params.TaskID.String(),
		"task_type", params.TaskType.String(),
		"token_count", len(params.Tokens),
	)

	result, err := h.fcmClient.SendBulkNotification(r.Context(), params.Tokens, params.TaskID, params.TaskType)
	if err != nil {
		slog.Error("FCM bulk notification failed", "error", err)
		respondProtoError(w, http.StatusInternalServerError, "FCM error: "+err.Error())
		return
	}

	slog.Info("notification sent",
		"total", result.Total,
		"success_count", result.SuccessCount,
		"failure_count", result.FailureCount,
	)

	protoResults := make([]*notifyv1.TokenResult, len(result.Results))
	for i, r := range result.Results {
		protoResults[i] = &notifyv1.TokenResult{
			Token:     r.Token,
			Success:   r.Success,
			MessageId: r.MessageID,
			Error:     r.Error,
		}
	}

	resp := &notifyv1.NotificationResponse{
		Success:      true,
		Total:        int32(result.Total),
		SuccessCount: int32(result.SuccessCount),
		FailureCount: int32(result.FailureCount),
		Results:      protoResults,
	}

	respBytes, err := pjson.Marshal(resp)
	if err != nil {
		slog.Error("failed to marshal response", "error", err)
		respondProtoError(w, http.StatusInternalServerError, "failed to marshal response")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(respBytes)
}

func Health(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondJSON(w, http.StatusMethodNotAllowed, model.ErrorResponse{
			Success: false,
			Error:   "method not allowed",
		})
		return
	}

	respondJSON(w, http.StatusOK, model.HealthResponse{
		Status: "ok",
	})
}

func respondJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func respondProtoError(w http.ResponseWriter, status int, message string) {
	resp := &notifyv1.ErrorResponse{
		Success: false,
		Error:   message,
	}
	respBytes, _ := pjson.Marshal(resp)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(respBytes)
}
