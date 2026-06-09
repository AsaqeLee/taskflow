package repository

import (
	"context"
	"sort"
	"time"

	"github.com/AsaqeLee/taskflow/internal/model"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

const taskCollectionName = "tasks"
const taskOperationTimeout = 5 * time.Second

type MongoTaskRepository struct {
	collection *mongo.Collection
}

type taskDocument struct {
	ID          string    `bson:"_id"`
	Title       string    `bson:"title"`
	Description string    `bson:"description"`
	Status      string    `bson:"status"`
	CreatorID   string    `bson:"creator_id"`
	AssigneeID  string    `bson:"assignee_id"`
	CreatedAt   time.Time `bson:"created_at"`
	UpdatedAt   time.Time `bson:"updated_at"`
}

func NewMongoTaskRepository(collection *mongo.Collection) *MongoTaskRepository {
	return &MongoTaskRepository{collection: collection}
}

func (r *MongoTaskRepository) Create(ctx context.Context, task model.Task) (model.Task, error) {
	ctx, cancel := context.WithTimeout(ctx, taskOperationTimeout)
	defer cancel()

	if task.ID == "" {
		task.ID = bson.NewObjectID().Hex()
	}

	_, err := r.collection.InsertOne(ctx, taskToDocument(task))
	if err != nil {
		return model.Task{}, err
	}

	return task, nil
}

func (r *MongoTaskRepository) GetByID(ctx context.Context, id string) (model.Task, error) {
	ctx, cancel := context.WithTimeout(ctx, taskOperationTimeout)
	defer cancel()

	var doc taskDocument
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&doc)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return model.Task{}, ErrTaskNotFound
		}
		return model.Task{}, err
	}

	return documentToTask(doc), nil
}

func (r *MongoTaskRepository) List(ctx context.Context) ([]model.Task, error) {
	ctx, cancel := context.WithTimeout(ctx, taskOperationTimeout)
	defer cancel()

	cursor, err := r.collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var docs []taskDocument
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, err
	}

	result := make([]model.Task, 0, len(docs))
	for _, doc := range docs {
		result = append(result, documentToTask(doc))
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt.Before(result[j].CreatedAt)
	})

	return result, nil
}

func (r *MongoTaskRepository) Update(ctx context.Context, task model.Task) (model.Task, error) {
	ctx, cancel := context.WithTimeout(ctx, taskOperationTimeout)
	defer cancel()

	update := bson.M{"$set": bson.M{
		"title":       task.Title,
		"description": task.Description,
		"status":      task.Status,
		"creator_id":  task.CreatorID,
		"assignee_id": task.AssigneeID,
		"created_at":  task.CreatedAt,
		"updated_at":  task.UpdatedAt,
	}}

	result, err := r.collection.UpdateOne(ctx, bson.M{"_id": task.ID}, update)
	if err != nil {
		return model.Task{}, err
	}
	if result.MatchedCount == 0 {
		return model.Task{}, ErrTaskNotFound
	}

	return task, nil
}

func taskToDocument(task model.Task) taskDocument {
	return taskDocument{
		ID:          task.ID,
		Title:       task.Title,
		Description: task.Description,
		Status:      task.Status,
		CreatorID:   task.CreatorID,
		AssigneeID:  task.AssigneeID,
		CreatedAt:   task.CreatedAt,
		UpdatedAt:   task.UpdatedAt,
	}
}

func documentToTask(doc taskDocument) model.Task {
	return model.Task{
		ID:          doc.ID,
		Title:       doc.Title,
		Description: doc.Description,
		Status:      doc.Status,
		CreatorID:   doc.CreatorID,
		AssigneeID:  doc.AssigneeID,
		CreatedAt:   doc.CreatedAt,
		UpdatedAt:   doc.UpdatedAt,
	}
}

func (r *MongoTaskRepository) Delete(ctx context.Context, id string) error {
	ctx, cancel := context.WithTimeout(ctx, taskOperationTimeout)
	defer cancel()

	result, err := r.collection.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return err
	}
	if result.DeletedCount == 0 {
		return ErrTaskNotFound
	}

	return nil
}
