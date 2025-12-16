package domain

import "errors"

var (
	ErrInvalidTaskType = errors.New("invalid task type")
	ErrInvalidTaskID   = errors.New("invalid task id")
	ErrInvalidToken    = errors.New("invalid fcm token")
)
