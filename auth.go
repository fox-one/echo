package echo

import (
	"errors"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/fox-one/pkg/uuid"
)

func SignToken(userID, sessionID, conversationID string) (string, error) {
	claim := jwt.StandardClaims{
		Id:       conversationID,
		IssuedAt: time.Now().Unix(),
		Issuer:   userID,
	}

	return jwt.NewWithClaims(jwt.SigningMethodHS256, claim).SignedString([]byte(sessionID))
}

func ParseToken(token string, sessionID string) (string, error) {
	var claim jwt.StandardClaims
	if _, err := jwt.ParseWithClaims(token, &claim, func(t *jwt.Token) (interface{}, error) {
		return []byte(sessionID), nil
	}); err != nil {
		return "", err
	}

	if !uuid.IsUUID(claim.Id) {
		return "", errors.New("invalid token")
	}

	return claim.Id, nil
}
