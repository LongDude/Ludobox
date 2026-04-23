package service

import (
	"authorization_service/internal/domain"
	"authorization_service/internal/repository"
	"context"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
)

func TestCreateUserMarksEmailConfirmedWhenVerificationDisabled(t *testing.T) {
	t.Parallel()

	password := "secret123"
	repo := &authServiceUserRepositoryStub{}
	service := NewAuthService(
		authServiceJWTStub{},
		authServiceEmailStub{},
		authServiceSessionStub{},
		repo,
		authServiceTokenBlocklistStub{},
		false,
		logrus.New(),
	)

	user, err := service.CreateUser(context.Background(), &domain.User{
		FirstName: "Local",
		LastName:  "User",
		Email:     "local@example.com",
		Password:  &password,
	})
	if err != nil {
		t.Fatalf("CreateUser returned error: %v", err)
	}

	if !user.EmailConfirmed {
		t.Fatalf("expected EmailConfirmed to be true when verification is disabled")
	}

	if repo.createdUser == nil || !repo.createdUser.EmailConfirmed {
		t.Fatalf("expected stored user to be marked confirmed")
	}
}

func TestLoginAllowsUnconfirmedUserWhenVerificationDisabled(t *testing.T) {
	t.Parallel()

	password := "secret123"
	hashedPassword := mustHashPassword(t, password)
	service := NewAuthService(
		authServiceJWTStub{},
		authServiceEmailStub{},
		authServiceSessionStub{},
		&authServiceUserRepositoryStub{
			userByEmail: &domain.User{
				ID:             42,
				Email:          "local@example.com",
				EmailConfirmed: false,
				Password:       &hashedPassword,
			},
		},
		authServiceTokenBlocklistStub{},
		false,
		logrus.New(),
	)

	tokens, err := service.Login(context.Background(), "local@example.com", password)
	if err != nil {
		t.Fatalf("Login returned error: %v", err)
	}

	if tokens == nil || tokens.AccessToken == "" || tokens.RefreshToken == "" {
		t.Fatalf("expected tokens to be returned")
	}
}

func mustHashPassword(t *testing.T, password string) string {
	t.Helper()

	hashedPassword, err := hashPassword(password)
	if err != nil {
		t.Fatalf("hashPassword returned error: %v", err)
	}

	return hashedPassword
}

func hashPassword(password string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}

	return string(hashedPassword), nil
}

type authServiceUserRepositoryStub struct {
	userByEmail *domain.User
	userByID    *domain.User
	createdUser *domain.User
}

func (s *authServiceUserRepositoryStub) GetUserByID(ctx context.Context, id int) (*domain.User, error) {
	if s.userByID != nil {
		return s.userByID, nil
	}
	if s.createdUser != nil && s.createdUser.ID == id {
		return s.createdUser, nil
	}
	return nil, repository.ErrorUserNotFound
}

func (s *authServiceUserRepositoryStub) GetUserByEmail(ctx context.Context, email string) (*domain.User, error) {
	if s.userByEmail != nil && s.userByEmail.Email == email {
		return s.userByEmail, nil
	}
	if s.createdUser != nil && s.createdUser.Email == email {
		return s.createdUser, nil
	}
	return nil, repository.ErrorUserNotFound
}

func (s *authServiceUserRepositoryStub) GetUserByGoogleID(ctx context.Context, id string) (*domain.User, error) {
	return nil, repository.ErrorUserNotFound
}

func (s *authServiceUserRepositoryStub) GetUserByYandexID(ctx context.Context, id string) (*domain.User, error) {
	return nil, repository.ErrorUserNotFound
}

func (s *authServiceUserRepositoryStub) GetUserByVkID(ctx context.Context, id string) (*domain.User, error) {
	return nil, repository.ErrorUserNotFound
}

func (s *authServiceUserRepositoryStub) UpdateUser(ctx context.Context, user *domain.User) error {
	s.userByID = user
	return nil
}

func (s *authServiceUserRepositoryStub) UpdatePassword(ctx context.Context, userID int, passwordHash string) error {
	return nil
}

func (s *authServiceUserRepositoryStub) SetOauthID(ctx context.Context, userID int, provider string, oauthID string) error {
	return nil
}

func (s *authServiceUserRepositoryStub) CreateUser(ctx context.Context, user *domain.User) (int, error) {
	clone := *user
	clone.ID = 1
	s.createdUser = &clone
	s.userByID = &clone
	s.userByEmail = &clone
	return clone.ID, nil
}

func (s *authServiceUserRepositoryStub) ConfirmEmail(ctx context.Context, userID int) error {
	return nil
}

func (s *authServiceUserRepositoryStub) ListUsers(ctx context.Context, filter repository.UserListFilter, page, limit int) ([]*domain.User, int, error) {
	return nil, 0, nil
}

func (s *authServiceUserRepositoryStub) DeleteUser(ctx context.Context, userID int) error {
	s.createdUser = nil
	return nil
}

type authServiceJWTStub struct{}

func (authServiceJWTStub) CreateJwtTokens(ctx context.Context, userID int) (*domain.UserTokens, error) {
	return &domain.UserTokens{
		AccessToken:  "access-token",
		RefreshToken: "refresh-token",
	}, nil
}

func (authServiceJWTStub) VerifyToken(ctx context.Context, token string, tokenType domain.TokenType) (int, error) {
	return 1, nil
}

func (authServiceJWTStub) ParseToken(ctx context.Context, refreshToken string) (*domain.TokenClaims, error) {
	return &domain.TokenClaims{}, nil
}

func (authServiceJWTStub) RefreshTokens(ctx context.Context, refreshToken string) (*domain.UserTokens, error) {
	return &domain.UserTokens{}, nil
}

func (authServiceJWTStub) ParseJTI(ctx context.Context, token string) (string, error) {
	return "jti", nil
}

type authServiceEmailStub struct{}

func (authServiceEmailStub) SendEmailConfirmation(ctx context.Context, userID int, email string) error {
	return nil
}

func (authServiceEmailStub) GenerateEmailConfirmationToken(ctx context.Context, userID int, email string) (string, error) {
	return "", nil
}

func (authServiceEmailStub) VerifyEmailConfirmationToken(ctx context.Context, token string) (int, error) {
	return 0, nil
}

func (authServiceEmailStub) SendPasswordResetConfirmation(ctx context.Context, userID int, email string) error {
	return nil
}

func (authServiceEmailStub) VerifyPasswordResetToken(ctx context.Context, token string) (int, string, string, time.Time, error) {
	return 0, "", "", time.Time{}, nil
}

func (authServiceEmailStub) SendNewPassword(ctx context.Context, email string, password string) error {
	return nil
}

type authServiceSessionStub struct{}

func (authServiceSessionStub) CreateSession(ctx context.Context, refreshToken string, accessJTI string) (*domain.Session, error) {
	return &domain.Session{}, nil
}

func (authServiceSessionStub) GetSessionByRefreshToken(ctx context.Context, token string) (*domain.Session, error) {
	return nil, nil
}

func (authServiceSessionStub) GetAllUserSessions(ctx context.Context, userID int) ([]*domain.Session, error) {
	return nil, nil
}

func (authServiceSessionStub) DeleteSession(ctx context.Context, sessionID string) error {
	return nil
}

func (authServiceSessionStub) DeleteAllUserSessions(ctx context.Context, userID int) error {
	return nil
}

func (authServiceSessionStub) ValidateSession(ctx context.Context, refreshToken string) (*domain.Session, error) {
	return nil, nil
}

type authServiceTokenBlocklistStub struct{}

func (authServiceTokenBlocklistStub) IsBlocked(ctx context.Context, jti string) (bool, error) {
	return false, nil
}

func (authServiceTokenBlocklistStub) Block(ctx context.Context, jti string, exp time.Duration) error {
	return nil
}
