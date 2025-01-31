package model

import (
	"time"

	"github.com/mattermost/mattermost-server/v6/utils"
)

// NotificationHint provides a hint that a block has been modified and has subscribers that
// should be notified.
// swagger:model
type NotificationHint struct {
	// BlockType is the block type of the entity (e.g. board, card) that was updated
	// required: true
	BlockType BlockType `json:"block_type"`

	// BlockID is id of the entity that was updated
	// required: true
	BlockID string `json:"block_id"`

	// WorkspaceID is id of workspace the block belongs to
	// required: true
	WorkspaceID string `json:"workspace_id"`

	// ModifiedByID is the id of the user who made the block change
	ModifiedByID string `json:"modified_by_id"`

	// CreatedAt is the timestamp this notification hint was created
	// required: true
	CreateAt int64 `json:"create_at"`

	// NotifyAt is the timestamp this notification should be scheduled
	// required: true
	NotifyAt int64 `json:"notify_at"`
}

func (s *NotificationHint) IsValid() error {
	if s == nil {
		return ErrInvalidNotificationHint{"cannot be nil"}
	}
	if s.BlockID == "" {
		return ErrInvalidNotificationHint{"missing block id"}
	}
	if s.WorkspaceID == "" {
		return ErrInvalidNotificationHint{"missing workspace id"}
	}
	if s.BlockType == "" {
		return ErrInvalidNotificationHint{"missing block type"}
	}
	if s.ModifiedByID == "" {
		return ErrInvalidNotificationHint{"missing modified_by id"}
	}
	return nil
}

func (s *NotificationHint) Copy() *NotificationHint {
	return &NotificationHint{
		BlockType:    s.BlockType,
		BlockID:      s.BlockID,
		WorkspaceID:  s.WorkspaceID,
		ModifiedByID: s.ModifiedByID,
		CreateAt:     s.CreateAt,
		NotifyAt:     s.NotifyAt,
	}
}

func (s *NotificationHint) LogClone() interface{} {
	return struct {
		BlockType    BlockType `json:"block_type"`
		BlockID      string    `json:"block_id"`
		WorkspaceID  string    `json:"workspace_id"`
		ModifiedByID string    `json:"modified_by_id"`
		CreateAt     string    `json:"create_at"`
		NotifyAt     string    `json:"notify_at"`
	}{
		BlockType:    s.BlockType,
		BlockID:      s.BlockID,
		WorkspaceID:  s.WorkspaceID,
		ModifiedByID: s.ModifiedByID,
		CreateAt:     utils.TimeFromMillis(s.CreateAt).Format(time.StampMilli),
		NotifyAt:     utils.TimeFromMillis(s.NotifyAt).Format(time.StampMilli),
	}
}

type ErrInvalidNotificationHint struct {
	msg string
}

func (e ErrInvalidNotificationHint) Error() string {
	return e.msg
}
