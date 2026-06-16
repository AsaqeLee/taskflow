package repository

import (
	"context"
	"time"

	domainidentity "github.com/AsaqeLee/taskflow/internal/domain/identity"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type MongoIdentityRepository struct {
	refreshTokens       *mongo.Collection
	passwordResetTokens *mongo.Collection
}

type refreshTokenDocument struct {
	ID                  string     `bson:"_id"`
	UserID              string     `bson:"user_id"`
	TokenHash           string     `bson:"token_hash"`
	CreatedAt           time.Time  `bson:"created_at"`
	ExpiresAt           time.Time  `bson:"expires_at"`
	RevokedAt           *time.Time `bson:"revoked_at,omitempty"`
	ReplacedByTokenHash string     `bson:"replaced_by_token_hash,omitempty"`
}

type passwordResetTokenDocument struct {
	ID         string     `bson:"_id"`
	UserID     string     `bson:"user_id"`
	TokenHash  string     `bson:"token_hash"`
	CreatedAt  time.Time  `bson:"created_at"`
	ExpiresAt  time.Time  `bson:"expires_at"`
	ConsumedAt *time.Time `bson:"consumed_at,omitempty"`
}

func NewMongoIdentityRepository(refreshTokens, passwordResetTokens *mongo.Collection) *MongoIdentityRepository {
	return &MongoIdentityRepository{
		refreshTokens:       refreshTokens,
		passwordResetTokens: passwordResetTokens,
	}
}

func (r *MongoIdentityRepository) SaveRefreshToken(ctx context.Context, token domainidentity.RefreshToken) error {
	ctx, cancel := context.WithTimeout(ctx, taskOperationTimeout)
	defer cancel()

	if token.ID() == "" {
		token = token.AssignID(bson.NewObjectID().Hex())
	}
	_, err := r.refreshTokens.InsertOne(ctx, refreshTokenToDocument(token))
	return err
}

func (r *MongoIdentityRepository) FindRefreshToken(ctx context.Context, tokenHash string) (domainidentity.RefreshToken, error) {
	ctx, cancel := context.WithTimeout(ctx, taskOperationTimeout)
	defer cancel()

	var doc refreshTokenDocument
	err := r.refreshTokens.FindOne(ctx, bson.M{"token_hash": tokenHash}).Decode(&doc)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return domainidentity.RefreshToken{}, ErrRefreshTokenNotFound
		}
		return domainidentity.RefreshToken{}, err
	}

	return refreshTokenDocumentToModel(doc), nil
}

func (r *MongoIdentityRepository) RevokeRefreshToken(ctx context.Context, tokenHash string, revokedAt time.Time, replacedByHash string) error {
	ctx, cancel := context.WithTimeout(ctx, taskOperationTimeout)
	defer cancel()

	update := bson.M{
		"$set": bson.M{
			"revoked_at": revokedAt,
		},
	}
	if replacedByHash != "" {
		update["$set"].(bson.M)["replaced_by_token_hash"] = replacedByHash
	}

	result, err := r.refreshTokens.UpdateOne(ctx, bson.M{"token_hash": tokenHash}, update)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return ErrRefreshTokenNotFound
	}
	return nil
}

func (r *MongoIdentityRepository) RevokeUserRefreshTokens(ctx context.Context, userID string, revokedAt time.Time) error {
	ctx, cancel := context.WithTimeout(ctx, taskOperationTimeout)
	defer cancel()

	_, err := r.refreshTokens.UpdateMany(ctx, bson.M{
		"user_id":    userID,
		"revoked_at": bson.M{"$exists": false},
	}, bson.M{
		"$set": bson.M{
			"revoked_at": revokedAt,
		},
	})
	return err
}

func (r *MongoIdentityRepository) SavePasswordResetToken(ctx context.Context, token domainidentity.PasswordResetToken) error {
	ctx, cancel := context.WithTimeout(ctx, taskOperationTimeout)
	defer cancel()

	if token.ID() == "" {
		token = token.AssignID(bson.NewObjectID().Hex())
	}
	_, err := r.passwordResetTokens.InsertOne(ctx, passwordResetTokenToDocument(token))
	return err
}

func (r *MongoIdentityRepository) FindPasswordResetToken(ctx context.Context, tokenHash string) (domainidentity.PasswordResetToken, error) {
	ctx, cancel := context.WithTimeout(ctx, taskOperationTimeout)
	defer cancel()

	var doc passwordResetTokenDocument
	err := r.passwordResetTokens.FindOne(ctx, bson.M{"token_hash": tokenHash}).Decode(&doc)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return domainidentity.PasswordResetToken{}, ErrPasswordResetTokenNotFound
		}
		return domainidentity.PasswordResetToken{}, err
	}

	return passwordResetTokenDocumentToModel(doc), nil
}

func (r *MongoIdentityRepository) ConsumePasswordResetToken(ctx context.Context, tokenHash string, consumedAt time.Time) (domainidentity.PasswordResetToken, error) {
	ctx, cancel := context.WithTimeout(ctx, taskOperationTimeout)
	defer cancel()

	current, err := r.FindPasswordResetToken(ctx, tokenHash)
	if err != nil {
		return domainidentity.PasswordResetToken{}, err
	}

	result, err := r.passwordResetTokens.UpdateOne(ctx, bson.M{"token_hash": tokenHash}, bson.M{
		"$set": bson.M{
			"consumed_at": consumedAt,
		},
	})
	if err != nil {
		return domainidentity.PasswordResetToken{}, err
	}
	if result.MatchedCount == 0 {
		return domainidentity.PasswordResetToken{}, ErrPasswordResetTokenNotFound
	}

	return current.Consume(consumedAt), nil
}

func (r *MongoIdentityRepository) DeletePasswordResetTokensByUser(ctx context.Context, userID string) error {
	ctx, cancel := context.WithTimeout(ctx, taskOperationTimeout)
	defer cancel()

	_, err := r.passwordResetTokens.DeleteMany(ctx, bson.M{"user_id": userID})
	return err
}

func refreshTokenToDocument(token domainidentity.RefreshToken) refreshTokenDocument {
	return refreshTokenDocument{
		ID:                  token.ID(),
		UserID:              token.UserID(),
		TokenHash:           token.TokenHash(),
		CreatedAt:           token.CreatedAt(),
		ExpiresAt:           token.ExpiresAt(),
		RevokedAt:           token.RevokedAt(),
		ReplacedByTokenHash: token.ReplacedByTokenHash(),
	}
}

func refreshTokenDocumentToModel(doc refreshTokenDocument) domainidentity.RefreshToken {
	return domainidentity.RestoreRefreshToken(doc.ID, doc.UserID, doc.TokenHash, doc.CreatedAt, doc.ExpiresAt, doc.RevokedAt, doc.ReplacedByTokenHash)
}

func passwordResetTokenToDocument(token domainidentity.PasswordResetToken) passwordResetTokenDocument {
	return passwordResetTokenDocument{
		ID:         token.ID(),
		UserID:     token.UserID(),
		TokenHash:  token.TokenHash(),
		CreatedAt:  token.CreatedAt(),
		ExpiresAt:  token.ExpiresAt(),
		ConsumedAt: token.ConsumedAt(),
	}
}

func passwordResetTokenDocumentToModel(doc passwordResetTokenDocument) domainidentity.PasswordResetToken {
	return domainidentity.RestorePasswordResetToken(doc.ID, doc.UserID, doc.TokenHash, doc.CreatedAt, doc.ExpiresAt, doc.ConsumedAt)
}
