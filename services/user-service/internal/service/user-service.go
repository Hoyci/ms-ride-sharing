package service

import (
	"context"
	"errors"
	"fmt"
	"ms-ride-sharing/services/user-service/internal/models"
	"ms-ride-sharing/services/user-service/internal/repository"
	"ms-ride-sharing/services/user-service/pkg"
	userpb "ms-ride-sharing/shared/proto/v1/user"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserService struct {
	repo repository.UserRepository
}

func NewUserService(repo repository.UserRepository) *UserService {
	return &UserService{
		repo: repo,
	}
}

func (s *UserService) CreateUser(ctx context.Context, user *userpb.CreateUserRequest) (*models.User, error) {
	existingUser, err := s.repo.GetByEmail(ctx, user.Email)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("error checking existing user: %w", err)
	}

	if existingUser != nil {
		return nil, ErrUserAlreadyExists
	}

	hashedPassword, err := pkg.HashPassword(user.Password)
	if err != nil {
		return nil, fmt.Errorf("error hashing password: %w", err)
	}

	modelUser := &models.User{
		ID:             uuid.New(),
		FullName:       user.FullName,
		Email:          user.Email,
		HashedPassword: hashedPassword,
		UserType:       models.UserType(user.UserType.String()),
	}

	err = s.repo.Create(ctx, modelUser)
	if err != nil {
		return nil, InternalServerError
	}

	return modelUser, nil
}

// func (s *userService) GetUserByEmail(ctx context.Context, email string) (*domain.UserModel, error) {
// 	return s.repo.GetUserByEmail(ctx, email)
// }

// func (s *userService) Authenticate(ctx context.Context, email, password string) (*domain.UserModel, error) {
// 	user, err := s.repo.GetUserByEmail(ctx, email)
// 	if err != nil || user == nil {
// 		log.Printf("error getting user by email: %v", err)
// 		return nil, errors.New("invalid credentials")
// 	}

// 	err = pkg.CheckPassword(user.PasswordHashed, password)
// 	if err != nil {
// 		log.Printf("error checking password: %v", err)
// 		return nil, errors.New("invalid credentials")
// 	}

// 	return user, nil
// }
