package main

import (
	"context"
	"fmt"
	"log"
	"ms-ride-sharing/services/user-service/internal/config"
	"ms-ride-sharing/services/user-service/internal/handler"
	"ms-ride-sharing/services/user-service/internal/repository"
	"ms-ride-sharing/services/user-service/internal/service"
	"net"
	"os"
	"os/signal"
	"syscall"

	grpcserver "google.golang.org/grpc"
)

func main() {
	configData := config.LoadConfig()
	log.Printf("starting user service on %s environment", configData.Environment)

	db := config.InitDB(configData)
	userRepo := repository.NewUserRepository(db)
	userSvc := service.NewUserService(userRepo)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", configData.Port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpcserver.NewServer()
	handler.NewGRPCHandler(grpcServer, userSvc)

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
