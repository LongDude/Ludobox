package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"user_service/internal/app"
	"user_service/internal/domain"
	"user_service/internal/repository"
	"user_service/internal/transport/http/presenters"

	"github.com/gin-gonic/gin"
)

func authenticatedUserID(ctx *gin.Context) (int, error) {
	userIDHeader := strings.TrimSpace(ctx.GetHeader("X-Authenticated-User"))
	if userIDHeader == "" {
		return 0, fmt.Errorf("missing X-Authenticated-User header")
	}

	userID, err := strconv.Atoi(userIDHeader)
	if err != nil || userID <= 0 {
		return 0, fmt.Errorf("invalid X-Authenticated-User header")
	}

	return userID, nil
}

func respondUser(ctx *gin.Context, user *domain.User) {
	ctx.JSON(http.StatusOK, presenters.UserResponse{
		UserID:   user.ID,
		Nickname: user.NickName,
		Balance:  user.Balance,
		Rating:   user.Rating,
		Rank:     user.Rank,
	})
}

func normalizeNickname(raw string) (string, error) {
	nickname := strings.TrimSpace(raw)
	if nickname == "" {
		return "", fmt.Errorf("nickname cannot be empty")
	}
	if len(nickname) > 128 {
		return "", fmt.Errorf("nickname is too long")
	}

	return nickname, nil
}

// Get user by id
// @Summary Get current user
// @Description Returns the current user profile. If the user is absent in user_service, it is created automatically with nickname user_{id} and zero balance.
// @Tags User
// @Accept json
// @Produce json
// @Param X-Authenticated-User header string true "Authenticated user id"
// @Param Authorization header string true "Authorization Token"
// @Success 200 {object} presenters.UserResponse
// @Failure 401 {object} presenters.ErrorResponse
// @Failure 500 {object} presenters.ErrorResponse
// @Router /user [get]
func GetUserByID(ctx *gin.Context, a *app.App) {
	userID, err := authenticatedUserID(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, presenters.Error(err))
		return
	}

	user, err := a.UserService.GetUserByID(ctx, userID)
	if err == nil {
		respondUser(ctx, user)
		return
	}
	if !errors.Is(err, repository.ErrorUserNotFound) {
		ctx.JSON(http.StatusInternalServerError, presenters.Error(fmt.Errorf("get user failed: %w", err)))
		return
	}

	user, err = a.UserService.CreateUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, repository.ErrorUserAlreadyExist) {
			user, err = a.UserService.GetUserByID(ctx, userID)
			if err == nil {
				respondUser(ctx, user)
				return
			}
		}
		ctx.JSON(http.StatusInternalServerError, presenters.Error(fmt.Errorf("auto-create user failed: %w", err)))
		return
	}

	respondUser(ctx, user)
}

// Create user by id
// @Summary Create current user
// @Description Creates the current user. If a request body is provided, nickname and initial balance are applied after creation.
// @Tags User
// @Accept json
// @Produce json
// @Param X-Authenticated-User header string true "Authenticated user id"
// @Param Authorization header string true "Authorization Token"
// @Param request body presenters.UserCreateRequest false "Optional create payload"
// @Success 201 {object} presenters.UserResponse
// @Failure 400 {object} presenters.ErrorResponse
// @Failure 401 {object} presenters.ErrorResponse
// @Failure 409 {object} presenters.ErrorResponse
// @Failure 500 {object} presenters.ErrorResponse
// @Router /user [post]
func CreateUserByID(ctx *gin.Context, a *app.App) {
	userID, err := authenticatedUserID(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, presenters.Error(err))
		return
	}

	var req presenters.UserCreateRequest
	if ctx.Request.ContentLength > 0 {
		if err := ctx.ShouldBindJSON(&req); err != nil {
			ctx.JSON(http.StatusBadRequest, presenters.Error(fmt.Errorf("invalid request: %w", err)))
			return
		}
		if req.Balance != nil && *req.Balance < 0 {
			ctx.JSON(http.StatusBadRequest, presenters.Error(repository.ErrorNegativeBalance))
			return
		}
	}

	user, err := a.UserService.CreateUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, repository.ErrorUserAlreadyExist) {
			ctx.JSON(http.StatusConflict, presenters.Error(err))
			return
		}
		ctx.JSON(http.StatusInternalServerError, presenters.Error(fmt.Errorf("create user failed: %w", err)))
		return
	}

	if req.Nickname != nil || req.Balance != nil {
		update := &domain.User{
			NickName: user.NickName,
			Balance:  user.Balance,
		}
		if req.Nickname != nil {
			nickname, err := normalizeNickname(*req.Nickname)
			if err != nil {
				ctx.JSON(http.StatusBadRequest, presenters.Error(err))
				return
			}
			update.NickName = nickname
		}
		if req.Balance != nil {
			update.Balance = *req.Balance
		}
		user, err = a.UserService.UpdateUserByID(ctx, userID, update)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, presenters.Error(fmt.Errorf("apply user payload failed: %w", err)))
			return
		}
	}

	ctx.JSON(http.StatusCreated, presenters.UserResponse{
		UserID:   user.ID,
		Nickname: user.NickName,
		Balance:  user.Balance,
		Rating:   user.Rating,
		Rank:     user.Rank,
	})
}

// Update user by id
// @Summary Update current user
// @Description Updates the current user nickname.
// @Tags User
// @Accept json
// @Produce json
// @Param X-Authenticated-User header string true "Authenticated user id"
// @Param Authorization header string true "Authorization Token"
// @Param request body presenters.UserUpdateRequest true "Nickname update payload"
// @Success 200 {object} presenters.UserResponse
// @Failure 400 {object} presenters.ErrorResponse
// @Failure 401 {object} presenters.ErrorResponse
// @Failure 404 {object} presenters.ErrorResponse
// @Failure 500 {object} presenters.ErrorResponse
// @Router /user [put]
func UpdateUserByID(ctx *gin.Context, a *app.App) {
	userID, err := authenticatedUserID(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, presenters.Error(err))
		return
	}

	var req presenters.UserUpdateRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, presenters.Error(fmt.Errorf("invalid request: %w", err)))
		return
	}
	if req.Nickname == nil {
		ctx.JSON(http.StatusBadRequest, presenters.Error(fmt.Errorf("nickname is required")))
		return
	}

	nickname, err := normalizeNickname(*req.Nickname)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, presenters.Error(err))
		return
	}

	user, err := a.UserService.UpdateUserByID(ctx, userID, &domain.User{
		NickName: nickname,
	})
	if err != nil {
		if errors.Is(err, repository.ErrorUserNotFound) {
			ctx.JSON(http.StatusNotFound, presenters.Error(err))
			return
		}
		ctx.JSON(http.StatusInternalServerError, presenters.Error(fmt.Errorf("update user failed: %w", err)))
		return
	}

	respondUser(ctx, user)
}

// Delete user by id
// @Summary Delete current user
// @Description Deletes the current user record from user_service.
// @Tags User
// @Accept json
// @Produce json
// @Param X-Authenticated-User header string true "Authenticated user id"
// @Param Authorization header string true "Authorization Token"
// @Success 204
// @Failure 401 {object} presenters.ErrorResponse
// @Failure 404 {object} presenters.ErrorResponse
// @Failure 500 {object} presenters.ErrorResponse
// @Router /user [delete]
func DeleteUserByID(ctx *gin.Context, a *app.App) {
	userID, err := authenticatedUserID(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, presenters.Error(err))
		return
	}

	err = a.UserService.DeleteUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, repository.ErrorUserNotFound) {
			ctx.JSON(http.StatusNotFound, presenters.Error(err))
			return
		}
		ctx.JSON(http.StatusInternalServerError, presenters.Error(fmt.Errorf("delete user failed: %w", err)))
		return
	}

	ctx.Status(http.StatusNoContent)
}

// Update user balance by id
// @Summary Update current user balance
// @Description Applies a relative delta to the current user balance. Use positive values to credit and negative values to debit.
// @Tags User
// @Accept json
// @Produce json
// @Param X-Authenticated-User header string true "Authenticated user id"
// @Param Authorization header string true "Authorization Token"
// @Param request body presenters.UserBalanceUpdateRequest true "Balance delta payload"
// @Success 200 {object} presenters.UserResponse
// @Failure 400 {object} presenters.ErrorResponse
// @Failure 401 {object} presenters.ErrorResponse
// @Failure 404 {object} presenters.ErrorResponse
// @Failure 500 {object} presenters.ErrorResponse
// @Router /user/balance [put]
func UpdateUserBalanceByID(ctx *gin.Context, a *app.App) {
	userID, err := authenticatedUserID(ctx)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, presenters.Error(err))
		return
	}

	var req presenters.UserBalanceUpdateRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, presenters.Error(fmt.Errorf("invalid request: %w", err)))
		return
	}

	user, err := a.UserService.UpdateUserBalance(ctx, req.Delta, userID)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrorUserNotFound):
			ctx.JSON(http.StatusNotFound, presenters.Error(err))
		case errors.Is(err, repository.ErrorNegativeBalance):
			ctx.JSON(http.StatusBadRequest, presenters.Error(err))
		default:
			ctx.JSON(http.StatusInternalServerError, presenters.Error(fmt.Errorf("update balance failed: %w", err)))
		}
		return
	}

	respondUser(ctx, user)
}
