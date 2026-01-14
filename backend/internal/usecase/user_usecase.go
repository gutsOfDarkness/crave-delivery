// Package usecase implements user business logic
package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"fooddelivery/internal/domain"
	"fooddelivery/internal/repository"
	"fooddelivery/pkg/logger"
)

// User-related errors
var (
	ErrUserExists     = errors.New("user with this phone number already exists")
	ErrUserNotFound   = errors.New("user not found")
	ErrInvalidOTP     = errors.New("invalid or expired OTP")
	ErrUnauthorized   = errors.New("unauthorized")
)

// UserUsecase handles user-related business logic
type UserUsecase struct {
	userRepo  *repository.UserRepository
	jwtSecret string
	jwtExpiry time.Duration
	log       *logger.Logger
}

// NewUserUsecase creates a new user usecase
func NewUserUsecase(userRepo *repository.UserRepository, log *logger.Logger) *UserUsecase {
	return &UserUsecase{
		userRepo:  userRepo,
		jwtSecret: "", // Set via SetJWTConfig
		jwtExpiry: 24 * time.Hour,
		log:       log,
	}
}

// SetJWTConfig sets JWT configuration
func (u *UserUsecase) SetJWTConfig(secret string, expiryHours int) {
	u.jwtSecret = secret
	u.jwtExpiry = time.Duration(expiryHours) * time.Hour
}

// RegisterRequest contains registration data
type RegisterRequest struct {
	PhoneNumber string `json:"phone_number"`
	Name        string `json:"name"`
	Email       string `json:"email,omitempty"`
}

// RegisterResponse contains registration result
type RegisterResponse struct {
	UserID  uuid.UUID `json:"user_id"`
	Message string    `json:"message"`
}

// Register creates a new user account
func (u *UserUsecase) Register(ctx context.Context, req RegisterRequest) (*RegisterResponse, error) {
	// Check if user exists
	existing, err := u.userRepo.GetByPhoneNumber(ctx, req.PhoneNumber)
	if err == nil && existing != nil {
		return nil, ErrUserExists
	}
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return nil, fmt.Errorf("failed to check existing user: %w", err)
	}

	now := time.Now()
	user := &domain.User{
		PhoneNumber: req.PhoneNumber,
		Name:        req.Name,
		Email:       req.Email,
		IsAdmin:     false,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := u.userRepo.Create(ctx, user); err != nil {
		if errors.Is(err, repository.ErrDuplicateKey) {
			return nil, ErrUserExists
		}
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	u.log.Info("User registered", "user_id", user.ID.String(), "phone", req.PhoneNumber)

	return &RegisterResponse{
		UserID:  user.ID,
		Message: "Registration successful. Please verify OTP.",
	}, nil
}

// LoginRequest contains login data
type LoginRequest struct {
	PhoneNumber string `json:"phone_number"`
}

// LoginResponse contains login result
type LoginResponse struct {
	UserID  uuid.UUID `json:"user_id"`
	Message string    `json:"message"`
}

// Login initiates OTP-based login
func (u *UserUsecase) Login(ctx context.Context, req LoginRequest) (*LoginResponse, error) {
	user, err := u.userRepo.GetByPhoneNumber(ctx, req.PhoneNumber)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to find user: %w", err)
	}

	// In production: Generate and send OTP via SMS service
	// For now, we simulate OTP generation
	u.log.Info("OTP requested", "user_id", user.ID.String(), "phone", req.PhoneNumber)

	return &LoginResponse{
		UserID:  user.ID,
		Message: "OTP sent to your phone number",
	}, nil
}

// VerifyOTPRequest contains OTP verification data
type VerifyOTPRequest struct {
	PhoneNumber string `json:"phone_number"`
	OTP         string `json:"otp"`
}

// VerifyOTPResponse contains verification result with JWT token
type VerifyOTPResponse struct {
	Token     string    `json:"token"`
	UserID    uuid.UUID `json:"user_id"`
	ExpiresAt time.Time `json:"expires_at"`
}

// VerifyOTP verifies OTP and returns JWT token
func (u *UserUsecase) VerifyOTP(ctx context.Context, req VerifyOTPRequest) (*VerifyOTPResponse, error) {
	// In production: Verify OTP from cache/database
	// For demo, accept "123456" as valid OTP
	if req.OTP != "123456" {
		return nil, ErrInvalidOTP
	}

	user, err := u.userRepo.GetByPhoneNumber(ctx, req.PhoneNumber)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to find user: %w", err)
	}

	// Generate JWT token
	expiresAt := time.Now().Add(u.jwtExpiry)
	token, err := u.generateJWT(user, expiresAt)
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	u.log.Info("User logged in", "user_id", user.ID.String())

	return &VerifyOTPResponse{
		Token:     token,
		UserID:    user.ID,
		ExpiresAt: expiresAt,
	}, nil
}

// JWTClaims contains JWT payload
type JWTClaims struct {
	UserID  uuid.UUID `json:"user_id"`
	IsAdmin bool      `json:"is_admin"`
	jwt.RegisteredClaims
}

// generateJWT creates a new JWT token
func (u *UserUsecase) generateJWT(user *domain.User, expiresAt time.Time) (string, error) {
	claims := JWTClaims{
		UserID:  user.ID,
		IsAdmin: user.IsAdmin,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   user.ID.String(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(u.jwtSecret))
}

// ValidateToken validates JWT token and returns claims
func (u *UserUsecase) ValidateToken(tokenString string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(u.jwtSecret), nil
	})

	if err != nil {
		return nil, ErrUnauthorized
	}

	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, ErrUnauthorized
}

// GetUser retrieves user by ID
func (u *UserUsecase) GetUser(ctx context.Context, userID uuid.UUID) (*domain.User, error) {
	user, err := u.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return user, nil
}
