package service

import (
	"context"

	domainaudit "github.com/AsaqeLee/taskflow/internal/domain/audit"
	"github.com/AsaqeLee/taskflow/internal/domain/ports"
	domainrecord "github.com/AsaqeLee/taskflow/internal/domain/record"
	domaintask "github.com/AsaqeLee/taskflow/internal/domain/task"
	"github.com/AsaqeLee/taskflow/internal/requestmeta"
)

type taskEventApplier struct {
	recordRepo ports.TaskRecordRepository
	auditRepo  ports.AuditLogRepository
}

func newTaskEventApplier(recordRepo ports.TaskRecordRepository, auditRepo ports.AuditLogRepository) taskEventApplier {
	return taskEventApplier{recordRepo: recordRepo, auditRepo: auditRepo}
}

func (a taskEventApplier) apply(ctx context.Context, transition domaintask.Transition) (domainrecord.Record, bool, error) {
	var savedRecord domainrecord.Record
	var hasRecord bool

	for _, evt := range transition.Events {
		switch e := evt.(type) {
		case domaintask.StatusChangedEvent:
			if err := a.persistStatusChanged(ctx, e); err != nil {
				return domainrecord.Record{}, false, err
			}
		case domaintask.RecordDraftedEvent:
			record, err := a.persistRecordDraft(ctx, e)
			if err != nil {
				return domainrecord.Record{}, false, err
			}
			savedRecord = record
			hasRecord = true
		}
	}

	return savedRecord, hasRecord, nil
}

func (a taskEventApplier) persistStatusChanged(ctx context.Context, evt domaintask.StatusChangedEvent) error {
	meta := requestmeta.FromContext(ctx)
	draft := domainaudit.NewDraft(
		evt.TaskID,
		evt.ActorID,
		evt.AuditAction,
		evt.FromStatus.String(),
		evt.ToStatus.String(),
		domainaudit.Metadata{
			RequestID:      meta.RequestID,
			TraceID:        meta.TraceID,
			IdempotencyKey: meta.IdempotencyKey,
			SourceIP:       meta.SourceIP,
			UserAgent:      meta.UserAgent,
		},
		evt.OccurredAt(),
	)
	_, err := a.auditRepo.Create(ctx, draft.ToLog())
	return err
}

func (a taskEventApplier) persistRecordDraft(ctx context.Context, evt domaintask.RecordDraftedEvent) (domainrecord.Record, error) {
	return a.recordRepo.Create(ctx, evt.Draft.ToRecord())
}
