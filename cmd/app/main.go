package main

import (
	"context"
	"log"
	"test/internal/config"
	bRepo "test/internal/core/balances/repository"
	"test/internal/core/withdrawal/api/v1"
	wRepo "test/internal/core/withdrawal/repository"
	wSvc "test/internal/core/withdrawal/service"
	"test/internal/postgres"
	"test/internal/server"
)

func main() {
	app, err := newApp()
	if err != nil {
		log.Fatal(err)
	}

	app.Start()
}

func newApp() (*server.Server, error) {
	cfg := config.Load()

	ctx := context.Background()

	pg, err := postgres.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, err
	}
	bRepo := bRepo.NewBalanceRepo()
	wRepo := wRepo.NewWithdrawalRepo()
	wSvc := wSvc.NewWithdrawalService(wRepo, bRepo, pg)
	wHandler := api.NewWithdrawalHandler(wSvc)

	s := server.NewServer(cfg.Addr, cfg.AuthToken, wHandler)

	return s, nil
}
