package handler

import (
	"errors"
	"net/http"
	"time"

	"github.com/danila-kuryakin/banking_mini_cores/services/api-gateway/internal/domain"
	customerv1 "github.com/danila-kuryakin/banking_mini_cores/services/api-gateway/internal/pb/gen/customer/v1"
	"github.com/gin-gonic/gin"
	"google.golang.org/genproto/googleapis/type/date"

	"github.com/danila-kuryakin/banking_mini_cores/services/api-gateway/internal/adapters/rest/handler/dto"
	"github.com/danila-kuryakin/banking_mini_cores/services/api-gateway/internal/app/service"
)

type CustomerHandler struct {
	service *service.Service
}

func NewCustomerHandler(service *service.Service) *CustomerHandler {
	return &CustomerHandler{
		service: service,
	}
}

// GetCustomer godoc
//
//	@Summary		Карточка клиента
//	@Description	Возвращает клиента вместе с профилем и текущим статусом.
//	@Tags			customer
//	@Produce		json
//	@Security		BearerAuth
//	@Param			customer_id	path		string	true	"Идентификатор клиента (UUID)"	format(uuid)
//	@Success		200			{object}	dto.Customer		"Клиент найден"
//	@Failure		401			{object}	dto.ErrorResponse	"Отсутствует или недействителен access-токен"
//	@Failure		500			{object}	dto.ErrorResponse	"Клиент не найден или ошибка customer-service"
//	@Router			/customer/{customer_id} [get]
func (h CustomerHandler) GetCustomer(c *gin.Context) {
	id := c.Param("id")

	resp, err := h.service.Customer.GetCustomer(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Message: "StatusInternalServerError"})
		return
	}

	c.JSON(http.StatusOK, dto.Customer{
		ID:        resp.Id,
		UserID:    resp.UserId,
		Status:    resp.Status.String(),
		CreatedAt: resp.CreatedAt.AsTime(),
		UpdatedAt: resp.UpdatedAt.AsTime(),
		Profile: dto.Profile{
			FirstName:   resp.Profile.FirstName,
			LastName:    resp.Profile.LastName,
			BirthDate:   resp.Profile.BirthDate.String(),
			Citizenship: resp.Profile.Citizenship,
		},
	})
}

// UpdateProfile godoc
//
//	@Summary		Обновление профиля клиента
//	@Description	Записывает анкетные данные. Поле birth_date передаётся строкой в формате YYYY-MM-DD. Когда профиль заполнен целиком, клиент переходит в статус profile_filled; после начала KYC редактирование запрещено.
//	@Tags			customer
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			customer_id	path		string				true	"Идентификатор клиента (UUID)"	format(uuid)
//	@Param			request		body		dto.UpdateProfile	true	"Поля профиля"
//	@Success		200			{object}	dto.Customer		"Профиль обновлён"
//	@Failure		400			{object}	dto.ErrorResponse	"Некорректное тело запроса или профиль заблокирован"
//	@Failure		401			{object}	dto.ErrorResponse	"Отсутствует или недействителен access-токен"
//	@Failure		500			{object}	dto.ErrorResponse	"birth_date не в формате YYYY-MM-DD"
//	@Router			/customer/{customer_id} [post]
func (h CustomerHandler) UpdateProfile(c *gin.Context) {
	var req dto.UpdateProfile
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Message: "StatusBadRequest"})
		return
	}

	id := c.Param("id")

	birthDate, err := parseBirthDate(req.BirthDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Message: "StatusBadRequest"})
		return
	}

	out, err := h.service.Customer.UpdateProfile(c.Request.Context(), id, &customerv1.Profile{
		FirstName:   req.FirstName,
		LastName:    req.LastName,
		BirthDate:   birthDate,
		Citizenship: req.Citizenship,
		Phone:       req.Phone,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Message: "StatusBadRequest"})
		return
	}

	c.JSON(http.StatusOK, dto.Customer{
		ID:        out.Id,
		UserID:    out.UserId,
		Status:    out.Status.String(),
		CreatedAt: out.CreatedAt.AsTime(),
		UpdatedAt: out.UpdatedAt.AsTime(),
		Profile: dto.Profile{
			FirstName:   out.Profile.FirstName,
			LastName:    out.Profile.LastName,
			BirthDate:   out.Profile.BirthDate.String(),
			Citizenship: out.Profile.Citizenship,
			Phone:       out.Profile.Phone,
		},
	})
}

// GetCustomerStatus godoc
//
//	@Summary		Статус клиента
//	@Description	Возвращает текущий статус и момент его последней смены. Возможные значения: new, profile_filled, on_kyc, active, rejected, blocked.
//	@Tags			customer
//	@Produce		json
//	@Security		BearerAuth
//	@Param			customer_id	path		string	true	"Идентификатор клиента (UUID)"	format(uuid)
//	@Success		200			{object}	dto.CustomerStatus	"Текущий статус"
//	@Failure		400			{object}	dto.ErrorResponse	"Клиент не найден или ошибка customer-service"
//	@Failure		401			{object}	dto.ErrorResponse	"Отсутствует или недействителен access-токен"
//	@Router			/customer/{customer_id}/status [get]
func (h CustomerHandler) GetCustomerStatus(c *gin.Context) {
	id := c.Param("id")
	resp, err := h.service.Customer.GetCustomerStatus(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Message: "StatusBadRequest"})
		return
	}

	c.JSON(http.StatusOK, dto.CustomerStatus{
		Status:    resp.Status.String(),
		ChangedAt: resp.StatusChangedAt.AsTime(),
	})
}

// ListCustomers godoc
//
//	@Summary		Список клиентов
//	@Description	Постраничный список клиентов, отсортированный по дате создания по убыванию. Доступно ролям officer и admin.
//	@Tags			customer
//	@Produce		json
//	@Security		BearerAuth
//	@Param			limit	query		int					false	"Размер страницы, по умолчанию 20"	minimum(0)	maximum(100)
//	@Param			offset	query		int					false	"Сколько записей пропустить"			minimum(0)
//	@Success		200		{object}	dto.ListCustomers	"Страница списка"
//	@Failure		400		{object}	dto.ErrorResponse	"Некорректные параметры запроса"
//	@Failure		401		{object}	dto.ErrorResponse	"Отсутствует или недействителен access-токен"
//	@Failure		403		{object}	dto.ErrorResponse	"Роль не officer и не admin"
//	@Router			/customer [get]
func (h CustomerHandler) ListCustomers(c *gin.Context) {
	var query dto.ListCustomersQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Message: "StatusBadRequest"})
		return
	}

	resp, err := h.service.Customer.ListCustomers(c.Request.Context(), &customerv1.ListCustomersRequest{
		Limit:  query.Limit,
		Offset: query.Offset,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Message: "StatusBadRequest"})
		return
	}

	customers := make([]dto.Customer, 0, len(resp))
	for _, customer := range resp {
		customers = append(customers, dto.Customer{
			ID:        customer.Id,
			UserID:    customer.UserId,
			Status:    customer.Status.String(),
			CreatedAt: customer.CreatedAt.AsTime(),
			UpdatedAt: customer.UpdatedAt.AsTime(),
			Profile: dto.Profile{
				FirstName:   customer.Profile.FirstName,
				LastName:    customer.Profile.LastName,
				BirthDate:   customer.Profile.BirthDate.String(),
				Citizenship: customer.Profile.Citizenship,
				Phone:       customer.Profile.Phone,
			},
		})
	}

	c.JSON(http.StatusOK, dto.ListCustomers{
		Customers: customers,
	})
}

func parseBirthDate(s string) (*date.Date, error) {
	if s == "" {
		return nil, errors.New("birth date is empty")
	}

	t, err := time.Parse(time.DateOnly, s)
	if err != nil {
		return nil, domain.ErrInvalidBirthDate
	}

	return &date.Date{
		Year:  int32(t.Year()),
		Month: int32(t.Month()),
		Day:   int32(t.Day()),
	}, nil
}

//func parseCustomerStatus(s string) (customerv1.CustomerStatus, error) {
//	var customerStatuses = map[string]customerv1.CustomerStatus{
//		string(models.STATUS_NEW):            customerv1.CustomerStatus_NEW,
//		string(models.STATUS_PROFILE_FILLED): customerv1.CustomerStatus_PROFILE_FILLED,
//		string(models.STATUS_ON_KYC):         customerv1.CustomerStatus_ON_KYC,
//		string(models.STATUS_ACTIVE):         customerv1.CustomerStatus_ACTIVE,
//		string(models.STATUS_REJECTED):       customerv1.CustomerStatus_REJECTED,
//		string(models.STATUS_BLOCKED):        customerv1.CustomerStatus_BLOCKED,
//	}
//
//	status, ok := customerStatuses[strings.ToLower(strings.TrimSpace(s))]
//	if !ok {
//		return 0, ErrInvalidCustomerStatus
//	}
//
//	return status, nil
//}

//func statusFromProto(status customerv1.CustomerStatus) models.Status {
//	switch status {
//	case customerv1.CustomerStatus_NEW:
//		return models.STATUS_NEW
//	case customerv1.CustomerStatus_PROFILE_FILLED:
//		return models.STATUS_PROFILE_FILLED
//	case customerv1.CustomerStatus_ON_KYC:
//		return models.STATUS_ON_KYC
//	case customerv1.CustomerStatus_ACTIVE:
//		return models.STATUS_ACTIVE
//	case customerv1.CustomerStatus_REJECTED:
//		return models.STATUS_REJECTED
//	case customerv1.CustomerStatus_BLOCKED:
//		return models.STATUS_BLOCKED
//	default:
//		return models.STATUS_NEW
//	}
//}
