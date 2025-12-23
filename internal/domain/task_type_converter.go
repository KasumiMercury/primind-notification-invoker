package domain

import (
	"fmt"

	commonv1 "github.com/KasumiMercury/primind-notification-invoker/internal/gen/common/v1"
)

func ProtoTaskTypeToDomain(pt commonv1.TaskType) (Type, error) {
	switch pt {
	case commonv1.TaskType_TASK_TYPE_SHORT:
		return TypeShort, nil
	case commonv1.TaskType_TASK_TYPE_NEAR:
		return TypeNear, nil
	case commonv1.TaskType_TASK_TYPE_RELAXED:
		return TypeRelaxed, nil
	case commonv1.TaskType_TASK_TYPE_SCHEDULED:
		return TypeScheduled, nil
	default:
		return "", fmt.Errorf("%w: %s", ErrInvalidTaskType, pt.String())
	}
}

func DomainTaskTypeToProto(dt Type) commonv1.TaskType {
	switch dt {
	case TypeShort:
		return commonv1.TaskType_TASK_TYPE_SHORT
	case TypeNear:
		return commonv1.TaskType_TASK_TYPE_NEAR
	case TypeRelaxed:
		return commonv1.TaskType_TASK_TYPE_RELAXED
	case TypeScheduled:
		return commonv1.TaskType_TASK_TYPE_SCHEDULED
	default:
		return commonv1.TaskType_TASK_TYPE_UNSPECIFIED
	}
}
