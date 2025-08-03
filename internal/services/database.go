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
	client         *mongo.Client
	db             *mongo.Database
	requests       *mongo.Collection
	sessions       *mongo.Collection
	users          *mongo.Collection
	availableDates *mongo.Collection
}

func NewDatabaseService(client *mongo.Client) *DatabaseService {
	db := database.GetDatabase(client, database.DatabaseName)

	return &DatabaseService{
		client:         client,
		db:             db,
		requests:       database.GetCollection(db, "service_requests"),
		sessions:       database.GetCollection(db, "user_sessions"),
		users:          database.GetCollection(db, "users"),
		availableDates: database.GetCollection(db, "available_dates"),
	}
}

// User methods
func (s *DatabaseService) SaveUser(ctx context.Context, user *models.User) error {
	// Проверяем, существует ли пользователь
	existingUser, err := s.GetUser(ctx, user.UserID)
	if err != nil && err != mongo.ErrNoDocuments {
		return err
	}

	if existingUser != nil {
		// Обновляем существующего пользователя
		user.ID = existingUser.ID
		user.CreatedAt = existingUser.CreatedAt
	} else {
		// Создаем нового пользователя
		user.ID = primitive.NewObjectID()
		user.CreatedAt = time.Now()
	}
	user.UpdatedAt = time.Now()

	filter := bson.M{"user_id": user.UserID}
	upsert := true

	_, err = s.users.ReplaceOne(ctx, filter, user, &options.ReplaceOptions{
		Upsert: &upsert,
	})

	return err
}

func (s *DatabaseService) GetUser(ctx context.Context, userID int64) (*models.User, error) {
	var user models.User
	err := s.users.FindOne(ctx, bson.M{"user_id": userID}).Decode(&user)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// AvailableDate methods
func (s *DatabaseService) SaveAvailableDate(ctx context.Context, date *models.AvailableDate) error {
	if date.ID.IsZero() {
		date.ID = primitive.NewObjectID()
		date.CreatedAt = time.Now()
	}
	date.UpdatedAt = time.Now()

	filter := bson.M{"_id": date.ID}
	upsert := true

	_, err := s.availableDates.ReplaceOne(ctx, filter, date, &options.ReplaceOptions{
		Upsert: &upsert,
	})

	return err
}

func (s *DatabaseService) GetAvailableDates(ctx context.Context) ([]*models.AvailableDate, error) {
	filter := bson.M{
		"is_active": true,
		"date":      bson.M{"$gte": time.Now().Truncate(24 * time.Hour)},
	}
	opts := options.Find().SetSort(bson.D{{Key: "date", Value: 1}})

	cursor, err := s.availableDates.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var dates []*models.AvailableDate
	for cursor.Next(ctx) {
		var date models.AvailableDate
		if err := cursor.Decode(&date); err != nil {
			continue
		}
		dates = append(dates, &date)
	}

	return dates, cursor.Err()
}

func (s *DatabaseService) GetAvailableDateByID(ctx context.Context, id primitive.ObjectID) (*models.AvailableDate, error) {
	var date models.AvailableDate
	err := s.availableDates.FindOne(ctx, bson.M{"_id": id}).Decode(&date)
	if err != nil {
		return nil, err
	}
	return &date, nil
}

// ServiceRequest methods
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

func (s *DatabaseService) GetServiceRequest(ctx context.Context, id primitive.ObjectID) (*models.ServiceRequest, error) {
	var request models.ServiceRequest
	err := s.requests.FindOne(ctx, bson.M{"_id": id}).Decode(&request)
	if err != nil {
		return nil, err
	}
	return &request, nil
}

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

// UserSession methods
func (s *DatabaseService) SaveUserSession(ctx context.Context, session *models.UserSession) error {
	session.UpdatedAt = time.Now()

	filter := bson.M{"user_id": session.UserID}
	upsert := true

	_, err := s.sessions.ReplaceOne(ctx, filter, session, &options.ReplaceOptions{
		Upsert: &upsert,
	})

	return err
}

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
