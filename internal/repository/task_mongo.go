package repository

import (
	"context"
	"sort"
	"time"

	domaintask "github.com/AsaqeLee/taskflow/internal/domain/task"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

const taskCollectionName = "tasks"
const taskOperationTimeout = 5 * time.Second

type MongoTaskRepository struct {
	collection *mongo.Collection
}

type taskDocument struct {
	ID          string     `bson:"_id"`
	Title       string     `bson:"title"`
	Description string     `bson:"description"`
	Status      string     `bson:"status"`
	CreatorID   string     `bson:"creator_id"`
	AssigneeID  string     `bson:"assignee_id"`
	CreatedAt   time.Time  `bson:"created_at"`
	UpdatedAt   time.Time  `bson:"updated_at"`
	DeletedAt   *time.Time `bson:"deleted_at,omitempty"`
	DeletedBy   string     `bson:"deleted_by,omitempty"`
}

func NewMongoTaskRepository(collection *mongo.Collection) *MongoTaskRepository {
	return &MongoTaskRepository{collection: collection}
}

func (r *MongoTaskRepository) Create(ctx context.Context, task domaintask.Task) (domaintask.Task, error) {
	ctx, cancel := context.WithTimeout(ctx, taskOperationTimeout)
	defer cancel()

	if task.ID() == "" {
		task = task.AssignID(bson.NewObjectID().Hex())
	}

	_, err := r.collection.InsertOne(ctx, taskToDocument(task))
	if err != nil {
		return domaintask.Task{}, err
	}

	return task, nil
}

func (r *MongoTaskRepository) GetByID(ctx context.Context, id string) (domaintask.Task, error) {
	ctx, cancel := context.WithTimeout(ctx, taskOperationTimeout)
	defer cancel()

	var doc taskDocument
	err := r.collection.FindOne(ctx, bson.M{"_id": id, "$or": []bson.M{{"deleted_at": bson.M{"$exists": false}}, {"deleted_at": nil}}}).Decode(&doc)
	if err != nil {
		return domaintask.Task{}, taskDocumentError(err)
	}

	return documentToTask(doc)
}

func (r *MongoTaskRepository) GetByIDIncludingDeleted(ctx context.Context, id string) (domaintask.Task, error) {
	ctx, cancel := context.WithTimeout(ctx, taskOperationTimeout)
	defer cancel()

	var doc taskDocument
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&doc)
	if err != nil {
		return domaintask.Task{}, taskDocumentError(err)
	}

	return documentToTask(doc)
}

func (r *MongoTaskRepository) List(ctx context.Context) ([]domaintask.Task, error) {
	ctx, cancel := context.WithTimeout(ctx, taskOperationTimeout)
	defer cancel()

	cursor, err := r.collection.Find(ctx, bson.M{"$or": []bson.M{{"deleted_at": bson.M{"$exists": false}}, {"deleted_at": nil}}})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var docs []taskDocument
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, err
	}

	result := make([]domaintask.Task, 0, len(docs))
	for _, doc := range docs {
		task, err := documentToTask(doc)
		if err != nil {
			return nil, err
		}
		result = append(result, task)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt().Before(result[j].CreatedAt())
	})

	return result, nil
}

func (r *MongoTaskRepository) ListVisibleToUser(ctx context.Context, userID string) ([]domaintask.Task, error) {
	ctx, cancel := context.WithTimeout(ctx, taskOperationTimeout)
	defer cancel()

	filter := bson.M{
		"$and": []bson.M{
			{
				"$or": []bson.M{
					{"deleted_at": bson.M{"$exists": false}},
					{"deleted_at": nil},
				},
			},
			{
				"$or": []bson.M{
					{"creator_id": userID},
					{"assignee_id": userID},
				},
			},
		},
	}

	cursor, err := r.collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var docs []taskDocument
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, err
	}

	result := make([]domaintask.Task, 0, len(docs))
	for _, doc := range docs {
		task, err := documentToTask(doc)
		if err != nil {
			return nil, err
		}
		result = append(result, task)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt().Before(result[j].CreatedAt())
	})

	return result, nil
}

func (r *MongoTaskRepository) Update(ctx context.Context, task domaintask.Task) (domaintask.Task, error) {
	ctx, cancel := context.WithTimeout(ctx, taskOperationTimeout)
	defer cancel()

	update := bson.M{"$set": bson.M{
		"title":       task.Title(),
		"description": task.Description(),
		"status":      task.Status().String(),
		"creator_id":  task.CreatorID(),
		"assignee_id": task.AssigneeID(),
		"created_at":  task.CreatedAt(),
		"updated_at":  task.UpdatedAt(),
		"deleted_at":  task.DeletedAt(),
		"deleted_by":  task.DeletedBy(),
	}}

	result, err := r.collection.UpdateOne(ctx, bson.M{"_id": task.ID()}, update)
	if err != nil {
		return domaintask.Task{}, err
	}
	if result.MatchedCount == 0 {
		return domaintask.Task{}, ErrTaskNotFound
	}

	return task, nil
}

func taskToDocument(task domaintask.Task) taskDocument {
	return taskDocument{
		ID:          task.ID(),
		Title:       task.Title(),
		Description: task.Description(),
		Status:      task.Status().String(),
		CreatorID:   task.CreatorID(),
		AssigneeID:  task.AssigneeID(),
		CreatedAt:   task.CreatedAt(),
		UpdatedAt:   task.UpdatedAt(),
		DeletedAt:   task.DeletedAt(),
		DeletedBy:   task.DeletedBy(),
	}
}

func documentToTask(doc taskDocument) (domaintask.Task, error) {
	status, err := domaintask.ParseStatus(doc.Status)
	if err != nil {
		return domaintask.Task{}, err
	}
	return domaintask.Restore(
		doc.ID,
		doc.Title,
		doc.Description,
		status,
		doc.CreatorID,
		doc.AssigneeID,
		doc.CreatedAt,
		doc.UpdatedAt,
		doc.DeletedAt,
		doc.DeletedBy,
	), nil
}

func taskDocumentError(err error) error {
	if err == mongo.ErrNoDocuments {
		return ErrTaskNotFound
	}
	return err
}
