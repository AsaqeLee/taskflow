package repository

import (
	"context"
	"errors"

	"github.com/AsaqeLee/taskflow/internal/model"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type MongoUserRepository struct {
	collection *mongo.Collection
}

type userDocument struct {
	ID    string `bson:"_id"`
	Name  string `bson:"name"`
	Role  string `bson:"role"`
	Token string `bson:"token"`
}

func NewMongoUserRepository(collection *mongo.Collection) *MongoUserRepository {
	return &MongoUserRepository{collection: collection}
}

func (r *MongoUserRepository) Create(user model.User) (model.User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), taskOperationTimeout)
	defer cancel()

	if user.ID == "" {
		user.ID = bson.NewObjectID().Hex()
	}

	_, err := r.collection.InsertOne(ctx, userToDocument(user))
	if err != nil {
		return model.User{}, err
	}

	return user, nil
}

func (r *MongoUserRepository) FindByID(id string) (model.User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), taskOperationTimeout)
	defer cancel()

	var doc userDocument
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&doc)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return model.User{}, errors.New("user not found")
		}
		return model.User{}, err
	}

	return userDocumentToModel(doc), nil
}

func (r *MongoUserRepository) FindByToken(token string) (model.User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), taskOperationTimeout)
	defer cancel()

	var doc userDocument
	err := r.collection.FindOne(ctx, bson.M{"token": token}).Decode(&doc)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return model.User{}, errors.New("user not found by token")
		}
		return model.User{}, err
	}

	return userDocumentToModel(doc), nil
}

func userToDocument(user model.User) userDocument {
	return userDocument{
		ID:    user.ID,
		Name:  user.Name,
		Role:  user.Role,
		Token: user.Token,
	}
}

func userDocumentToModel(doc userDocument) model.User {
	return model.User{
		ID:    doc.ID,
		Name:  doc.Name,
		Role:  doc.Role,
		Token: doc.Token,
	}
}
