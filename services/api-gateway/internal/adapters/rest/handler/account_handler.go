package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/danila-kuryakin/banking_mini_cores/services/api-gateway/internal/adapters/rest/handler/dto"
	"github.com/danila-kuryakin/banking_mini_cores/services/api-gateway/internal/app/service"
	"github.com/danila-kuryakin/banking_mini_cores/services/api-gateway/internal/domain"
	accountv1 "github.com/danila-kuryakin/banking_mini_cores/services/api-gateway/internal/pb/gen/account/v1"
)

type AccountHandler struct {
	service *service.Service
}

func NewAccountHandler(service *service.Service) *AccountHandler {
	return &AccountHandler{service: service}
}

// OpenAccount godoc
//
//	@Summary		Открыть счёт
//	@Description	Счёт открывается только активному клиенту (прошёл KYC) и только в валюте из списка сервиса (RUB, USD, EUR). Без валюты - RUB. Не больше трёх незакрытых счетов в одной валюте.
//	@Tags			accounts
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			user_id	path		string				true	"UUID пользователя или me - текущий пользователь"
//	@Param			request	body		dto.OpenAccount		false	"Валюта счёта"
//	@Success		201		{object}	dto.Account			"Счёт открыт"
//	@Failure		400		{object}	dto.ErrorResponse	"Валюта не поддерживается, клиент не активен или превышен лимит"
//	@Failure		401		{object}	dto.ErrorResponse	"Отсутствует или недействителен access-токен"
//	@Failure		403		{object}	dto.ErrorResponse	"Чужой клиент"
//	@Failure		404		{object}	dto.ErrorResponse	"Клиент не найден"
//	@Router			/customer/{user_id}/accounts [post]
func (h AccountHandler) OpenAccount(c *gin.Context) {
	var req dto.OpenAccount

	if c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(&req); err != nil {
			writeBindError(c, err)

			return
		}
	}

	account, err := h.service.Account.Open(c.Request.Context(), targetUserID(c), req.Currency)
	if err != nil {
		writeError(c, err)

		return
	}

	c.JSON(http.StatusCreated, accountResponse(account))
}

// ListAccounts godoc
//
//	@Summary		Счета клиента
//	@Description	Счета от новых к старым: последний открытый - первым.
//	@Tags			accounts
//	@Produce		json
//	@Security		BearerAuth
//	@Param			user_id	path		string				true	"UUID пользователя или me - текущий пользователь"
//	@Param			limit	query		int					false	"Размер страницы"
//	@Param			offset	query		int					false	"Смещение"
//	@Success		200		{object}	dto.ListAccounts	"Счета"
//	@Failure		400		{object}	dto.ErrorResponse	"Некорректная пагинация"
//	@Failure		401		{object}	dto.ErrorResponse	"Отсутствует или недействителен access-токен"
//	@Failure		403		{object}	dto.ErrorResponse	"Чужой клиент"
//	@Router			/customer/{user_id}/accounts [get]
func (h AccountHandler) ListAccounts(c *gin.Context) {
	var query dto.ListAccountsQuery

	if err := c.ShouldBindQuery(&query); err != nil {
		writeBindError(c, err)

		return
	}

	accounts, err := h.service.Account.List(c.Request.Context(), &accountv1.ListAccountsRequest{
		UserId: targetUserID(c),
		Limit:  query.Limit,
		Offset: query.Offset,
	})
	if err != nil {
		writeError(c, err)

		return
	}

	items := make([]dto.Account, 0, len(accounts))
	for _, account := range accounts {
		items = append(items, accountResponse(account))
	}

	c.JSON(http.StatusOK, dto.ListAccounts{Accounts: items})
}

// GetAccount godoc
//
//	@Summary		Счёт клиента
//	@Tags			accounts
//	@Produce		json
//	@Security		BearerAuth
//	@Param			user_id		path		string				true	"UUID пользователя или me - текущий пользователь"
//	@Param			account_id	path		string				true	"UUID счёта"
//	@Success		200			{object}	dto.Account			"Счёт"
//	@Failure		401			{object}	dto.ErrorResponse	"Отсутствует или недействителен access-токен"
//	@Failure		403			{object}	dto.ErrorResponse	"Чужой клиент"
//	@Failure		404			{object}	dto.ErrorResponse	"Счёт не найден у этого клиента"
//	@Router			/customer/{user_id}/accounts/{account_id} [get]
func (h AccountHandler) GetAccount(c *gin.Context) {
	account, err := h.service.Account.Get(c.Request.Context(), targetUserID(c), c.Param(domain.ACCOUNT_ID_PARAM))
	if err != nil {
		writeError(c, err)

		return
	}

	c.JSON(http.StatusOK, accountResponse(account))
}

// GetBalance godoc
//
//	@Summary		Баланс счёта
//	@Description	Баланс хранит ledger-service, account-service его только проксирует. Сумма в минорных единицах (копейки, центы).
//	@Tags			accounts
//	@Produce		json
//	@Security		BearerAuth
//	@Param			user_id		path		string				true	"UUID пользователя или me - текущий пользователь"
//	@Param			account_id	path		string				true	"UUID счёта"
//	@Success		200			{object}	dto.Balance			"Баланс"
//	@Failure		401			{object}	dto.ErrorResponse	"Отсутствует или недействителен access-токен"
//	@Failure		403			{object}	dto.ErrorResponse	"Чужой клиент"
//	@Failure		404			{object}	dto.ErrorResponse	"Счёт не найден у этого клиента"
//	@Router			/customer/{user_id}/accounts/{account_id}/balance [get]
func (h AccountHandler) GetBalance(c *gin.Context) {
	balance, err := h.service.Account.Balance(c.Request.Context(), targetUserID(c), c.Param(domain.ACCOUNT_ID_PARAM))
	if err != nil {
		writeError(c, err)

		return
	}

	c.JSON(http.StatusOK, dto.Balance{
		AccountID: balance.AccountId,
		Amount:    balance.Amount,
		Currency:  balance.Currency,
	})
}

// CloseAccount godoc
//
//	@Summary		Закрыть счёт
//	@Description	Закрыть можно только активный счёт с нулевым балансом. Закрытый счёт снова не открывается.
//	@Tags			accounts
//	@Produce		json
//	@Security		BearerAuth
//	@Param			user_id		path		string				true	"UUID пользователя или me - текущий пользователь"
//	@Param			account_id	path		string				true	"UUID счёта"
//	@Success		200			{object}	dto.Account			"Счёт закрыт"
//	@Failure		400			{object}	dto.ErrorResponse	"Баланс не нулевой или счёт заблокирован"
//	@Failure		401			{object}	dto.ErrorResponse	"Отсутствует или недействителен access-токен"
//	@Failure		403			{object}	dto.ErrorResponse	"Чужой клиент"
//	@Failure		404			{object}	dto.ErrorResponse	"Счёт не найден у этого клиента"
//	@Router			/customer/{user_id}/accounts/{account_id}/close [post]
func (h AccountHandler) CloseAccount(c *gin.Context) {
	account, err := h.service.Account.Close(c.Request.Context(), targetUserID(c), c.Param(domain.ACCOUNT_ID_PARAM))
	if err != nil {
		writeError(c, err)

		return
	}

	c.JSON(http.StatusOK, accountResponse(account))
}

// BlockAccount godoc
//
//	@Summary		Заблокировать счёт
//	@Description	Блокировка с обязательной причиной. Заблокированный счёт не участвует в переводах и пополнениях.
//	@Tags			accounts
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			account_id	path		string				true	"UUID счёта"
//	@Param			request		body		dto.BlockAccount	true	"Причина блокировки"
//	@Success		200			{object}	dto.Account			"Счёт заблокирован"
//	@Failure		400			{object}	dto.ErrorResponse	"Причина не указана или счёт закрыт"
//	@Failure		401			{object}	dto.ErrorResponse	"Отсутствует или недействителен access-токен"
//	@Failure		403			{object}	dto.ErrorResponse	"Недостаточно прав"
//	@Failure		404			{object}	dto.ErrorResponse	"Счёт не найден"
//	@Router			/accounts/{account_id}/block [post]
func (h AccountHandler) BlockAccount(c *gin.Context) {
	var req dto.BlockAccount

	if err := c.ShouldBindJSON(&req); err != nil {
		writeBindError(c, err)

		return
	}

	account, err := h.service.Account.Block(c.Request.Context(), c.Param(domain.ACCOUNT_ID_PARAM), actorID(c), req.Reason)
	if err != nil {
		writeError(c, err)

		return
	}

	c.JSON(http.StatusOK, accountResponse(account))
}

// UnblockAccount godoc
//
//	@Summary		Разблокировать счёт
//	@Tags			accounts
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			account_id	path		string				true	"UUID счёта"
//	@Param			request		body		dto.UnblockAccount	false	"Комментарий"
//	@Success		200			{object}	dto.Account			"Счёт снова активен"
//	@Failure		400			{object}	dto.ErrorResponse	"Счёт закрыт"
//	@Failure		401			{object}	dto.ErrorResponse	"Отсутствует или недействителен access-токен"
//	@Failure		403			{object}	dto.ErrorResponse	"Недостаточно прав"
//	@Failure		404			{object}	dto.ErrorResponse	"Счёт не найден"
//	@Router			/accounts/{account_id}/unblock [post]
func (h AccountHandler) UnblockAccount(c *gin.Context) {
	var req dto.UnblockAccount

	if c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(&req); err != nil {
			writeBindError(c, err)

			return
		}
	}

	account, err := h.service.Account.Unblock(c.Request.Context(), c.Param(domain.ACCOUNT_ID_PARAM), actorID(c), req.Comment)
	if err != nil {
		writeError(c, err)

		return
	}

	c.JSON(http.StatusOK, accountResponse(account))
}

func accountResponse(account *accountv1.Account) dto.Account {
	return dto.Account{
		ID:              account.Id,
		UserID:          account.UserId,
		Number:          account.Number,
		Currency:        account.Currency,
		Status:          enumName(account.Status.String(), domain.ACCOUNT_STATUS_ENUM_PREFIX),
		StatusChangedAt: account.StatusChangedAt.AsTime(),
		CreatedAt:       account.CreatedAt.AsTime(),
		UpdatedAt:       account.UpdatedAt.AsTime(),
	}
}

// ListStatusHistory godoc
//
//	@Summary		История статусов счёта
//	@Description	Все смены статуса счёта, от новых к старым. Первая запись без from_status - открытие счёта. Пустой actor_id - действие владельца или системы. Только для сотрудников.
//	@Tags			accounts
//	@Produce		json
//	@Security		BearerAuth
//	@Param			account_id	path		string					true	"UUID счёта"
//	@Param			limit		query		int						false	"Размер страницы"
//	@Param			offset		query		int						false	"Смещение"
//	@Success		200			{object}	dto.ListStatusHistory	"История статусов"
//	@Failure		400			{object}	dto.ErrorResponse		"Некорректная пагинация"
//	@Failure		401			{object}	dto.ErrorResponse		"Отсутствует или недействителен access-токен"
//	@Failure		403			{object}	dto.ErrorResponse		"Недостаточно прав"
//	@Failure		404			{object}	dto.ErrorResponse		"Счёт не найден"
//	@Router			/accounts/{account_id}/history [get]
func (h AccountHandler) ListStatusHistory(c *gin.Context) {
	var query dto.ListStatusHistoryQuery

	if err := c.ShouldBindQuery(&query); err != nil {
		writeBindError(c, err)

		return
	}

	changes, err := h.service.Account.History(c.Request.Context(), &accountv1.ListStatusHistoryRequest{
		AccountId: c.Param(domain.ACCOUNT_ID_PARAM),
		Limit:     query.Limit,
		Offset:    query.Offset,
	})
	if err != nil {
		writeError(c, err)

		return
	}

	items := make([]dto.StatusChange, 0, len(changes))
	for _, change := range changes {
		items = append(items, dto.StatusChange{
			ID:         change.Id,
			AccountID:  change.AccountId,
			FromStatus: enumName(change.FromStatus.String(), domain.ACCOUNT_STATUS_ENUM_PREFIX),
			ToStatus:   enumName(change.ToStatus.String(), domain.ACCOUNT_STATUS_ENUM_PREFIX),
			ActorID:    change.ActorId,
			Reason:     change.Reason,
			CreatedAt:  change.CreatedAt.AsTime(),
		})
	}

	c.JSON(http.StatusOK, dto.ListStatusHistory{Changes: items})
}
