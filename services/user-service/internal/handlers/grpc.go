package handlers

import (
	"context"
	"errors"
	"log"
	"ms-ride-sharing/services/user-service/internal/service"
	"ms-ride-sharing/shared/jwt"
	userpb "ms-ride-sharing/shared/proto/v1/user"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type GRPCHandler struct {
	userpb.UnimplementedUserServiceServer
	userService *service.UserService
	jwtService  *jwt.JWTService
}

func NewGRPCHandler(
	server *grpc.Server,
	userService *service.UserService,
	jwtService *jwt.JWTService,
) *GRPCHandler {
	handler := &GRPCHandler{
		userService: userService,
		jwtService:  jwtService,
	}

	userpb.RegisterUserServiceServer(server, handler)
	return handler
}

func (h *GRPCHandler) CreateUser(ctx context.Context, req *userpb.CreateUserRequest) (*userpb.CreateUserResponse, error) {
	if err := req.ValidateAll(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	user, err := h.userService.CreateUser(ctx, req)

	if err != nil {
		// Se o erro for que o usuário já existe, aplicamos a máscara de segurança
		// Retornamos sucesso falso para evitar User Enumeration
		if errors.Is(err, service.ErrUserAlreadyExists) {
			return &userpb.CreateUserResponse{Id: uuid.New().String()}, nil
		}

		log.Printf("create user error: %v", err)
		return nil, status.Error(codes.Internal, service.ErrInternalServer.Error())
	}

	return &userpb.CreateUserResponse{Id: user.ID.String()}, nil
}

func (h *GRPCHandler) Login(ctx context.Context, req *userpb.LoginRequest) (*userpb.LoginResponse, error) {
	authRes, err := h.userService.Authenticate(ctx, req.Email, req.Password)
	if err != nil {
		log.Printf("login error: %v", err)
		switch {
		case errors.Is(err, service.ErrInvalidCredentials):
			return nil, status.Error(codes.InvalidArgument, service.ErrInvalidCredentials.Error())
		case errors.Is(err, service.ErrInternalServer):
			return nil, status.Error(codes.Internal, service.ErrInternalServer.Error())
		default:
			return nil, status.Error(codes.Internal, "unexpected error")
		}
	}

	return authRes, nil
}

func (h *GRPCHandler) Logout(ctx context.Context, req *userpb.LogoutRequest) (*userpb.LogoutResponse, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	var userID string

	if ok && len(md.Get("x-user-id")) > 0 {
		userID = md.Get("x-user-id")[0]
	}

	if userID == "" {
		return nil, status.Error(codes.Unauthenticated, "user_id is required")
	}

	ok, err := h.userService.Logout(ctx, userID)
	if err != nil {
		return nil, status.Error(codes.Internal, service.ErrInternalServer.Error())
	}

	return &userpb.LogoutResponse{Success: ok}, nil
}

func (h *GRPCHandler) RefreshToken(ctx context.Context, req *userpb.RefreshTokenRequest) (*userpb.RefreshTokenResponse, error) {
	data, err := h.userService.RefreshToken(ctx, req.Token)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidToken):
			return nil, status.Error(codes.Unauthenticated, service.ErrInvalidToken.Error())
		case errors.Is(err, service.ErrInvalidTokenType):
			return nil, status.Error(codes.Unauthenticated, service.ErrInvalidTokenType.Error())
		case errors.Is(err, service.ErrRefreshTokenReuseDetected):
			return nil, status.Error(codes.PermissionDenied, service.ErrRefreshTokenReuseDetected.Error())
		case errors.Is(err, service.ErrInternalServer):
			return nil, status.Error(codes.Internal, service.ErrInternalServer.Error())
		default:
			return nil, status.Error(codes.Internal, "unexpected error")
		}
	}

	return &userpb.RefreshTokenResponse{
		AccessToken:  data.AccessToken,
		RefreshToken: data.RefreshToken,
	}, nil
}
