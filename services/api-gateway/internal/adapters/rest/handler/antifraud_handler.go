package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/danila-kuryakin/banking_mini_cores/services/api-gateway/internal/adapters/rest/handler/dto"
	"github.com/danila-kuryakin/banking_mini_cores/services/api-gateway/internal/app/service"
	"github.com/danila-kuryakin/banking_mini_cores/services/api-gateway/internal/domain"
	antifraudv1 "github.com/danila-kuryakin/banking_mini_cores/services/api-gateway/internal/pb/gen/antifraud/v1"
)

type AntifraudHandler struct {
	service *service.Service
}

func NewAntifraudHandler(service *service.Service) *AntifraudHandler {
	return &AntifraudHandler{service: service}
}

// ListRules godoc
//
//	@Summary		Правила антифрода
//	@Description	Все правила с порогами и признаком включения. Суммы в минорных единицах по валютам.
//	@Tags			antifraud
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	dto.ListRules		"Правила"
//	@Failure		401	{object}	dto.ErrorResponse	"Отсутствует или недействителен access-токен"
//	@Failure		403	{object}	dto.ErrorResponse	"Недостаточно прав"
//	@Router			/antifraud/rules [get]
func (h AntifraudHandler) ListRules(c *gin.Context) {
	rules, err := h.service.Antifraud.ListRules(c.Request.Context())
	if err != nil {
		writeError(c, err)

		return
	}

	items := make([]dto.Rule, 0, len(rules))
	for _, rule := range rules {
		items = append(items, ruleResponse(rule))
	}

	c.JSON(http.StatusOK, dto.ListRules{Rules: items})
}

// UpdateRule godoc
//
//	@Summary		Изменить правило антифрода
//	@Description	Включает, выключает правило или меняет пороги. Не переданное поле не меняется, переданные params заменяют пороги целиком. single_amount_limit и daily_amount_limit читают limits по валютам из конфига, velocity - max_count и window_seconds, у same_account порогов нет. Действует со следующей проверки.
//	@Tags			antifraud
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			code	path		string				true	"Код правила"	Enums(single_amount_limit, velocity, daily_amount_limit, same_account)
//	@Param			request	body		dto.UpdateRule		true	"Изменения"
//	@Success		200		{object}	dto.Rule			"Правило после изменения"
//	@Failure		400		{object}	dto.ErrorResponse	"Пустой запрос или пороги не подходят правилу"
//	@Failure		401		{object}	dto.ErrorResponse	"Отсутствует или недействителен access-токен"
//	@Failure		403		{object}	dto.ErrorResponse	"Недостаточно прав"
//	@Failure		404		{object}	dto.ErrorResponse	"Правило не найдено"
//	@Router			/antifraud/rules/{code} [patch]
func (h AntifraudHandler) UpdateRule(c *gin.Context) {
	var req dto.UpdateRule

	if err := c.ShouldBindJSON(&req); err != nil {
		writeBindError(c, err)

		return
	}

	in := &antifraudv1.UpdateRuleRequest{
		Code:      c.Param(domain.RULE_CODE_PARAM),
		Enabled:   req.Enabled,
		OfficerId: actorID(c),
	}

	if req.Params != nil {
		in.Params = &antifraudv1.RuleParams{
			Limits:        req.Params.Limits,
			MaxCount:      req.Params.MaxCount,
			WindowSeconds: req.Params.WindowSeconds,
		}
	}

	rule, err := h.service.Antifraud.UpdateRule(c.Request.Context(), in)
	if err != nil {
		writeError(c, err)

		return
	}

	c.JSON(http.StatusOK, ruleResponse(rule))
}

func ruleResponse(rule *antifraudv1.Rule) dto.Rule {
	out := dto.Rule{
		Code:        rule.Code,
		Description: rule.Description,
		Enabled:     rule.Enabled,
		UpdatedBy:   rule.UpdatedBy,
		UpdatedAt:   rule.UpdatedAt.AsTime(),
	}

	if rule.Params != nil {
		out.Params = dto.RuleParams{
			Limits:        rule.Params.Limits,
			MaxCount:      rule.Params.MaxCount,
			WindowSeconds: rule.Params.WindowSeconds,
		}
	}

	return out
}
