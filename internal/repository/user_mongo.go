package repository

import (
	"context"
	"errors"
	"time"

	domainuser "github.com/AsaqeLee/taskflow/internal/domain/user"
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

func (r *MongoUserRepository) Create(ctx context.Context, account domainuser.Account) (domainuser.Account, error) {
	ctx, cancel := context.WithTimeout(ctx, taskOperationTimeout)
	defer cancel()

	if account.ID() == "" {
		account = account.AssignID(bson.NewObjectID().Hex())
	}
	now := time.Now().UTC()
	if account.CreatedAt().IsZero() {
		account = domainuser.Restore(
			account.ID(),
			account.Name(),
			account.Role(),
			account.PasswordHash(),
			account.LegacyToken(),
			true,
			nil,
			"",
			now,
			now,
		)
	}

	_, err := r.collection.InsertOne(ctx, accountToDocument(account))
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return domainuser.Account{}, ErrUserAlreadyExists
		}
		var writeErr mongo.WriteException
		if errors.As(err, &writeErr) && writeErr.HasErrorCode(11000) {
			return domainuser.Account{}, ErrUserAlreadyExists
		}
		return domainuser.Account{}, err
	}

	return account, nil
}

func (r *MongoUserRepository) FindByID(ctx context.Context, id string) (domainuser.Account, error) {
	ctx, cancel := context.WithTimeout(ctx, taskOperationTimeout)
	defer cancel()

	var doc userDocument
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&doc)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return domainuser.Account{}, ErrUserNotFound
		}
		return domainuser.Account{}, err
	}

	return userDocumentToAccount(doc), nil
}

func (r *MongoUserRepository) FindByToken(ctx context.Context, token string) (domainuser.Account, error) {
	ctx, cancel := context.WithTimeout(ctx, taskOperationTimeout)
	defer cancel()

	var doc userDocument
	err := r.collection.FindOne(ctx, bson.M{"token": token}).Decode(&doc)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return domainuser.Account{}, ErrUserNotFoundByToken
		}
		return domainuser.Account{}, err
	}

	return userDocumentToAccount(doc), nil
}

func (r *MongoUserRepository) UpdatePassword(ctx context.Context, id, passwordHash string, updatedAt time.Time) (domainuser.Account, error) {
	ctx, cancel := context.WithTimeout(ctx, taskOperationTimeout)
	defer cancel()

	result, err := r.collection.UpdateOne(ctx, bson.M{"_id": id}, bson.M{
		"$set": bson.M{
			"password_hash": passwordHash,
			"updated_at":    updatedAt,
		},
	})
	if err != nil {
		return domainuser.Account{}, err
	}
	if result.MatchedCount == 0 {
		return domainuser.Account{}, ErrUserNotFound
	}

	return r.FindByID(ctx, id)
}

func (r *MongoUserRepository) Disable(ctx context.Context, id, disabledBy string, disabledAt time.Time) (domainuser.Account, error) {
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
		return domainuser.Account{}, err
	}
	if result.MatchedCount == 0 {
		return domainuser.Account{}, ErrUserNotFound
	}

	return r.FindByID(ctx, id)
}

func (r *MongoUserRepository) Update(ctx context.Context, account domainuser.Account) (domainuser.Account, error) {
	ctx, cancel := context.WithTimeout(ctx, taskOperationTimeout)
	defer cancel()

	result, err := r.collection.UpdateOne(ctx, bson.M{"_id": account.ID()}, bson.M{
		"$set": bson.M{
			"name":          account.Name(),
			"role":          account.Role().String(),
			"password_hash": account.PasswordHash(),
			"token":         account.LegacyToken(),
			"active":        account.Active(),
			"disabled_at":   account.DisabledAt(),
			"disabled_by":   account.DisabledBy(),
			"updated_at":    account.UpdatedAt(),
		},
	})
	if err != nil {
		return domainuser.Account{}, err
	}
	if result.MatchedCount == 0 {
		return domainuser.Account{}, ErrUserNotFound
	}

	return account, nil
}

func (r *MongoUserRepository) List(ctx context.Context, activeOnly bool) ([]domainuser.Account, error) {
	ctx, cancel := context.WithTimeout(ctx, taskOperationTimeout)
	defer cancel()

	filter := bson.M{}
	if activeOnly {
		filter["active"] = true
	}

	cursor, err := r.collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var docs []userDocument
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, err
	}

	result := make([]domainuser.Account, 0, len(docs))
	for _, doc := range docs {
		result = append(result, userDocumentToAccount(doc))
	}
	return result, nil
}

func accountToDocument(account domainuser.Account) userDocument {
	return userDocument{
		ID:           account.ID(),
		Name:         account.Name(),
		Role:         account.Role().String(),
		PasswordHash: account.PasswordHash(),
		Token:        account.LegacyToken(),
		Active:       account.Active(),
		DisabledAt:   account.DisabledAt(),
		DisabledBy:   account.DisabledBy(),
		CreatedAt:    account.CreatedAt(),
		UpdatedAt:    account.UpdatedAt(),
	}
}

func userDocumentToAccount(doc userDocument) domainuser.Account {
	active := doc.Active
	if doc.DisabledAt != nil {
		active = false
	}
	role, _ := domainuser.ParseRole(doc.Role)

	return domainuser.Restore(
		doc.ID,
		doc.Name,
		role,
		doc.PasswordHash,
		doc.Token,
		active,
		doc.DisabledAt,
		doc.DisabledBy,
		doc.CreatedAt,
		doc.UpdatedAt,
	)
}
