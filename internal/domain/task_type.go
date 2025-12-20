package domain

import "fmt"

type Type string

const (
	TypeShort     Type = "short"
	TypeNear      Type = "near"
	TypeRelaxed   Type = "relaxed"
	TypeScheduled Type = "scheduled"
)

func NewType(t string) (Type, error) {
	switch t {
	case string(TypeShort), string(TypeNear), string(TypeRelaxed), string(TypeScheduled):
		return Type(t), nil
	default:
		return "", fmt.Errorf("%w: %s", ErrInvalidTaskType, t)
	}
}

func (t Type) String() string {
	return string(t)
}
