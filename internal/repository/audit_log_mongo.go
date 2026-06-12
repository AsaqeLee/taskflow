package repository

import (
	"context"
	"sort"
	"time"

	"github.com/AsaqeLee/taskflow/internal/model"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type MongoAuditLogRepository struct {
	collection *mongo.Collection
}

type auditLogDocument struct {
	ID             string    `bson:"_id"`
	TaskID         string    `bson:"task_id"`
	ActorID        string    `bson:"actor_id"`
	Action         string    `bson:"action"`
	RequestID      string    `bson:"request_id,omitempty"`
	TraceID        string    `bson:"trace_id,omitempty"`
	IdempotencyKey string    `bson:"idempotency_key,omitempty"`
	SourceIP       string    `bson:"source_ip,omitempty"`
	UserAgent      string    `bson:"user_agent,omitempty"`
	FromStatus     string    `bson:"from_status,omitempty"`
	ToStatus       string    `bson:"to_status,omitempty"`
	CreatedAt      time.Time `bson:"created_at"`
}

func NewMongoAuditLogRepository(collection *mongo.Collection) *MongoAuditLogRepository {
	return &MongoAuditLogRepository{collection: collection}
}

func (r *MongoAuditLogRepository) Create(ctx context.Context, log model.AuditLog) (model.AuditLog, error) {
	ctx, cancel := context.WithTimeout(ctx, taskOperationTimeout)
	defer cancel()

	if log.ID == "" {
		log.ID = bson.NewObjectID().Hex()
	}
	if log.CreatedAt.IsZero() {
		log.CreatedAt = time.Now().UTC()
	}

	_, err := r.collection.InsertOne(ctx, auditLogToDocument(log))
	if err != nil {
		return model.AuditLog{}, err
	}

	return log, nil
}

func (r *MongoAuditLogRepository) ListByTaskID(ctx context.Context, taskID string) ([]model.AuditLog, error) {
	ctx, cancel := context.WithTimeout(ctx, taskOperationTimeout)
	defer cancel()

	cursor, err := r.collection.Find(ctx, bson.M{"task_id": taskID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var docs []auditLogDocument
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, err
	}

	result := make([]model.AuditLog, 0, len(docs))
	for _, doc := range docs {
		result = append(result, auditLogDocumentToModel(doc))
	}

	sort.SliceStable(result, func(i, j int) bool {
		if result[i].CreatedAt.Equal(result[j].CreatedAt) {
			return result[i].ID < result[j].ID
		}
		return result[i].CreatedAt.Before(result[j].CreatedAt)
	})

	return result, nil
}

func (r *MongoAuditLogRepository) DeleteByTaskID(ctx context.Context, taskID string) error {
	ctx, cancel := context.WithTimeout(ctx, taskOperationTimeout)
	defer cancel()

	_, err := r.collection.DeleteMany(ctx, bson.M{"task_id": taskID})
	return err
}

func auditLogToDocument(log model.AuditLog) auditLogDocument {
	return auditLogDocument{
		ID:             log.ID,
		TaskID:         log.TaskID,
		ActorID:        log.ActorID,
		Action:         log.Action,
		RequestID:      log.RequestID,
		TraceID:        log.TraceID,
		IdempotencyKey: log.IdempotencyKey,
		SourceIP:       log.SourceIP,
		UserAgent:      log.UserAgent,
		FromStatus:     log.FromStatus,
		ToStatus:       log.ToStatus,
		CreatedAt:      log.CreatedAt,
	}
}

func auditLogDocumentToModel(doc auditLogDocument) model.AuditLog {
	return model.AuditLog{
		ID:             doc.ID,
		TaskID:         doc.TaskID,
		ActorID:        doc.ActorID,
		Action:         doc.Action,
		RequestID:      doc.RequestID,
		TraceID:        doc.TraceID,
		IdempotencyKey: doc.IdempotencyKey,
		SourceIP:       doc.SourceIP,
		UserAgent:      doc.UserAgent,
		FromStatus:     doc.FromStatus,
		ToStatus:       doc.ToStatus,
		CreatedAt:      doc.CreatedAt,
	}
}
