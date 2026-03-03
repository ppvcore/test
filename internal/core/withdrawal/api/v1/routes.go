package api

import (
	"test/internal/middleware"

	"github.com/gin-gonic/gin"
)

func SetupRoutesWindrawal(r *gin.Engine, h *WithdrawalHandler, authToken string) *gin.RouterGroup {
	api := r.Group("v1/withdrawals")
	{
		api.Use(middleware.Auth(authToken))

		api.POST("", h.CreateWithdrawal)
		api.GET("/:id", h.GetWithdrawal)
	}

	return api
}
