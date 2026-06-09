package repository

import (
	"context"
	"sort"
	"time"

	"github.com/AsaqeLee/taskflow/internal/model"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

const taskRecordCollectionName = "task_records"

type MongoTaskRecordRepository struct {
	collection *mongo.Collection
}

type taskRecordDocument struct {
	ID        string    `bson:"_id"`
	TaskID    string    `bson:"task_id"`
	AuthorID  string    `bson:"author_id"`
	Type      string    `bson:"type"`
	Content   string    `bson:"content"`
	CreatedAt time.Time `bson:"created_at"`
}

func NewMongoTaskRecordRepository(collection *mongo.Collection) *MongoTaskRecordRepository {
	return &MongoTaskRecordRepository{collection: collection}
}

func (r *MongoTaskRecordRepository) Create(ctx context.Context, record model.TaskRecord) (model.TaskRecord, error) {
	ctx, cancel := context.WithTimeout(ctx, taskOperationTimeout)
	defer cancel()

	if record.ID == "" {
		record.ID = bson.NewObjectID().Hex()
	}
	if record.CreatedAt.IsZero() {
		record.CreatedAt = time.Now().UTC()
	}

	_, err := r.collection.InsertOne(ctx, taskRecordToDocument(record))
	if err != nil {
		return model.TaskRecord{}, err
	}

	return record, nil
}

func (r *MongoTaskRecordRepository) ListByTaskID(ctx context.Context, taskID string) ([]model.TaskRecord, error) {
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

	result := make([]model.TaskRecord, 0, len(docs))
	for _, doc := range docs {
		result = append(result, taskRecordDocumentToModel(doc))
	}

	sort.SliceStable(result, func(i, j int) bool {
		if result[i].CreatedAt.Equal(result[j].CreatedAt) {
			return result[i].ID < result[j].ID
		}
		return result[i].CreatedAt.Before(result[j].CreatedAt)
	})

	return result, nil
}

func taskRecordToDocument(record model.TaskRecord) taskRecordDocument {
	return taskRecordDocument{
		ID:        record.ID,
		TaskID:    record.TaskID,
		AuthorID:  record.AuthorID,
		Type:      record.Type,
		Content:   record.Content,
		CreatedAt: record.CreatedAt,
	}
}

func taskRecordDocumentToModel(doc taskRecordDocument) model.TaskRecord {
	return model.TaskRecord{
		ID:        doc.ID,
		TaskID:    doc.TaskID,
		AuthorID:  doc.AuthorID,
		Type:      doc.Type,
		Content:   doc.Content,
		CreatedAt: doc.CreatedAt,
	}
}

func (r *MongoTaskRecordRepository) DeleteByTaskID(ctx context.Context, taskID string) error {
	ctx, cancel := context.WithTimeout(ctx, taskOperationTimeout)
	defer cancel()

	_, err := r.collection.DeleteMany(ctx, bson.M{"task_id": taskID})
	return err
}
