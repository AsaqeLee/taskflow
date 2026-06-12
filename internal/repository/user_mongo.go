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
	ID           string    `bson:"_id"`
	Name         string    `bson:"name"`
	Role         string    `bson:"role"`
	PasswordHash string    `bson:"password_hash"`
	Token        string    `bson:"token,omitempty"`
	CreatedAt    time.Time `bson:"created_at"`
	UpdatedAt    time.Time `bson:"updated_at"`
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

func userToDocument(user model.User) userDocument {
	return userDocument{
		ID:           user.ID,
		Name:         user.Name,
		Role:         user.Role,
		PasswordHash: user.PasswordHash,
		Token:        user.Token,
		CreatedAt:    user.CreatedAt,
		UpdatedAt:    user.UpdatedAt,
	}
}

func userDocumentToModel(doc userDocument) model.User {
	return model.User{
		ID:           doc.ID,
		Name:         doc.Name,
		Role:         doc.Role,
		PasswordHash: doc.PasswordHash,
		Token:        doc.Token,
		CreatedAt:    doc.CreatedAt,
		UpdatedAt:    doc.UpdatedAt,
	}
}
