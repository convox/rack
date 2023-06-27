package jwt

import (
	"fmt"
	"time"

	"github.com/convox/rack/pkg/structs"
	"github.com/golang-jwt/jwt/v4"
)

type TokenData struct {
	User      string
	Role      string
	ExpiresAt time.Time
}

type JwtManager struct {
	signKey []byte
}

func NewJwtManager(signKey string) *JwtManager {
	return &JwtManager{
		signKey: []byte(signKey),
	}
}

func (j *JwtManager) ReadToken(duration time.Duration) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user":      "system-read",
		"role":      structs.ConvoxRoleRead,
		"expiresAt": time.Now().UTC().Add(duration).Unix(),
	})

	// Sign and get the complete encoded token as a string using the secret
	tokenString, err := token.SignedString(j.signKey)
	if err != nil {
		return "", err
	}
	return tokenString, nil
}

func (j *JwtManager) WriteToken(duration time.Duration) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user":      "system-write",
		"role":      structs.ConvoxRoleReadWrite,
		"expiresAt": time.Now().UTC().Add(duration).Unix(),
	})

	// Sign and get the complete encoded token as a string using the secret
	tokenString, err := token.SignedString(j.signKey)
	if err != nil {
		return "", err
	}
	return tokenString, nil
}

func (j *JwtManager) Verify(token string) (*TokenData, error) {
	d := &TokenData{}
	tk, err := jwt.Parse(token, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}

		return j.signKey, nil
	})
	if err != nil {
		return nil, err
	}

	if !tk.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	if claims, ok := tk.Claims.(jwt.MapClaims); ok {
		d.User = claims["user"].(string)
		d.Role = claims["role"].(string)
		expiresAt := (int64)(claims["expiresAt"].(float64))
		d.ExpiresAt = time.Unix(expiresAt, 0)
		if d.ExpiresAt.UTC().Before(time.Now().UTC()) {
			return nil, fmt.Errorf("token is expired")
		}
	} else {
		return nil, fmt.Errorf("invalid token")
	}
	return d, nil
}
