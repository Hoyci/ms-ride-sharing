package main

import (
	"context"
	"fmt"
	"log"
	"ms-ride-sharing/services/user-service/internal/config"
	"ms-ride-sharing/services/user-service/internal/handlers"
	"ms-ride-sharing/services/user-service/internal/repository"
	"ms-ride-sharing/services/user-service/internal/service"
	"ms-ride-sharing/shared/jwt"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/redis/go-redis/v9"
	grpcserver "google.golang.org/grpc"
)

func main() {
	configData := config.LoadConfig()
	log.Printf("starting user service on %s environment", configData.Environment)

	db := config.InitDB(configData)
	userRepo := repository.NewUserRepository(db)
	fmt.Println(configData.RedisHost, configData.RedisPort)
	rdbRepo := redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%s", configData.RedisHost, configData.RedisPort),
	})

	defer func() {
		if err := rdbRepo.Close(); err != nil {
			log.Printf("error while closing redis connection: %v", err)
		}
	}()

	jwtSvc := jwt.NewJWTService(configData.JWTSecret)
	userSvc := service.NewUserService(userRepo, jwtSvc, rdbRepo)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", configData.Port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpcserver.NewServer()
	handlers.NewGRPCHandler(grpcServer, userSvc, jwtSvc)

	// gracefull shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		signCh := make(chan os.Signal, 1)
		signal.Notify(signCh, os.Interrupt, syscall.SIGTERM)
		<-signCh
		cancel()
	}()

	go func() {
		log.Printf("starting gRPC user service on port %s", lis.Addr().String())
		if err := grpcServer.Serve(lis); err != nil {
			log.Printf("failed to serve: %v", err)
			cancel()
		}
	}()

	// wait for the shutdown signal
	<-ctx.Done()
	log.Println("shutting down the server...")
	grpcServer.GracefulStop()
}
