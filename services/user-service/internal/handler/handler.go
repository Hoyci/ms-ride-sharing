package handler

import (
	"context"
	"errors"
	"log"
	"ms-ride-sharing/services/user-service/internal/service"
	pbu "ms-ride-sharing/shared/proto/v1/user"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type gRPCHandler struct {
	pbu.UnimplementedUserServiceServer
	userService *service.UserService
}

func NewGRPCHandler(
	server *grpc.Server,
	userService *service.UserService,
) *gRPCHandler {
	handler := &gRPCHandler{
		userService: userService,
	}

	pbu.RegisterUserServiceServer(server, handler)
	return handler
}

func (h *gRPCHandler) CreateUser(ctx context.Context, req *pbu.CreateUserRequest) (*pbu.CreateUserResponse, error) {
	user, err := h.userService.CreateUser(ctx, req)

	if err != nil {
		// Se o erro for que o usuário já existe, aplicamos a máscara de segurança
		// Retornamos sucesso falso para evitar User Enumeration
		if errors.Is(err, service.ErrUserAlreadyExists) {
			return &pbu.CreateUserResponse{Id: uuid.New().String()}, nil
		}

		log.Printf("create user error: %v", err)
		return nil, status.Error(codes.Internal, service.InternalServerError.Error())
	}

	return &pbu.CreateUserResponse{Id: user.ID.String()}, nil
}
