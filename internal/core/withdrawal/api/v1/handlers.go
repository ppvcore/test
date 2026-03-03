package api

import (
	"net/http"

	"test/internal/core/withdrawal/dto"
	"test/internal/core/withdrawal/service"

	"github.com/gin-gonic/gin"
)

type WithdrawalHandler struct {
	service *service.WithdrawalService
}

func NewWithdrawalHandler(svc *service.WithdrawalService) *WithdrawalHandler {
	return &WithdrawalHandler{service: svc}
}

func (h *WithdrawalHandler) CreateWithdrawal(c *gin.Context) {
	var req dto.Request
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	if req.IdempotencyKey == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "idempotency_key required"})
		return
	}

	w, created, err := h.service.Create(c.Request.Context(), req)
	if err != nil {
		switch err {
		case service.ErrInvalidAmount:
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		case service.ErrInsufficientBalance:
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		case service.ErrInvalidPayload:
			c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		}
		return
	}

	if created {
		c.JSON(http.StatusCreated, w)
	} else {
		c.JSON(http.StatusOK, w)
	}
}

func (h *WithdrawalHandler) GetWithdrawal(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id required"})
		return
	}

	w, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}

	c.JSON(http.StatusOK, w)
}
