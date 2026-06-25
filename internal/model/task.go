package model

import "time"

type Task struct {
	ID               string     `json:"id"`
	Title            string     `json:"title"`
	Description      string     `json:"description"`
	Status           string     `json:"status"`
	AvailableActions []string   `json:"available_actions,omitempty"`
	CreatorID        string     `json:"creator_id"`
	AssigneeID       string     `json:"assignee_id"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
	DeletedAt        *time.Time `json:"deleted_at,omitempty"`
	DeletedBy        string     `json:"deleted_by,omitempty"`
}
