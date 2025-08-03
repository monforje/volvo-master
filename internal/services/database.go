package services

import (
	"context"
	"time"

	"volvomaster/internal/database"
	"volvomaster/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type DatabaseService struct {
	client   *mongo.Client
	db       *mongo.Database
	requests *mongo.Collection
	sessions *mongo.Collection
}

func NewDatabaseService(client *mongo.Client) *DatabaseService {
	db := database.GetDatabase(client, database.DatabaseName)

	return &DatabaseService{
		client:   client,
		db:       db,
		requests: database.GetCollection(db, "service_requests"),
		sessions: database.GetCollection(db, "user_sessions"),
	}
}

// SaveServiceRequest сохраняет заявку на обслуживание
func (s *DatabaseService) SaveServiceRequest(ctx context.Context, request *models.ServiceRequest) error {
	if request.ID.IsZero() {
		request.ID = primitive.NewObjectID()
		request.CreatedAt = time.Now()
	}
	request.UpdatedAt = time.Now()

	filter := bson.M{"_id": request.ID}
	upsert := true

	_, err := s.requests.ReplaceOne(ctx, filter, request, &options.ReplaceOptions{
		Upsert: &upsert,
	})

	return err
}

// GetServiceRequest получает заявку по ID
func (s *DatabaseService) GetServiceRequest(ctx context.Context, id primitive.ObjectID) (*models.ServiceRequest, error) {
	var request models.ServiceRequest
	err := s.requests.FindOne(ctx, bson.M{"_id": id}).Decode(&request)
	if err != nil {
		return nil, err
	}
	return &request, nil
}

// GetServiceRequestByUserID получает активную заявку пользователя
func (s *DatabaseService) GetServiceRequestByUserID(ctx context.Context, userID int64) (*models.ServiceRequest, error) {
	var request models.ServiceRequest
	filter := bson.M{
		"user_id": userID,
		"status":  bson.M{"$ne": "completed"},
	}

	err := s.requests.FindOne(ctx, filter).Decode(&request)
	if err != nil {
		return nil, err
	}
	return &request, nil
}

// SaveUserSession сохраняет сессию пользователя
func (s *DatabaseService) SaveUserSession(ctx context.Context, session *models.UserSession) error {
	session.UpdatedAt = time.Now()

	filter := bson.M{"user_id": session.UserID}
	upsert := true

	_, err := s.sessions.ReplaceOne(ctx, filter, session, &options.ReplaceOptions{
		Upsert: &upsert,
	})

	return err
}

// GetUserSession получает сессию пользователя
func (s *DatabaseService) GetUserSession(ctx context.Context, userID int64) (*models.UserSession, error) {
	var session models.UserSession
	err := s.sessions.FindOne(ctx, bson.M{"user_id": userID}).Decode(&session)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			// Создаем новую сессию
			session = models.UserSession{
				UserID:    userID,
				Stage:     models.StageStart,
				Data:      make(map[string]interface{}),
				UpdatedAt: time.Now(),
			}
			return &session, nil
		}
		return nil, err
	}
	return &session, nil
}

// DeleteUserSession удаляет сессию пользователя
func (s *DatabaseService) DeleteUserSession(ctx context.Context, userID int64) error {
	_, err := s.sessions.DeleteOne(ctx, bson.M{"user_id": userID})
	return err
}

// GetServiceRequests получает все заявки с фильтрацией
func (s *DatabaseService) GetServiceRequests(ctx context.Context, filter bson.M) ([]*models.ServiceRequest, error) {
	cursor, err := s.requests.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var requests []*models.ServiceRequest
	for cursor.Next(ctx) {
		var request models.ServiceRequest
		if err := cursor.Decode(&request); err != nil {
			continue
		}
		requests = append(requests, &request)
	}

	return requests, cursor.Err()
}
