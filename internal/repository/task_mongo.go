package repository

import (
	"context"
	"regexp"
	"time"

	"github.com/AsaqeLee/taskflow/internal/domain/ports"
	domaintask "github.com/AsaqeLee/taskflow/internal/domain/task"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
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
	result, err := r.Search(ctx, ports.TaskListQuery{})
	if err != nil {
		return nil, err
	}
	return result.Tasks, nil
}

func (r *MongoTaskRepository) ListVisibleToUser(ctx context.Context, userID string) ([]domaintask.Task, error) {
	result, err := r.SearchVisibleToUser(ctx, userID, ports.TaskListQuery{})
	if err != nil {
		return nil, err
	}
	return result.Tasks, nil
}

func (r *MongoTaskRepository) Search(ctx context.Context, query ports.TaskListQuery) (ports.TaskListResult, error) {
	return r.search(ctx, "", query)
}

func (r *MongoTaskRepository) SearchVisibleToUser(ctx context.Context, userID string, query ports.TaskListQuery) (ports.TaskListResult, error) {
	return r.search(ctx, userID, query)
}

func (r *MongoTaskRepository) search(ctx context.Context, userID string, query ports.TaskListQuery) (ports.TaskListResult, error) {
	ctx, cancel := context.WithTimeout(ctx, taskOperationTimeout)
	defer cancel()

	filter := buildTaskSearchFilter(userID, query)
	total, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return ports.TaskListResult{}, err
	}

	findOptions := options.Find().SetSort(bson.D{{Key: "created_at", Value: 1}})
	if query.Offset > 0 {
		findOptions.SetSkip(int64(query.Offset))
	}
	if query.Limit > 0 {
		findOptions.SetLimit(int64(query.Limit))
	}

	cursor, err := r.collection.Find(ctx, filter, findOptions)
	if err != nil {
		return ports.TaskListResult{}, err
	}
	defer cursor.Close(ctx)

	var docs []taskDocument
	if err := cursor.All(ctx, &docs); err != nil {
		return ports.TaskListResult{}, err
	}

	result := make([]domaintask.Task, 0, len(docs))
	for _, doc := range docs {
		task, err := documentToTask(doc)
		if err != nil {
			return ports.TaskListResult{}, err
		}
		result = append(result, task)
	}

	return ports.TaskListResult{
		Tasks: result,
		Total: int(total),
	}, nil
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

func buildTaskSearchFilter(userID string, query ports.TaskListQuery) bson.M {
	filters := []bson.M{{
		"$or": []bson.M{
			{"deleted_at": bson.M{"$exists": false}},
			{"deleted_at": nil},
		},
	}}

	if userID != "" {
		filters = append(filters, bson.M{
			"$or": []bson.M{
				{"creator_id": userID},
				{"assignee_id": userID},
			},
		})
	}
	if query.Status != "" {
		filters = append(filters, bson.M{"status": query.Status})
	}
	if query.Query != "" {
		pattern := regexp.QuoteMeta(query.Query)
		filters = append(filters, bson.M{
			"$or": []bson.M{
				{"title": bson.M{"$regex": pattern, "$options": "i"}},
				{"description": bson.M{"$regex": pattern, "$options": "i"}},
			},
		})
	}

	if len(filters) == 1 {
		return filters[0]
	}
	return bson.M{"$and": filters}
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
