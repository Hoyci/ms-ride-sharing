package handlers

import (
	"log"
	"net/http"
	"strings"
	"time"
)

type Middleware func(http.Handler) http.Handler

func Chain(middlewares ...Middleware) Middleware {
	return func(final http.Handler) http.Handler {
		for i := len(middlewares) - 1; i >= 0; i-- {
			final = middlewares[i](final)
		}
		return final
	}
}

func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		next.ServeHTTP(w, r)

		log.Printf(
			"%s %s %s",
			r.Method,
			r.URL.Path,
			time.Since(start),
		)
	})
}

func Recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("Panic recovered: %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}()

		next.ServeHTTP(w, r)
	})
}

func CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func AuthMiddleware() Middleware {
	publicRoutes := map[string]bool{
		"POST:/api/v1/users":       true, // Login
		"POST:/api/v1/users/login": true, // Register user
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			routeKey := r.Method + ":" + r.URL.Path

			if publicRoutes[routeKey] {
				next.ServeHTTP(w, r)
				return
			}

			var tokenStr string

			authHeader := r.Header.Get("Authorization")
			if authHeader != "" {
				tokenStr = strings.TrimPrefix(authHeader, "Bearer ")
			} else {
				tokenStr = r.URL.Query().Get("token")
			}

			if tokenStr == "" {
				http.Error(w, "token not found", http.StatusUnauthorized)
				return
			}

			// token, err := jwtSvc.Validate(tokenStr)

			// if err != nil || !token.Valid {
			// 	http.Error(w, "token invalid or expired", http.StatusUnauthorized)
			// 	return
			// }

			// claims, ok := token.Claims.(jwtLib.MapClaims)
			// if !ok {
			// 	http.Error(w, "invalid claims", http.StatusUnauthorized)
			// 	return
			// }

			// userID := claims["sub"].(string)
			// jti := claims["jti"].(string)

			// activeJti, err := rdb.Get(r.Context(), "session:"+userID).Result()
			// if err == redis.Nil || activeJti != jti {
			// 	http.Error(w, "session expired", http.StatusUnauthorized)
			// 	return
			// }

			// ctx := context.WithValue(r.Context(), "user_id", userID)
			// next.ServeHTTP(w, r.WithContext(ctx))
			next.ServeHTTP(w, r)
		})
	}
}
