package repository

import (
	"context"
	"time"

	"github.com/AsaqeLee/taskflow/internal/model"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type MongoUserRepository struct {
	collection *mongo.Collection
}

type userDocument struct {
	ID           string     `bson:"_id"`
	Name         string     `bson:"name"`
	Role         string     `bson:"role"`
	PasswordHash string     `bson:"password_hash"`
	Token        string     `bson:"token,omitempty"`
	Active       bool       `bson:"active"`
	DisabledAt   *time.Time `bson:"disabled_at,omitempty"`
	DisabledBy   string     `bson:"disabled_by,omitempty"`
	CreatedAt    time.Time  `bson:"created_at"`
	UpdatedAt    time.Time  `bson:"updated_at"`
}

func NewMongoUserRepository(collection *mongo.Collection) *MongoUserRepository {
	return &MongoUserRepository{collection: collection}
}

func (r *MongoUserRepository) Create(ctx context.Context, user model.User) (model.User, error) {
	ctx, cancel := context.WithTimeout(ctx, taskOperationTimeout)
	defer cancel()

	if user.ID == "" {
		user.ID = bson.NewObjectID().Hex()
	}
	if user.CreatedAt.IsZero() {
		now := time.Now().UTC()
		user.CreatedAt = now
		user.UpdatedAt = now
	}
	if !user.Active && user.DisabledAt == nil {
		user.Active = true
	}

	_, err := r.collection.InsertOne(ctx, userToDocument(user))
	if err != nil {
		return model.User{}, ErrUserAlreadyExists
	}

	return user, nil
}

func (r *MongoUserRepository) FindByID(ctx context.Context, id string) (model.User, error) {
	ctx, cancel := context.WithTimeout(ctx, taskOperationTimeout)
	defer cancel()

	var doc userDocument
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&doc)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return model.User{}, ErrUserNotFound
		}
		return model.User{}, err
	}

	return userDocumentToModel(doc), nil
}

func (r *MongoUserRepository) FindByToken(ctx context.Context, token string) (model.User, error) {
	ctx, cancel := context.WithTimeout(ctx, taskOperationTimeout)
	defer cancel()

	var doc userDocument
	err := r.collection.FindOne(ctx, bson.M{"token": token}).Decode(&doc)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return model.User{}, ErrUserNotFoundByToken
		}
		return model.User{}, err
	}

	return userDocumentToModel(doc), nil
}

func (r *MongoUserRepository) UpdatePassword(ctx context.Context, id, passwordHash string, updatedAt time.Time) (model.User, error) {
	ctx, cancel := context.WithTimeout(ctx, taskOperationTimeout)
	defer cancel()

	result, err := r.collection.UpdateOne(ctx, bson.M{"_id": id}, bson.M{
		"$set": bson.M{
			"password_hash": passwordHash,
			"updated_at":    updatedAt,
		},
	})
	if err != nil {
		return model.User{}, err
	}
	if result.MatchedCount == 0 {
		return model.User{}, ErrUserNotFound
	}

	return r.FindByID(ctx, id)
}

func (r *MongoUserRepository) Disable(ctx context.Context, id, disabledBy string, disabledAt time.Time) (model.User, error) {
	ctx, cancel := context.WithTimeout(ctx, taskOperationTimeout)
	defer cancel()

	result, err := r.collection.UpdateOne(ctx, bson.M{"_id": id}, bson.M{
		"$set": bson.M{
			"active":      false,
			"disabled_at": disabledAt,
			"disabled_by": disabledBy,
			"updated_at":  disabledAt,
		},
	})
	if err != nil {
		return model.User{}, err
	}
	if result.MatchedCount == 0 {
		return model.User{}, ErrUserNotFound
	}

	return r.FindByID(ctx, id)
}

func userToDocument(user model.User) userDocument {
	return userDocument{
		ID:           user.ID,
		Name:         user.Name,
		Role:         user.Role,
		PasswordHash: user.PasswordHash,
		Token:        user.Token,
		Active:       user.Active,
		DisabledAt:   user.DisabledAt,
		DisabledBy:   user.DisabledBy,
		CreatedAt:    user.CreatedAt,
		UpdatedAt:    user.UpdatedAt,
	}
}

func userDocumentToModel(doc userDocument) model.User {
	active := doc.Active
	if !active && doc.DisabledAt == nil {
		active = true
	}

	return model.User{
		ID:           doc.ID,
		Name:         doc.Name,
		Role:         doc.Role,
		PasswordHash: doc.PasswordHash,
		Token:        doc.Token,
		Active:       active,
		DisabledAt:   doc.DisabledAt,
		DisabledBy:   doc.DisabledBy,
		CreatedAt:    doc.CreatedAt,
		UpdatedAt:    doc.UpdatedAt,
	}
}
