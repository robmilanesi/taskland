// Package auth issues and verifies the JWTs used for authentication.
package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// ErrTokenExpired is returned by Verify when the token's exp claim is in the past.
var ErrTokenExpired = errors.New("token expired")

// ErrTokenInvalid is returned by Verify for any other rejection: bad signature,
// wrong signing method, malformed token, missing or non-UUID subject.
var ErrTokenInvalid = errors.New("token invalid")

// Issuer signs and verifies HS256 tokens with a fixed secret and lifetime.
type Issuer struct {
	secret []byte
	ttl    time.Duration
}

// NewIssuer returns an Issuer that signs tokens valid for ttl.
func NewIssuer(secret string, ttl time.Duration) *Issuer {
	return &Issuer{secret: []byte(secret), ttl: ttl}
}

// Issue returns a signed token whose subject is userID.
func (i *Issuer) Issue(userID uuid.UUID) (string, error) {
	now := time.Now()
	claims := jwt.RegisteredClaims{
		Subject:   userID.String(),
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(i.ttl)),
	}

	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(i.secret)
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}
	return signed, nil
}

// Verify checks the token's signature and expiry and returns its subject.
func (i *Issuer) Verify(token string) (uuid.UUID, error) {
	var claims jwt.RegisteredClaims

	_, err := jwt.ParseWithClaims(token, &claims,
		func(*jwt.Token) (any, error) { return i.secret, nil },
		jwt.WithValidMethods([]string{"HS256"}),
		jwt.WithExpirationRequired(),
	)
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return uuid.Nil, ErrTokenExpired
		}
		return uuid.Nil, fmt.Errorf("%w: %v", ErrTokenInvalid, err)
	}

	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return uuid.Nil, fmt.Errorf("%w: subject %q is not a uuid", ErrTokenInvalid, claims.Subject)
	}
	return userID, nil
}
