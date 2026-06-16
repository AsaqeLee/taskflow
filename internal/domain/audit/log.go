package audit

import "time"

// Log captures a system-level task lifecycle event.
type Log struct {
	id             string
	taskID         string
	actorID        string
	action         Action
	requestID      string
	traceID        string
	idempotencyKey string
	sourceIP       string
	userAgent      string
	fromStatus     string
	toStatus       string
	createdAt      time.Time
}

type Metadata struct {
	RequestID      string
	TraceID        string
	IdempotencyKey string
	SourceIP       string
	UserAgent      string
}

type Draft struct {
	TaskID     string
	ActorID    string
	Action     Action
	FromStatus string
	ToStatus   string
	Metadata   Metadata
	CreatedAt  time.Time
}

func NewDraft(taskID, actorID string, action Action, fromStatus, toStatus string, meta Metadata, at time.Time) Draft {
	return Draft{
		TaskID:     taskID,
		ActorID:    actorID,
		Action:     action,
		FromStatus: fromStatus,
		ToStatus:   toStatus,
		Metadata:   meta,
		CreatedAt:  at,
	}
}

func Restore(
	id, taskID, actorID string,
	action Action,
	requestID, traceID, idempotencyKey, sourceIP, userAgent, fromStatus, toStatus string,
	createdAt time.Time,
) Log {
	return Log{
		id:             id,
		taskID:         taskID,
		actorID:        actorID,
		action:         action,
		requestID:      requestID,
		traceID:        traceID,
		idempotencyKey: idempotencyKey,
		sourceIP:       sourceIP,
		userAgent:      userAgent,
		fromStatus:     fromStatus,
		toStatus:       toStatus,
		createdAt:      createdAt,
	}
}

func (l Log) AssignID(id string) Log {
	l.id = id
	return l
}

func (l Log) ID() string             { return l.id }
func (l Log) TaskID() string         { return l.taskID }
func (l Log) ActorID() string        { return l.actorID }
func (l Log) Action() Action         { return l.action }
func (l Log) RequestID() string      { return l.requestID }
func (l Log) TraceID() string        { return l.traceID }
func (l Log) IdempotencyKey() string { return l.idempotencyKey }
func (l Log) SourceIP() string       { return l.sourceIP }
func (l Log) UserAgent() string      { return l.userAgent }
func (l Log) FromStatus() string     { return l.fromStatus }
func (l Log) ToStatus() string       { return l.toStatus }
func (l Log) CreatedAt() time.Time   { return l.createdAt }

func (d Draft) ToLog() Log {
	return Restore("", d.TaskID, d.ActorID, d.Action, d.Metadata.RequestID, d.Metadata.TraceID,
		d.Metadata.IdempotencyKey, d.Metadata.SourceIP, d.Metadata.UserAgent, d.FromStatus, d.ToStatus, d.CreatedAt)
}
