package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/benebobaa/simple-order-service/internal/apperr"
	"github.com/benebobaa/simple-order-service/internal/store"
	"github.com/benebobaa/simple-order-service/internal/store/sqlc"
)

const (
	minPasswordLength = 8
	// bcrypt only considers the first 72 bytes of a password.
	maxPasswordBytes = 72
)

// Service implements user registration and login.
type Service struct {
	store  *store.Store
	tokens TokenIssuer
}

// NewService creates a Service.
func NewService(st *store.Store, tokens TokenIssuer) *Service {
	return &Service{store: st, tokens: tokens}
}

// RegisterInput carries the fields needed to register a user.
type RegisterInput struct {
	Name     string
	Email    string
	Password string
}

// Result is returned after a successful registration or login.
type Result struct {
	User      sqlc.User
	Token     string
	ExpiresAt time.Time
}

// Register creates a new user account and returns an access token.
func (s *Service) Register(ctx context.Context, in RegisterInput) (*Result, error) {
	name := strings.TrimSpace(in.Name)
	email := normalizeEmail(in.Email)

	if name == "" {
		return nil, apperr.Validation("name must not be blank", map[string]any{"name": "required"})
	}
	if len(in.Password) < minPasswordLength {
		return nil, apperr.Validation("password is too short", map[string]any{"password": fmt.Sprintf("min=%d", minPasswordLength)})
	}
	if len(in.Password) > maxPasswordBytes {
		return nil, apperr.Validation("password is too long", map[string]any{"password": fmt.Sprintf("max_bytes=%d", maxPasswordBytes)})
	}

	// Friendly pre-check; the unique constraint below remains the source of truth.
	if _, err := s.store.Queries().GetUserByEmail(ctx, email); err == nil {
		return nil, apperr.Conflict(apperr.CodeEmailExists, "an account with this email already exists", nil)
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("lookup user by email: %w", err)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	id, err := uuid.NewV7()
	if err != nil {
		return nil, fmt.Errorf("generate user id: %w", err)
	}

	user, err := s.store.Queries().CreateUser(ctx, sqlc.CreateUserParams{
		ID:           id,
		Name:         name,
		Email:        email,
		PasswordHash: string(hash),
	})
	if err != nil {
		if store.IsUniqueViolation(err) {
			return nil, apperr.Conflict(apperr.CodeEmailExists, "an account with this email already exists", nil)
		}
		return nil, fmt.Errorf("create user: %w", err)
	}

	return s.issueToken(user)
}

// Login authenticates a user by email and password.
func (s *Service) Login(ctx context.Context, email, password string) (*Result, error) {
	user, err := s.store.Queries().GetUserByEmail(ctx, normalizeEmail(email))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperr.InvalidCredentials()
		}
		return nil, fmt.Errorf("lookup user by email: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, apperr.InvalidCredentials()
	}

	return s.issueToken(user)
}

// GetUser returns a user by ID.
func (s *Service) GetUser(ctx context.Context, id uuid.UUID) (sqlc.User, error) {
	user, err := s.store.Queries().GetUserByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return sqlc.User{}, apperr.NotFound(apperr.CodeUserNotFound, "user not found")
		}
		return sqlc.User{}, fmt.Errorf("get user: %w", err)
	}
	return user, nil
}

func (s *Service) issueToken(user sqlc.User) (*Result, error) {
	token, expiresAt, err := s.tokens.Generate(user.ID, user.Email)
	if err != nil {
		return nil, fmt.Errorf("issue token: %w", err)
	}
	return &Result{User: user, Token: token, ExpiresAt: expiresAt}, nil
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
