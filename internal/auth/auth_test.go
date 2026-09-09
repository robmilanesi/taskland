package auth

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const testSecret = "test-secret-that-is-at-least-32-chars"

func TestIssuer_RoundTrip(t *testing.T) {
	iss := NewIssuer(testSecret, time.Hour)
	want := uuid.New()

	token, err := iss.Issue(want)
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}

	got, err := iss.Verify(token)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if got != want {
		t.Errorf("subject = %s, want %s", got, want)
	}
}

func TestIssuer_Expired(t *testing.T) {
	iss := NewIssuer(testSecret, -time.Minute)

	token, err := iss.Issue(uuid.New())
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}

	if _, err := iss.Verify(token); !errors.Is(err, ErrTokenExpired) {
		t.Errorf("expected ErrTokenExpired, got %v", err)
	}
}

func TestIssuer_WrongSecret(t *testing.T) {
	signer := NewIssuer(strings.Repeat("a", 32), time.Hour)
	verifier := NewIssuer(strings.Repeat("b", 32), time.Hour)

	token, err := signer.Issue(uuid.New())
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}

	if _, err := verifier.Verify(token); !errors.Is(err, ErrTokenInvalid) {
		t.Errorf("expected ErrTokenInvalid, got %v", err)
	}
}

func TestIssuer_Garbage(t *testing.T) {
	iss := NewIssuer(testSecret, time.Hour)

	for _, s := range []string{"", "not-a-jwt", "a.b.c"} {
		if _, err := iss.Verify(s); !errors.Is(err, ErrTokenInvalid) {
			t.Errorf("Verify(%q): expected ErrTokenInvalid, got %v", s, err)
		}
	}
}

func TestIssuer_RejectsNoneAlg(t *testing.T) {
	claims := jwt.RegisteredClaims{
		Subject:   uuid.NewString(),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodNone, claims).
		SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatalf("craft none token: %v", err)
	}

	iss := NewIssuer(testSecret, time.Hour)
	if _, err := iss.Verify(token); !errors.Is(err, ErrTokenInvalid) {
		t.Errorf("alg=none token must be rejected, got %v", err)
	}
}

func TestIssuer_BadSubject(t *testing.T) {
	claims := jwt.RegisteredClaims{
		Subject:   "not-a-uuid",
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).
		SignedString([]byte(testSecret))
	if err != nil {
		t.Fatalf("sign: %v", err)
	}

	iss := NewIssuer(testSecret, time.Hour)
	if _, err := iss.Verify(token); !errors.Is(err, ErrTokenInvalid) {
		t.Errorf("non-uuid subject must be ErrTokenInvalid, got %v", err)
	}
}

func TestIssuer_RejectsMissingExpiry(t *testing.T) {
	claims := jwt.RegisteredClaims{Subject: uuid.NewString()}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).
		SignedString([]byte(testSecret))
	if err != nil {
		t.Fatalf("sign: %v", err)
	}

	iss := NewIssuer(testSecret, time.Hour)
	if _, err := iss.Verify(token); !errors.Is(err, ErrTokenInvalid) {
		t.Errorf("token without exp must be rejected, got %v", err)
	}
}
