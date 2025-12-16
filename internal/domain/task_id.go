package domain

import "fmt"

type TaskID string

func NewTaskID(id string) (TaskID, error) {
	if id == "" {
		return "", fmt.Errorf("%w: empty task id", ErrInvalidTaskID)
	}
	return TaskID(id), nil
}

func (t TaskID) String() string {
	return string(t)
}
