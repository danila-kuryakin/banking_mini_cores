package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/danila-kuryakin/banking_mini_cores/services/api-gateway/internal/adapters/rest/handler/dto"
	"github.com/danila-kuryakin/banking_mini_cores/services/api-gateway/internal/app/service"
	"github.com/danila-kuryakin/banking_mini_cores/services/api-gateway/internal/domain"
	kycv1 "github.com/danila-kuryakin/banking_mini_cores/services/api-gateway/internal/pb/gen/kyc/v1"
)

type KycHandler struct {
	service *service.Service
}

func NewKycHandler(service *service.Service) *KycHandler {
	return &KycHandler{service: service}
}

// SubmitApplication godoc
//
//	@Summary		Подать заявку на верификацию
//	@Description	Требует заполненной анкеты и подтверждённых документов (паспорт и селфи). Если по заявке запрашивали документы, в очередь возвращается та же заявка. Проверки идут в фоне, ответ приходит сразу.
//	@Tags			kyc
//	@Produce		json
//	@Security		BearerAuth
//	@Param			user_id	path		string				true	"UUID пользователя или me - текущий пользователь"
//	@Success		200		{object}	dto.Application		"Заявка подана"
//	@Failure		400		{object}	dto.ErrorResponse	"Анкета не заполнена или не хватает документов"
//	@Failure		401		{object}	dto.ErrorResponse	"Отсутствует или недействителен access-токен"
//	@Failure		403		{object}	dto.ErrorResponse	"Заявка принадлежит другому пользователю"
//	@Failure		409		{object}	dto.ErrorResponse	"У клиента уже есть активная заявка"
//	@Router			/customer/{user_id}/kyc [post]
func (h KycHandler) SubmitApplication(c *gin.Context) {
	app, err := h.service.Kyc.SubmitApplication(c.Request.Context(), targetUserID(c))
	if err != nil {
		writeError(c, err)

		return
	}

	c.JSON(http.StatusOK, applicationResponse(app))
}

// GetApplication godoc
//
//	@Summary		Последняя заявка клиента
//	@Description	Отдаёт последнюю заявку вместе с результатами авто-проверок и историей переходов.
//	@Tags			kyc
//	@Produce		json
//	@Security		BearerAuth
//	@Param			user_id	path		string				true	"UUID пользователя или me - текущий пользователь"
//	@Success		200		{object}	dto.Application		"Заявка"
//	@Failure		401		{object}	dto.ErrorResponse	"Отсутствует или недействителен access-токен"
//	@Failure		403		{object}	dto.ErrorResponse	"Заявка принадлежит другому пользователю"
//	@Failure		404		{object}	dto.ErrorResponse	"Заявок у клиента нет"
//	@Router			/customer/{user_id}/kyc [get]
func (h KycHandler) GetApplication(c *gin.Context) {
	app, err := h.service.Kyc.GetByUser(c.Request.Context(), targetUserID(c))
	if err != nil {
		writeError(c, err)

		return
	}

	c.JSON(http.StatusOK, applicationResponse(app))
}

// ListApplications godoc
//
//	@Summary		Очередь заявок
//	@Description	Очередь для сотрудника: фильтры по статусу и по офицеру, взявшему заявку в работу. Результаты проверок и история в листинге не отдаются.
//	@Tags			kyc
//	@Produce		json
//	@Security		BearerAuth
//	@Param			status			query		string					false	"Фильтр по статусу"	Enums(submitted, auto_checks, auto_rejected, in_review, docs_requested, approved, rejected)
//	@Param			reviewer_id		query		string					false	"UUID офицера"
//	@Param			limit			query		int						false	"Размер страницы"
//	@Param			offset			query		int						false	"Смещение"
//	@Success		200				{object}	dto.ListApplications	"Заявки"
//	@Failure		400				{object}	dto.ErrorResponse		"Неизвестный статус в фильтре"
//	@Failure		401				{object}	dto.ErrorResponse		"Отсутствует или недействителен access-токен"
//	@Failure		403				{object}	dto.ErrorResponse		"Недостаточно прав"
//	@Router			/kyc/applications [get]
func (h KycHandler) ListApplications(c *gin.Context) {
	var query dto.ListApplicationsQuery

	if err := c.ShouldBindQuery(&query); err != nil {
		writeBindError(c, err)

		return
	}

	req := &kycv1.ListApplicationsRequest{
		ReviewerId: query.Reviewer,
		Limit:      query.Limit,
		Offset:     query.Offset,
	}

	if query.Status != "" {
		appStatus, err := applicationStatusFromName(query.Status)
		if err != nil {
			writeBindError(c, err)

			return
		}

		req.Statuses = []kycv1.ApplicationStatus{appStatus}
	}

	apps, err := h.service.Kyc.ListApplications(c.Request.Context(), req)
	if err != nil {
		writeError(c, err)

		return
	}

	items := make([]dto.Application, 0, len(apps))
	for _, app := range apps {
		items = append(items, applicationResponse(app))
	}

	c.JSON(http.StatusOK, dto.ListApplications{Applications: items})
}

// GetApplicationByID godoc
//
//	@Summary		Заявка по идентификатору
//	@Description	Заявка вместе с результатами авто-проверок и историей переходов. Для сотрудников.
//	@Tags			kyc
//	@Produce		json
//	@Security		BearerAuth
//	@Param			application_id	path		string				true	"UUID заявки"
//	@Success		200				{object}	dto.Application		"Заявка"
//	@Failure		401				{object}	dto.ErrorResponse	"Отсутствует или недействителен access-токен"
//	@Failure		403				{object}	dto.ErrorResponse	"Недостаточно прав"
//	@Failure		404				{object}	dto.ErrorResponse	"Заявка не найдена"
//	@Router			/kyc/applications/{application_id} [get]
func (h KycHandler) GetApplicationByID(c *gin.Context) {
	app, err := h.service.Kyc.GetByID(c.Request.Context(), c.Param(domain.APPLICATION_ID_PARAM))
	if err != nil {
		writeError(c, err)

		return
	}

	c.JSON(http.StatusOK, applicationResponse(app))
}

// ClaimApplication godoc
//
//	@Summary		Взять заявку в работу
//	@Description	Закрепляет заявку за текущим сотрудником. Занятую заявку второй раз взять нельзя.
//	@Tags			kyc
//	@Produce		json
//	@Security		BearerAuth
//	@Param			application_id	path		string				true	"UUID заявки"
//	@Success		200				{object}	dto.Application		"Заявка закреплена"
//	@Failure		400				{object}	dto.ErrorResponse	"Заявка уже взята или не в том статусе"
//	@Failure		401				{object}	dto.ErrorResponse	"Отсутствует или недействителен access-токен"
//	@Failure		403				{object}	dto.ErrorResponse	"Недостаточно прав"
//	@Failure		404				{object}	dto.ErrorResponse	"Заявка не найдена"
//	@Router			/kyc/applications/{application_id}/claim [post]
func (h KycHandler) ClaimApplication(c *gin.Context) {
	app, err := h.service.Kyc.Claim(c.Request.Context(), c.Param(domain.APPLICATION_ID_PARAM), actorID(c))
	if err != nil {
		writeError(c, err)

		return
	}

	c.JSON(http.StatusOK, applicationResponse(app))
}

// ApproveApplication godoc
//
//	@Summary		Одобрить заявку
//	@Description	Одобряет заявку и переводит клиента в active.
//	@Tags			kyc
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			application_id	path		string					true	"UUID заявки"
//	@Param			request			body		dto.ApproveApplication	false	"Комментарий"
//	@Success		200				{object}	dto.Application			"Заявка одобрена"
//	@Failure		400				{object}	dto.ErrorResponse		"Заявка не в том статусе"
//	@Failure		401				{object}	dto.ErrorResponse		"Отсутствует или недействителен access-токен"
//	@Failure		403				{object}	dto.ErrorResponse		"Недостаточно прав"
//	@Failure		404				{object}	dto.ErrorResponse		"Заявка не найдена"
//	@Router			/kyc/applications/{application_id}/approve [post]
func (h KycHandler) ApproveApplication(c *gin.Context) {
	var req dto.ApproveApplication

	// Тело необязательно: одобрение без комментария - нормальный случай.
	if c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(&req); err != nil {
			writeBindError(c, err)

			return
		}
	}

	app, err := h.service.Kyc.Approve(c.Request.Context(), c.Param(domain.APPLICATION_ID_PARAM), actorID(c), req.Comment)
	if err != nil {
		writeError(c, err)

		return
	}

	c.JSON(http.StatusOK, applicationResponse(app))
}

// RejectApplication godoc
//
//	@Summary		Отказать по заявке
//	@Description	Отказ с обязательной причиной. Клиент уходит в rejected окончательно - в отличие от авто-отказа, который возвращает его в profile_filled.
//	@Tags			kyc
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			application_id	path		string					true	"UUID заявки"
//	@Param			request			body		dto.RejectApplication	true	"Причина отказа"
//	@Success		200				{object}	dto.Application			"Отказ зафиксирован"
//	@Failure		400				{object}	dto.ErrorResponse		"Причина не указана или заявка не в том статусе"
//	@Failure		401				{object}	dto.ErrorResponse		"Отсутствует или недействителен access-токен"
//	@Failure		403				{object}	dto.ErrorResponse		"Недостаточно прав"
//	@Failure		404				{object}	dto.ErrorResponse		"Заявка не найдена"
//	@Router			/kyc/applications/{application_id}/reject [post]
func (h KycHandler) RejectApplication(c *gin.Context) {
	var req dto.RejectApplication

	if err := c.ShouldBindJSON(&req); err != nil {
		writeBindError(c, err)

		return
	}

	app, err := h.service.Kyc.Reject(c.Request.Context(), c.Param(domain.APPLICATION_ID_PARAM), actorID(c), req.Reason)
	if err != nil {
		writeError(c, err)

		return
	}

	c.JSON(http.StatusOK, applicationResponse(app))
}

// RequestDocuments godoc
//
//	@Summary		Запросить документы
//	@Description	Возвращает заявку клиенту с просьбой догрузить документы указанных типов. Клиент остаётся на проверке; после загрузки он подаёт ту же заявку снова.
//	@Tags			kyc
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			application_id	path		string					true	"UUID заявки"
//	@Param			request			body		dto.RequestDocuments	true	"Типы документов и комментарий"
//	@Success		200				{object}	dto.Application			"Заявка возвращена клиенту"
//	@Failure		400				{object}	dto.ErrorResponse		"Неизвестный тип документа или заявка не в том статусе"
//	@Failure		401				{object}	dto.ErrorResponse		"Отсутствует или недействителен access-токен"
//	@Failure		403				{object}	dto.ErrorResponse		"Недостаточно прав"
//	@Failure		404				{object}	dto.ErrorResponse		"Заявка не найдена"
//	@Router			/kyc/applications/{application_id}/documents [post]
func (h KycHandler) RequestDocuments(c *gin.Context) {
	var req dto.RequestDocuments

	if err := c.ShouldBindJSON(&req); err != nil {
		writeBindError(c, err)

		return
	}

	types := make([]kycv1.DocumentType, 0, len(req.Types))

	for _, raw := range req.Types {
		docType, err := documentTypeFromName(raw)
		if err != nil {
			writeBindError(c, err)

			return
		}

		types = append(types, docType)
	}

	app, err := h.service.Kyc.RequestDocuments(c.Request.Context(), c.Param(domain.APPLICATION_ID_PARAM), actorID(c), types, req.Comment)
	if err != nil {
		writeError(c, err)

		return
	}

	c.JSON(http.StatusOK, applicationResponse(app))
}

// applicationResponse переводит заявку в DTO. Статусы, типы документов и
// результаты проверок отдаются доменными именами в нижнем регистре: наружу
// имена значений enum не уходят.
func applicationResponse(app *kycv1.Application) dto.Application {
	out := dto.Application{
		ID:             app.Id,
		UserID:         app.UserId,
		Status:         applicationStatusName(app.Status),
		ReviewerID:     app.ReviewerId,
		Reason:         app.Reason,
		RequestedTypes: documentTypeNames(app.RequestedTypes),
		SubmittedAt:    app.SubmittedAt.AsTime(),
		CreatedAt:      app.CreatedAt.AsTime(),
		UpdatedAt:      app.UpdatedAt.AsTime(),
		Checks:         checksResponse(app.Checks),
		Events:         eventsResponse(app.Events),
	}

	if app.DecidedAt != nil {
		decidedAt := app.DecidedAt.AsTime()
		out.DecidedAt = &decidedAt
	}

	return out
}

func checksResponse(checks []*kycv1.KycCheck) []dto.Check {
	out := make([]dto.Check, 0, len(checks))

	for _, check := range checks {
		out = append(out, dto.Check{
			ID:        check.Id,
			Type:      enumName(check.Type.String(), domain.CHECK_TYPE_ENUM_PREFIX),
			Result:    enumName(check.Result.String(), domain.CHECK_RESULT_ENUM_PREFIX),
			Details:   check.Details,
			CreatedAt: check.CreatedAt.AsTime(),
		})
	}

	return out
}

func eventsResponse(events []*kycv1.KycEvent) []dto.Event {
	out := make([]dto.Event, 0, len(events))

	for _, event := range events {
		out = append(out, dto.Event{
			ID:         event.Id,
			FromStatus: applicationStatusName(event.FromStatus),
			ToStatus:   applicationStatusName(event.ToStatus),
			ActorID:    event.ActorId,
			Comment:    event.Comment,
			CreatedAt:  event.CreatedAt.AsTime(),
		})
	}

	return out
}

func documentTypeNames(types []kycv1.DocumentType) []string {
	out := make([]string, 0, len(types))

	for _, docType := range types {
		out = append(out, enumName(docType.String(), domain.DOCUMENT_TYPE_ENUM_PREFIX))
	}

	return out
}

// enumName отрезает префикс имени значения enum и опускает регистр.
// UNSPECIFIED превращается в пустую строку: "статуса нет" в JSON лучше
// выглядит как отсутствующее поле, чем как слово "unspecified".
func enumName(value, prefix string) string {
	name := strings.ToLower(strings.TrimPrefix(value, prefix))
	if name == "unspecified" {
		return ""
	}

	return name
}

func applicationStatusName(status kycv1.ApplicationStatus) string {
	return enumName(status.String(), domain.APPLICATION_STATUS_ENUM_PREFIX)
}

func applicationStatusFromName(raw string) (kycv1.ApplicationStatus, error) {
	name := domain.APPLICATION_STATUS_ENUM_PREFIX + strings.ToUpper(strings.TrimSpace(raw))

	value, ok := kycv1.ApplicationStatus_value[name]
	if !ok || kycv1.ApplicationStatus(value) == kycv1.ApplicationStatus_APPLICATION_STATUS_UNSPECIFIED {
		return kycv1.ApplicationStatus_APPLICATION_STATUS_UNSPECIFIED, domain.ErrApplicationStatusUnknown
	}

	return kycv1.ApplicationStatus(value), nil
}

func documentTypeFromName(raw string) (kycv1.DocumentType, error) {
	name := domain.DOCUMENT_TYPE_ENUM_PREFIX + strings.ToUpper(strings.TrimSpace(raw))

	value, ok := kycv1.DocumentType_value[name]
	if !ok || kycv1.DocumentType(value) == kycv1.DocumentType_DOCUMENT_TYPE_UNSPECIFIED {
		return kycv1.DocumentType_DOCUMENT_TYPE_UNSPECIFIED, domain.ErrDocumentTypeUnknown
	}

	return kycv1.DocumentType(value), nil
}

// actorID отдаёт идентификатор сотрудника из токена: kyc-service прав не
// проверяет и ждёт инициатора решения полем запроса.
func actorID(c *gin.Context) string {
	actor, ok := ActorFrom(c)
	if !ok {
		return ""
	}

	return actor.ID
}
