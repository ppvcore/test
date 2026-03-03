package server

import (
	"test/internal/core/withdrawal/api/v1"

	"github.com/gin-gonic/gin"
)

type Server struct {
	addr   string
	engine *gin.Engine
}

func NewServer(addr string, authToken string, handler *api.WithdrawalHandler) *Server {
	engine := gin.New()

	api.SetupRoutesWindrawal(engine, handler, authToken)

	return &Server{
		addr:   addr,
		engine: engine,
	}
}

func (s *Server) Start() error {
	return s.engine.Run(s.addr)
}
