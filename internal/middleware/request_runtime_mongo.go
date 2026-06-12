package middleware

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const runtimeStoreTimeout = 5 * time.Second

type MongoRateLimiter struct {
	collection *mongo.Collection
	limit      int
	window     time.Duration
}

type mongoRateLimitDocument struct {
	ID          string    `bson:"_id"`
	ClientID    string    `bson:"client_id"`
	WindowStart time.Time `bson:"window_start"`
	WindowEnd   time.Time `bson:"window_end"`
	Count       int       `bson:"count"`
	ExpiresAt   time.Time `bson:"expires_at"`
}

func NewMongoRateLimiter(collection *mongo.Collection, limit int, window time.Duration) *MongoRateLimiter {
	return &MongoRateLimiter{
		collection: collection,
		limit:      limit,
		window:     window,
	}
}

func (l *MongoRateLimiter) Enabled() bool {
	return l != nil && l.collection != nil && l.limit > 0 && l.window > 0
}

func (l *MongoRateLimiter) Allow(ctx context.Context, clientID string, now time.Time) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, runtimeStoreTimeout)
	defer cancel()

	windowStart := now.UTC().Truncate(l.window)
	windowEnd := windowStart.Add(l.window)
	storageKey := clientID + "|" + windowStart.Format(time.RFC3339Nano)
	opts := options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After)

	var doc mongoRateLimitDocument
	err := l.collection.FindOneAndUpdate(ctx, bson.M{"_id": storageKey}, bson.M{
		"$setOnInsert": bson.M{
			"client_id":    clientID,
			"window_start": windowStart,
			"window_end":   windowEnd,
			"count":        0,
		},
		"$set": bson.M{
			"expires_at": windowEnd,
		},
		"$inc": bson.M{
			"count": 1,
		},
	}, opts).Decode(&doc)
	if err != nil {
		return false, err
	}

	return doc.Count <= l.limit, nil
}

type MongoIdempotencyStore struct {
	collection *mongo.Collection
	ttl        time.Duration
}

type mongoIdempotencyDocument struct {
	ID         string            `bson:"_id"`
	Scope      string            `bson:"scope"`
	Method     string            `bson:"method"`
	Path       string            `bson:"path"`
	RequestSum string            `bson:"request_sum"`
	Status     int               `bson:"status"`
	Body       []byte            `bson:"body,omitempty"`
	Headers    map[string]string `bson:"headers,omitempty"`
	ExpiresAt  time.Time         `bson:"expires_at"`
	State      string            `bson:"state"`
}

func NewMongoIdempotencyStore(collection *mongo.Collection, ttl time.Duration) *MongoIdempotencyStore {
	return &MongoIdempotencyStore{
		collection: collection,
		ttl:        ttl,
	}
}

func (s *MongoIdempotencyStore) Enabled() bool {
	return s != nil && s.collection != nil && s.ttl > 0
}

func (s *MongoIdempotencyStore) TTL() time.Duration {
	if s == nil {
		return 0
	}
	return s.ttl
}

func (s *MongoIdempotencyStore) Reserve(ctx context.Context, scope, key, requestSum string, now time.Time) (IdempotencyDecision, IdempotencyRecord, error) {
	ctx, cancel := context.WithTimeout(ctx, runtimeStoreTimeout)
	defer cancel()

	storageKey := idempotencyStorageKey(scope, key)
	current, found, err := s.find(ctx, storageKey)
	if err != nil {
		return IdempotencyDecisionAccept, IdempotencyRecord{}, err
	}
	if found && now.After(current.ExpiresAt) {
		if _, err := s.collection.DeleteOne(ctx, bson.M{"_id": storageKey}); err != nil {
			return IdempotencyDecisionAccept, IdempotencyRecord{}, err
		}
		found = false
	}
	if !found {
		doc := mongoIdempotencyDocument{
			ID:         storageKey,
			Scope:      scope,
			RequestSum: requestSum,
			ExpiresAt:  now.Add(s.ttl),
			State:      idempotencyStatePending,
		}
		_, err := s.collection.InsertOne(ctx, doc)
		if err == nil {
			return IdempotencyDecisionAccept, IdempotencyRecord{}, nil
		}
		current, found, err = s.find(ctx, storageKey)
		if err != nil {
			return IdempotencyDecisionAccept, IdempotencyRecord{}, err
		}
		if !found {
			return IdempotencyDecisionAccept, IdempotencyRecord{}, err
		}
	}

	if current.RequestSum != requestSum {
		return IdempotencyDecisionConflict, IdempotencyRecord{}, nil
	}
	if current.State == idempotencyStateCompleted {
		return IdempotencyDecisionReplay, idempotencyDocumentToRecord(current), nil
	}
	return IdempotencyDecisionInProgress, IdempotencyRecord{}, nil
}

func (s *MongoIdempotencyStore) Complete(ctx context.Context, scope, key, requestSum string, record IdempotencyRecord) error {
	ctx, cancel := context.WithTimeout(ctx, runtimeStoreTimeout)
	defer cancel()

	_, err := s.collection.UpdateOne(ctx, bson.M{
		"_id":         idempotencyStorageKey(scope, key),
		"request_sum": requestSum,
	}, bson.M{
		"$set": bson.M{
			"scope":       record.Scope,
			"method":      record.Method,
			"path":        record.Path,
			"request_sum": requestSum,
			"status":      record.Status,
			"body":        record.Body,
			"headers":     record.Headers,
			"expires_at":  record.ExpiresAt,
			"state":       idempotencyStateCompleted,
		},
	})
	return err
}

func (s *MongoIdempotencyStore) Release(ctx context.Context, scope, key, requestSum string) error {
	ctx, cancel := context.WithTimeout(ctx, runtimeStoreTimeout)
	defer cancel()

	_, err := s.collection.DeleteOne(ctx, bson.M{
		"_id":         idempotencyStorageKey(scope, key),
		"request_sum": requestSum,
		"state":       idempotencyStatePending,
	})
	return err
}

func (s *MongoIdempotencyStore) find(ctx context.Context, storageKey string) (mongoIdempotencyDocument, bool, error) {
	var doc mongoIdempotencyDocument
	err := s.collection.FindOne(ctx, bson.M{"_id": storageKey}).Decode(&doc)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return mongoIdempotencyDocument{}, false, nil
		}
		return mongoIdempotencyDocument{}, false, err
	}
	return doc, true, nil
}

func idempotencyDocumentToRecord(doc mongoIdempotencyDocument) IdempotencyRecord {
	return IdempotencyRecord{
		Scope:      doc.Scope,
		Method:     doc.Method,
		Path:       doc.Path,
		RequestSum: doc.RequestSum,
		Status:     doc.Status,
		Body:       doc.Body,
		Headers:    doc.Headers,
		ExpiresAt:  doc.ExpiresAt,
		State:      doc.State,
	}
}
