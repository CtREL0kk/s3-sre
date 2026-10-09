package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"myS3/internal/domain"
	"myS3/internal/dto"
)

func respondJSON(c *gin.Context, status int, data any) {
	c.JSON(status, data)
}

func respondError(c *gin.Context, status int, code, message string) {
	c.JSON(status, dto.Error{Error: dto.ErrorDetail{Code: code, Message: message}})
}

func respondValidationError(c *gin.Context, err error) {
	respondError(c, http.StatusBadRequest, "validation_error", err.Error())
}

func respondDomainError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrInvalidCredentials):
		respondError(c, http.StatusUnauthorized, "invalid_credentials", "неверный логин или пароль")
	case errors.Is(err, domain.ErrUserAlreadyExists):
		respondError(c, http.StatusConflict, "user_exists", "пользователь уже существует")
	case errors.Is(err, domain.ErrInvalidToken):
		respondError(c, http.StatusUnauthorized, "invalid_token", "недействительный или просроченный токен")
	case errors.Is(err, domain.ErrNotFound):
		respondError(c, http.StatusNotFound, "not_found", "не найдено")
	case errors.Is(err, domain.ErrForbidden):
		respondError(c, http.StatusForbidden, "forbidden", "доступ запрещён")
	case errors.Is(err, domain.ErrNameConflict):
		respondError(c, http.StatusConflict, "name_conflict", "имя уже существует")
	case errors.Is(err, domain.ErrNotFolder):
		respondError(c, http.StatusBadRequest, "not_folder", "объект не является папкой")
	case errors.Is(err, domain.ErrNotFile):
		respondError(c, http.StatusBadRequest, "not_file", "объект не является файлом")
	case errors.Is(err, domain.ErrInvalidMove):
		respondError(c, http.StatusBadRequest, "invalid_move", "нельзя перенести объект в себя или своего потомка")
	case errors.Is(err, domain.ErrReadOnly):
		respondError(c, http.StatusForbidden, "readonly", "хранилище в режиме только для чтения")
	default:
		respondError(c, http.StatusInternalServerError, "internal", "внутренняя ошибка")
	}
}
