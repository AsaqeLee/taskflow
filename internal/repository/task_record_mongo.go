package repository

import (
	"context"
	"sort"
	"time"

	domainrecord "github.com/AsaqeLee/taskflow/internal/domain/record"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

const taskRecordCollectionName = "task_records"

type MongoTaskRecordRepository struct {
	collection *mongo.Collection
}

type taskRecordDocument struct {
	ID        string            `bson:"_id"`
	TaskID    string            `bson:"task_id"`
	AuthorID  string            `bson:"author_id"`
	Type      string            `bson:"type"`
	Content   string            `bson:"content"`
	Metadata  map[string]string `bson:"metadata,omitempty"`
	CreatedAt time.Time         `bson:"created_at"`
}

func NewMongoTaskRecordRepository(collection *mongo.Collection) *MongoTaskRecordRepository {
	return &MongoTaskRecordRepository{collection: collection}
}

func (r *MongoTaskRecordRepository) Create(ctx context.Context, record domainrecord.Record) (domainrecord.Record, error) {
	ctx, cancel := context.WithTimeout(ctx, taskOperationTimeout)
	defer cancel()

	if record.ID() == "" {
		record = record.AssignID(bson.NewObjectID().Hex())
	}
	createdAt := record.CreatedAt()
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
		record = domainrecord.RestoreWithMetadata(
			record.ID(),
			record.TaskID(),
			record.AuthorID(),
			record.Type(),
			record.Content(),
			record.Metadata(),
			createdAt,
		)
	}

	_, err := r.collection.InsertOne(ctx, taskRecordToDocument(record))
	if err != nil {
		return domainrecord.Record{}, err
	}

	return record, nil
}

func (r *MongoTaskRecordRepository) ListByTaskID(ctx context.Context, taskID string) ([]domainrecord.Record, error) {
	ctx, cancel := context.WithTimeout(ctx, taskOperationTimeout)
	defer cancel()

	cursor, err := r.collection.Find(ctx, bson.M{"task_id": taskID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var docs []taskRecordDocument
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, err
	}

	result := make([]domainrecord.Record, 0, len(docs))
	for _, doc := range docs {
		result = append(result, taskRecordDocumentToDomain(doc))
	}

	sort.SliceStable(result, func(i, j int) bool {
		if result[i].CreatedAt().Equal(result[j].CreatedAt()) {
			return result[i].ID() < result[j].ID()
		}
		return result[i].CreatedAt().Before(result[j].CreatedAt())
	})

	return result, nil
}

func taskRecordToDocument(record domainrecord.Record) taskRecordDocument {
	return taskRecordDocument{
		ID:        record.ID(),
		TaskID:    record.TaskID(),
		AuthorID:  record.AuthorID(),
		Type:      record.Type().String(),
		Content:   record.Content(),
		Metadata:  record.Metadata(),
		CreatedAt: record.CreatedAt(),
	}
}

func taskRecordDocumentToDomain(doc taskRecordDocument) domainrecord.Record {
	return domainrecord.RestoreWithMetadata(
		doc.ID,
		doc.TaskID,
		doc.AuthorID,
		domainrecord.Type(doc.Type),
		doc.Content,
		doc.Metadata,
		doc.CreatedAt,
	)
}

func (r *MongoTaskRecordRepository) DeleteByTaskID(ctx context.Context, taskID string) error {
	ctx, cancel := context.WithTimeout(ctx, taskOperationTimeout)
	defer cancel()

	_, err := r.collection.DeleteMany(ctx, bson.M{"task_id": taskID})
	return err
}
