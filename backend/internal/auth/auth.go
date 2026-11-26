package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrTokenExpired       = errors.New("token expired")
	ErrTokenInvalid       = errors.New("invalid token")
	ErrUserLocked         = errors.New("account locked")
	ErrMFARequired        = errors.New("MFA required")
	ErrInvalidMFA         = errors.New("invalid MFA code")
	ErrPasswordTooWeak    = errors.New("password does not meet requirements")
)

type Config struct {
	JWTSecret        string
	JWTIssuer        string
	AccessTokenTTL   time.Duration
	RefreshTokenTTL  time.Duration
	MFAIssuer        string
	BcryptCost       int
	MaxLoginAttempts int
	LockoutDuration  time.Duration
	PasswordMinLength int
	RequireMFA       bool
}

type Claims struct {
	UserID         uuid.UUID `json:"user_id"`
	Email          string    `json:"email"`
	Role           string    `json:"role"`
	OrganizationID uuid.UUID `json:"organization_id"`
	TokenType      string    `json:"token_type"` // access, refresh
	jwt.RegisteredClaims
}

type MFAConfig struct {
	Secret string
	Issuer string
}

type AuthService struct {
	config Config
}

func NewAuthService(cfg Config) *AuthService {
	return &AuthService{config: cfg}
}

func (s *AuthService) HashPassword(password string) (string, error) {
	if len(password) < s.config.PasswordMinLength {
		return "", ErrPasswordTooWeak
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), s.config.BcryptCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	return string(hash), nil
}

func (s *AuthService) VerifyPassword(password, hash string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}

func (s *AuthService) GenerateAccessToken(user *UserClaims) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID:         user.UserID,
		Email:          user.Email,
		Role:           user.Role,
		OrganizationID: user.OrganizationID,
		TokenType:      "access",
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    s.config.JWTIssuer,
			Subject:   user.UserID.String(),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.config.AccessTokenTTL)),
			NotBefore: jwt.NewNumericDate(now),
			IssuedAt:  jwt.NewNumericDate(now),
			ID:        uuid.New().String(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.config.JWTSecret))
}

func (s *AuthService) GenerateRefreshToken(user *UserClaims) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID:         user.UserID,
		Email:          user.Email,
		Role:           user.Role,
		OrganizationID: user.OrganizationID,
		TokenType:      "refresh",
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    s.config.JWTIssuer,
			Subject:   user.UserID.String(),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.config.RefreshTokenTTL)),
			NotBefore: jwt.NewNumericDate(now),
			IssuedAt:  jwt.NewNumericDate(now),
			ID:        uuid.New().String(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.config.JWTSecret))
}

func (s *AuthService) ParseToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.config.JWTSecret), nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, ErrTokenInvalid
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		if claims.TokenType != "access" && claims.TokenType != "refresh" {
			return nil, ErrTokenInvalid
		}
		return claims, nil
	}

	return nil, ErrTokenInvalid
}

func (s *AuthService) ParseRefreshToken(tokenString string) (*Claims, error) {
	claims, err := s.ParseToken(tokenString)
	if err != nil {
		return nil, err
	}
	if claims.TokenType != "refresh" {
		return nil, ErrTokenInvalid
	}
	return claims, nil
}

func (s *AuthService) GenerateMFSecret() (string, error) {
	secret := make([]byte, 20)
	if _, err := rand.Read(secret); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(secret), nil
}

func (s *AuthService) GenerateMFAURI(email, secret string) string {
	return fmt.Sprintf("otpauth://totp/%s:%s?secret=%s&issuer=%s&algorithm=SHA1&digits=6&period=30",
		s.config.MFAIssuer, email, secret, s.config.MFAIssuer)
}

func (s *AuthService) VerifyMFACode(secret, code string) bool {
	// Simplified TOTP verification - in production use a proper TOTP library
	// This is a placeholder implementation
	return subtle.ConstantTimeCompare([]byte(code), []byte(secret[:6])) == 1
}

func (s *AuthService) GenerateRandomToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

func (s *AuthService) GenerateAPIKey() (string, string, error) {
	prefix := "nrmm"
	randomPart, err := s.GenerateRandomToken(32)
	if err != nil {
		return "", "", err
	}
	fullKey := fmt.Sprintf("%s_%s", prefix, randomPart)

	hash, err := s.HashPassword(fullKey)
	if err != nil {
		return "", "", err
	}

	return fullKey, hash, nil
}

func (s *AuthService) VerifyAPIKey(key, hash string) error {
	return s.VerifyPassword(key, hash)
}

func (s *AuthService) ValidatePasswordStrength(password string) error {
	if len(password) < s.config.PasswordMinLength {
		return ErrPasswordTooWeak
	}

	hasUpper := false
	hasLower := false
	hasNumber := false
	hasSpecial := false

	for _, c := range password {
		switch {
		case c >= 'A' && c <= 'Z':
			hasUpper = true
		case c >= 'a' && c <= 'z':
			hasLower = true
		case c >= '0' && c <= '9':
			hasNumber = true
		case strings.ContainsRune("!@#$%^&*()_+-=[]{}|;:,.<>?", c):
			hasSpecial = true
		}
	}

	if !hasUpper || !hasLower || !hasNumber || !hasSpecial {
		return ErrPasswordTooWeak
	}

	return nil
}

func (s *AuthService) Argon2Hash(password string) (string, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}

	hash := argon2.IDKey([]byte(password), salt, 3, 64*1024, 4, 32)

	b64Salt := base64.StdEncoding.EncodeToString(salt)
	b64Hash := base64.StdEncoding.EncodeToString(hash)

	return fmt.Sprintf("$argon2id$v=19$m=65536,t=3,p=4$%s$%s", b64Salt, b64Hash), nil
}

func (s *AuthService) Argon2Verify(password, encodedHash string) error {
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 {
		return ErrInvalidCredentials
	}

	var memory, iterations uint32
	var parallelism uint8
	_, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &iterations, &parallelism)
	if err != nil {
		return ErrInvalidCredentials
	}

	salt, err := base64.StdEncoding.DecodeString(parts[4])
	if err != nil {
		return ErrInvalidCredentials
	}

	expectedHash, err := base64.StdEncoding.DecodeString(parts[5])
	if err != nil {
		return ErrInvalidCredentials
	}

	actualHash := argon2.IDKey([]byte(password), salt, iterations, memory, parallelism, uint32(len(expectedHash)))

	if subtle.ConstantTimeCompare(actualHash, expectedHash) == 1 {
		return nil
	}
	return ErrInvalidCredentials
}

type UserClaims struct {
	UserID         uuid.UUID
	Email          string
	Role           string
	OrganizationID uuid.UUID
}

func (s *AuthService) Config() Config {
	return s.config
}