package jwt

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type tokenType string

const (
	ACCESS             tokenType = "ACCESS"
	REFRESH            tokenType = "REFRESH"
	ACCESS_EXPIRATION            = 15 * time.Minute
	REFRESH_EXPIRATION           = 24 * time.Hour
)

type JWTService struct {
	secret []byte
}

func NewJWTService(secret string) *JWTService {
	return &JWTService{secret: []byte(secret)}
}

type TokenResponse struct {
	SignedToken string
	JTI         string
}

func (j *JWTService) GenerateToken(userID string, tokenType tokenType) (*TokenResponse, error) {
	var exp int64

	switch tokenType {
	case ACCESS:
		exp = time.Now().Add(ACCESS_EXPIRATION).Unix()
	case REFRESH:
		exp = time.Now().Add(REFRESH_EXPIRATION).Unix()
	default:
		return nil, fmt.Errorf("invalid token type")
	}

	jti := uuid.New().String()

	claims := jwt.MapClaims{
		"sub":  userID,
		"jti":  jti,
		"type": string(tokenType),
		"exp":  exp,
		"iat":  time.Now().Unix(),
	}

	tokenObj := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := tokenObj.SignedString(j.secret)
	if err != nil {
		return nil, err
	}

	return &TokenResponse{
		SignedToken: signedToken,
		JTI:         jti,
	}, nil
}

func (j *JWTService) Validate(tokenStr string) (*jwt.Token, error) {
	return jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		return j.secret, nil
	})
}
