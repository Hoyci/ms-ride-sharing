package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"ms-ride-sharing/services/user-service/internal/models"
	"ms-ride-sharing/services/user-service/internal/repository"
	"ms-ride-sharing/services/user-service/pkg"
	"ms-ride-sharing/shared/jwt"
	userpb "ms-ride-sharing/shared/proto/v1/user"
	"ms-ride-sharing/shared/types"

	jwtLib "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type UserService struct {
	repo       repository.UserRepository
	jwtService *jwt.JWTService
	rdbRepo    *redis.Client
}

func NewUserService(repo repository.UserRepository, jwtService *jwt.JWTService, rdbRepo *redis.Client) *UserService {
	return &UserService{
		repo:       repo,
		jwtService: jwtService,
		rdbRepo:    rdbRepo,
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
		return nil, ErrInternalServer
	}

	return modelUser, nil
}

func (s *UserService) Authenticate(ctx context.Context, email, password string) (*userpb.LoginResponse, error) {
	user, err := s.repo.GetByEmail(ctx, email)
	if err != nil || user == nil {
		log.Printf("error getting user by email: %v", err)
		return nil, ErrInvalidCredentials
	}

	err = pkg.CheckPassword(user.HashedPassword, password)
	if err != nil {
		log.Printf("error checking password: %v", err)
		return nil, ErrInvalidCredentials
	}

	accessTokenData, err := s.jwtService.GenerateToken(user.ID.String(), jwt.ACCESS)
	if err != nil {
		return nil, ErrInternalServer
	}
	refreshTokenData, err := s.jwtService.GenerateToken(user.ID.String(), jwt.REFRESH)
	if err != nil {
		return nil, ErrInternalServer
	}

	pipe := s.rdbRepo.Pipeline()

	pipe.Set(ctx, "session:"+user.ID.String(), accessTokenData.JTI, jwt.ACCESS_EXPIRATION)
	pipe.Set(ctx, "refresh_session:"+user.ID.String(), refreshTokenData.JTI, jwt.REFRESH_EXPIRATION)

	_, err = pipe.Exec(ctx)
	if err != nil {
		return nil, err
	}

	return &userpb.LoginResponse{
		Id:           user.ID.String(),
		Name:         user.FullName,
		Email:        user.Email,
		Type:         types.MapUserTypeDomainToProto(types.UserType(user.UserType)),
		AccessToken:  accessTokenData.SignedToken,
		RefreshToken: refreshTokenData.SignedToken,
	}, nil
}

func (s *UserService) Logout(ctx context.Context, userId string) (bool, error) {
	err := s.rdbRepo.Del(
		ctx,
		"session:"+userId,
		"refresh_session:"+userId,
	).Err()
	if err != nil {
		return false, ErrInternalServer
	}

	return true, nil
}

func (s *UserService) RefreshToken(ctx context.Context, tokenStr string) (*TokenResponse, error) {
	token, err := s.jwtService.Validate(tokenStr)
	if err != nil {
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(jwtLib.MapClaims)
	if !ok || claims["type"] != "REFRESH" {
		return nil, ErrInvalidTokenType
	}

	userID := claims["sub"].(string)
	incomingJTI := claims["jti"].(string)

	currentRefreshJTI, err := s.rdbRepo.Get(ctx, "refresh_session:"+userID).Result()
	if err != nil || currentRefreshJTI != incomingJTI {
		// Ensures the endpoint cannot be used with a previously issued refresh token
		s.rdbRepo.Del(ctx, "session:"+userID, "refresh_session:"+userID)
		return nil, ErrRefreshTokenReuseDetected
	}

	accessTokenData, err := s.jwtService.GenerateToken(userID, jwt.ACCESS)
	if err != nil {
		return nil, ErrInternalServer
	}
	refreshTokenData, err := s.jwtService.GenerateToken(userID, jwt.REFRESH)
	if err != nil {
		return nil, ErrInternalServer
	}

	s.rdbRepo.Set(ctx, "session:"+userID, accessTokenData.JTI, jwt.ACCESS_EXPIRATION)
	s.rdbRepo.Set(ctx, "refresh_session:"+userID, refreshTokenData.JTI, jwt.REFRESH_EXPIRATION)

	return &TokenResponse{
		AccessToken:  accessTokenData.SignedToken,
		RefreshToken: refreshTokenData.SignedToken,
	}, nil
}
