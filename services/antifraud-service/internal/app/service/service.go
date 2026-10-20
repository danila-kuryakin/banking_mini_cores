package service

import (
	"github.com/danila-kuryakin/banking_mini_cores/services/antifraud-service/internal/adapters/repository"
	"github.com/danila-kuryakin/banking_mini_cores/services/antifraud-service/internal/config"
	"github.com/danila-kuryakin/banking_mini_cores/services/antifraud-service/internal/domain/interfaces"
)

type Service struct {
	Checks interfaces.CheckService
	Rules  interfaces.RuleService
}

func NewService(repo *repository.Repository, cfg *config.Config) *Service {
	return &Service{
		Checks: NewCheckService(repo.Checks, repo.Rules),
		Rules:  NewRuleService(repo.Rules, cfg.CurrencySet),
	}
}
