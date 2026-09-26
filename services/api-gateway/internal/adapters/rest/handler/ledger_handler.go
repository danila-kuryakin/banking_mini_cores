package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/danila-kuryakin/banking_mini_cores/services/api-gateway/internal/adapters/rest/handler/dto"
	"github.com/danila-kuryakin/banking_mini_cores/services/api-gateway/internal/app/service"
	"github.com/danila-kuryakin/banking_mini_cores/services/api-gateway/internal/domain"
	ledgerv1 "github.com/danila-kuryakin/banking_mini_cores/services/api-gateway/internal/pb/gen/ledger/v1"
)

type LedgerHandler struct {
	service *service.Service
}

func NewLedgerHandler(service *service.Service) *LedgerHandler {
	return &LedgerHandler{service: service}
}

// Transfer godoc
//
//	@Summary		Перевод между счетами
//	@Description	Списывает со своего счёта и зачисляет на счёт получателя в той же валюте. Оба счёта должны быть активны. Сумма в минорных единицах. При нехватке средств перевод сохраняется со статусом declined и возвращается 400 с его id в тексте ошибки.
//	@Tags			ledger
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			user_id	path		string				true	"UUID пользователя или me - владелец счёта-источника"
//	@Param			request	body		dto.Transfer		true	"Перевод"
//	@Success		200		{object}	dto.Transaction		"Перевод проведён"
//	@Failure		400		{object}	dto.ErrorResponse	"Некорректная сумма или валюта, счёт не активен, не хватает средств"
//	@Failure		401		{object}	dto.ErrorResponse	"Отсутствует или недействителен access-токен"
//	@Failure		403		{object}	dto.ErrorResponse	"Счёт-источник принадлежит другому клиенту"
//	@Failure		404		{object}	dto.ErrorResponse	"Счёт не найден"
//	@Router			/customer/{user_id}/transfers [post]
func (h LedgerHandler) Transfer(c *gin.Context) {
	var req dto.Transfer

	if err := c.ShouldBindJSON(&req); err != nil {
		writeBindError(c, err)

		return
	}

	tx, err := h.service.Ledger.Transfer(c.Request.Context(), &ledgerv1.TransferRequest{
		InitiatorId:   targetUserID(c),
		FromAccountId: req.FromAccountID,
		ToAccountId:   req.ToAccountID,
		Amount:        req.Amount,
		Currency:      req.Currency,
	})
	if err != nil {
		writeError(c, err)

		return
	}

	c.JSON(http.StatusOK, transactionResponse(tx))
}

// ListTransactions godoc
//
//	@Summary		Выписка по счёту
//	@Description	Исходящие переводы (включая отклонённые) и проведённые входящие, от новых к старым.
//	@Tags			ledger
//	@Produce		json
//	@Security		BearerAuth
//	@Param			user_id		path		string					true	"UUID пользователя или me - текущий пользователь"
//	@Param			account_id	path		string					true	"UUID счёта"
//	@Param			limit		query		int						false	"Размер страницы"
//	@Param			offset		query		int						false	"Смещение"
//	@Success		200			{object}	dto.ListTransactions	"Выписка"
//	@Failure		400			{object}	dto.ErrorResponse		"Некорректная пагинация"
//	@Failure		401			{object}	dto.ErrorResponse		"Отсутствует или недействителен access-токен"
//	@Failure		403			{object}	dto.ErrorResponse		"Чужой клиент"
//	@Failure		404			{object}	dto.ErrorResponse		"Счёт не найден у этого клиента"
//	@Router			/customer/{user_id}/accounts/{account_id}/transactions [get]
func (h LedgerHandler) ListTransactions(c *gin.Context) {
	var query dto.ListTransactionsQuery

	if err := c.ShouldBindQuery(&query); err != nil {
		writeBindError(c, err)

		return
	}

	transactions, err := h.service.Ledger.ListTransactions(c.Request.Context(), &ledgerv1.ListTransactionsRequest{
		UserId:    targetUserID(c),
		AccountId: c.Param(domain.ACCOUNT_ID_PARAM),
		Limit:     query.Limit,
		Offset:    query.Offset,
	})
	if err != nil {
		writeError(c, err)

		return
	}

	items := make([]dto.Transaction, 0, len(transactions))
	for _, tx := range transactions {
		items = append(items, transactionResponse(tx))
	}

	c.JSON(http.StatusOK, dto.ListTransactions{Transactions: items})
}

// Deposit godoc
//
//	@Summary		Пополнить счёт из кассы
//	@Description	Пополнение для тестов: дебет кассового счёта леджера, кредит счёта клиента. Сумма в минорных единицах.
//	@Tags			ledger
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			account_id	path		string				true	"UUID счёта"
//	@Param			request		body		dto.Deposit			true	"Сумма и валюта"
//	@Success		200			{object}	dto.Transaction		"Пополнение проведено"
//	@Failure		400			{object}	dto.ErrorResponse	"Некорректная сумма, валюта не совпадает или счёт не активен"
//	@Failure		401			{object}	dto.ErrorResponse	"Отсутствует или недействителен access-токен"
//	@Failure		403			{object}	dto.ErrorResponse	"Недостаточно прав"
//	@Failure		404			{object}	dto.ErrorResponse	"Счёт не найден"
//	@Router			/accounts/{account_id}/deposit [post]
func (h LedgerHandler) Deposit(c *gin.Context) {
	var req dto.Deposit

	if err := c.ShouldBindJSON(&req); err != nil {
		writeBindError(c, err)

		return
	}

	tx, err := h.service.Ledger.Deposit(c.Request.Context(), &ledgerv1.DepositRequest{
		AccountId: c.Param(domain.ACCOUNT_ID_PARAM),
		Amount:    req.Amount,
		Currency:  req.Currency,
		OfficerId: actorID(c),
	})
	if err != nil {
		writeError(c, err)

		return
	}

	c.JSON(http.StatusOK, transactionResponse(tx))
}

// GetTransaction godoc
//
//	@Summary		Транзакция платежей
//	@Description	Для сотрудников: транзакция вместе с дебетовой и кредитовой платежами.
//	@Tags			ledger
//	@Produce		json
//	@Security		BearerAuth
//	@Param			transaction_id	path		string				true	"UUID транзакции"
//	@Success		200				{object}	dto.Transaction		"Транзакция"
//	@Failure		401				{object}	dto.ErrorResponse	"Отсутствует или недействителен access-токен"
//	@Failure		403				{object}	dto.ErrorResponse	"Недостаточно прав"
//	@Failure		404				{object}	dto.ErrorResponse	"Транзакция не найдена"
//	@Router			/transactions/{transaction_id} [get]
func (h LedgerHandler) GetTransaction(c *gin.Context) {
	tx, err := h.service.Ledger.GetTransaction(c.Request.Context(), c.Param(domain.TRANSACTION_ID_PARAM))
	if err != nil {
		writeError(c, err)

		return
	}

	c.JSON(http.StatusOK, transactionResponse(tx))
}

func transactionResponse(tx *ledgerv1.Transaction) dto.Transaction {
	out := dto.Transaction{
		ID:            tx.Id,
		Type:          enumName(tx.Type.String(), domain.TRANSACTION_TYPE_ENUM_PREFIX),
		Status:        enumName(tx.Status.String(), domain.TRANSACTION_STATUS_ENUM_PREFIX),
		FromAccountID: tx.FromAccountId,
		ToAccountID:   tx.ToAccountId,
		Amount:        tx.Amount,
		Currency:      tx.Currency,
		InitiatorID:   tx.InitiatorId,
		DeclineReason: tx.DeclineReason,
		CreatedAt:     tx.CreatedAt.AsTime(),
	}

	for _, entry := range tx.Entries {
		out.Entries = append(out.Entries, dto.Entry{
			ID:        entry.Id,
			AccountID: entry.AccountId,
			Direction: enumName(entry.Direction.String(), domain.ENTRY_DIRECTION_ENUM_PREFIX),
			Amount:    entry.Amount,
			Currency:  entry.Currency,
		})
	}

	return out
}
