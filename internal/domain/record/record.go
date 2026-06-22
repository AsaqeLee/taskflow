package record

import (
	"strings"
	"time"

	"github.com/AsaqeLee/taskflow/internal/domain"
)

// Record stores collaboration text produced during a task transition.
type Record struct {
	id        string
	taskID    string
	authorID  string
	typ       Type
	content   string
	metadata  map[string]string
	createdAt time.Time
}

type Draft struct {
	TaskID    string
	AuthorID  string
	Type      Type
	Content   string
	Metadata  map[string]string
	CreatedAt time.Time
}

func NewDraft(authorID string, typ Type, content string, at time.Time) (Draft, error) {
	content = strings.TrimSpace(content)
	if content == "" {
		return Draft{}, domain.ErrEmptyTaskRecordContent
	}
	return Draft{
		AuthorID:  authorID,
		Type:      typ,
		Content:   content,
		CreatedAt: at,
	}, nil
}

func Restore(id, taskID, authorID string, typ Type, content string, createdAt time.Time) Record {
	return RestoreWithMetadata(id, taskID, authorID, typ, content, nil, createdAt)
}

func RestoreWithMetadata(
	id,
	taskID,
	authorID string,
	typ Type,
	content string,
	metadata map[string]string,
	createdAt time.Time,
) Record {
	return Record{
		id:        id,
		taskID:    taskID,
		authorID:  authorID,
		typ:       typ,
		content:   content,
		metadata:  sanitizeMetadata(metadata),
		createdAt: createdAt,
	}
}

func (r Record) AssignID(id string) Record {
	r.id = id
	return r
}

func (r Record) AssignTaskID(taskID string) Record {
	r.taskID = taskID
	return r
}

func (d Draft) AssignTaskID(taskID string) Draft {
	d.TaskID = taskID
	return d
}

func (d Draft) WithMetadata(metadata map[string]string) Draft {
	d.Metadata = sanitizeMetadata(metadata)
	return d
}

func (d Draft) ToRecord() Record {
	return RestoreWithMetadata("", d.TaskID, d.AuthorID, d.Type, d.Content, d.Metadata, d.CreatedAt)
}

func (r Record) ID() string       { return r.id }
func (r Record) TaskID() string   { return r.taskID }
func (r Record) AuthorID() string { return r.authorID }
func (r Record) Type() Type       { return r.typ }
func (r Record) Content() string  { return r.content }
func (r Record) Metadata() map[string]string {
	return sanitizeMetadata(r.metadata)
}
func (r Record) CreatedAt() time.Time { return r.createdAt }

func sanitizeMetadata(metadata map[string]string) map[string]string {
	if len(metadata) == 0 {
		return nil
	}

	sanitized := make(map[string]string, len(metadata))
	for key, value := range metadata {
		trimmedKey := strings.TrimSpace(key)
		trimmedValue := strings.TrimSpace(value)
		if trimmedKey == "" || trimmedValue == "" {
			continue
		}
		sanitized[trimmedKey] = trimmedValue
	}
	if len(sanitized) == 0 {
		return nil
	}
	return sanitized
}
