package domain

import "fmt"

type Type string

const (
	TypeUrgent    Type = "urgent"
	TypeNormal    Type = "normal"
	TypeLow       Type = "low"
	TypeScheduled Type = "scheduled"
)

func NewType(t string) (Type, error) {
	switch t {
	case string(TypeUrgent), string(TypeNormal), string(TypeLow), string(TypeScheduled):
		return Type(t), nil
	default:
		return "", fmt.Errorf("%w: %s", ErrInvalidTaskType, t)
	}
}

func (t Type) String() string {
	return string(t)
}
