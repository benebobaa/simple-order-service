package auth

import (
	"time"

	"github.com/google/uuid"

	"github.com/benebobaa/simple-order-service/internal/store/sqlc"
)

type registerRequest struct {
	Name     string `json:"name" binding:"required,max=255"`
	Email    string `json:"email" binding:"required,email,max=254"`
	Password string `json:"password" binding:"required,min=8,max=72"`
}

type loginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// UserResponse is the public representation of a user.
type UserResponse struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

// Response is returned by register and login.
type Response struct {
	Token     string       `json:"token"`
	TokenType string       `json:"token_type"`
	ExpiresAt time.Time    `json:"expires_at"`
	User      UserResponse `json:"user"`
}

func newResponse(result *Result) Response {
	return Response{
		Token:     result.Token,
		TokenType: "Bearer",
		ExpiresAt: result.ExpiresAt,
		User:      newUserResponse(result.User),
	}
}

func newUserResponse(user sqlc.User) UserResponse {
	return UserResponse{
		ID:        user.ID,
		Name:      user.Name,
		Email:     user.Email,
		CreatedAt: user.CreatedAt,
	}
}
