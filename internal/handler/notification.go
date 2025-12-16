package handler

import (
	"encoding/json"
	"net/http"

	"github.com/KasumiMercury/primind-notification-invoker/internal/fcm"
	"github.com/KasumiMercury/primind-notification-invoker/internal/model"
)

type NotificationHandler struct {
	fcmClient *fcm.Client
}

func NewNotificationHandler(client *fcm.Client) *NotificationHandler {
	return &NotificationHandler{fcmClient: client}
}

func (h *NotificationHandler) SendNotification(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondJSON(w, http.StatusMethodNotAllowed, model.ErrorResponse{
			Success: false,
			Error:   "method not allowed",
		})
		return
	}

	var req model.NotificationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, model.ErrorResponse{
			Success: false,
			Error:   "invalid JSON: " + err.Error(),
		})
		return
	}

	params, err := req.ToDomain()
	if err != nil {
		respondJSON(w, http.StatusBadRequest, model.ErrorResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	result, err := h.fcmClient.SendBulkNotification(r.Context(), params.Tokens, params.TaskID, params.TaskType)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, model.ErrorResponse{
			Success: false,
			Error:   "FCM error: " + err.Error(),
		})
		return
	}

	respondJSON(w, http.StatusOK, model.NotificationResponse{
		Success:      true,
		Total:        result.Total,
		SuccessCount: result.SuccessCount,
		FailureCount: result.FailureCount,
		Results:      result.Results,
	})
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
