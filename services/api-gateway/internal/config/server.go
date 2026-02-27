package config

import (
	"context"
	"fmt"
	"log"
	"ms-ride-sharing/shared/jwt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	httpHandler "ms-ride-sharing/services/api-gateway/internal/handlers"
	userpb "ms-ride-sharing/shared/proto/v1/user"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/encoding/protojson"
)

func ConfigServer() {
	log.Println("registering gRPC service with gRPC-Gateway")
	configData := LoadConfig()

	jwtSvc := jwt.NewJWTService(configData.JWTSecret)
	rdbRepo := redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%s", configData.RedisHost, configData.RedisPort),
	})

	gwmux := runtime.NewServeMux(
		runtime.WithMarshalerOption(runtime.MIMEWildcard, &runtime.JSONPb{
			MarshalOptions: protojson.MarshalOptions{
				UseEnumNumbers:  false,
				EmitUnpopulated: true,
			},
		}),
		runtime.WithMetadata(func(ctx context.Context, req *http.Request) metadata.MD {
			if userID, ok := req.Context().Value("user_id").(string); ok {
				return metadata.Pairs("x-user-id", userID)
			}
			return nil
		}),
	)

	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	err := userpb.RegisterUserServiceHandlerFromEndpoint(ctx, gwmux, configData.UserSvcAddr, opts)
	if err != nil {
		log.Fatalf("failed to register user grpc service : %v", err)
	}

	mainMux := http.NewServeMux()

	protectedGateway := httpHandler.Chain(
		httpHandler.Logger,
		httpHandler.Recoverer,
		httpHandler.CORS,
		// httpHandler.AuthMiddleware(jwtSvc, rdb),
		httpHandler.AuthMiddleware(jwtSvc, rdbRepo),
	)(gwmux)

	serverAddr := fmt.Sprintf(":%s", configData.Port)
	mainMux.Handle("/", protectedGateway)

	server := &http.Server{
		Addr:    serverAddr,
		Handler: mainMux,
	}

	serverErrors := make(chan error, 1)
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Printf("Server listening on %s", serverAddr)
		serverErrors <- server.ListenAndServe()
	}()

	select {
	case err := <-serverErrors:
		log.Printf("Error starting the server: %v", err)
	case sig := <-shutdown:
		log.Printf("Server is shutting down due to %v signal", sig)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			log.Printf("Could not stop server gracefully: %v", err)
			server.Close()
		}
	}
}
