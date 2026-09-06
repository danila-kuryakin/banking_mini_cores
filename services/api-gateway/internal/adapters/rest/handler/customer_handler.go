package handler

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"google.golang.org/genproto/googleapis/type/date"

	"github.com/danila-kuryakin/banking_mini_cores/services/api-gateway/internal/adapters/rest/handler/dto"
	"github.com/danila-kuryakin/banking_mini_cores/services/api-gateway/internal/app/service"
	"github.com/danila-kuryakin/banking_mini_cores/services/api-gateway/internal/domain"
	customerv1 "github.com/danila-kuryakin/banking_mini_cores/services/api-gateway/internal/pb/gen/customer/v1"
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
//	@Description	Возвращает клиента вместе с профилем и текущим статусом. Клиент видит только себя, officer и admin - любого.
//	@Tags			customer
//	@Produce		json
//	@Security		BearerAuth
//	@Param			user_id	path		string	true	"UUID пользователя или me - текущий пользователь"
//	@Success		200		{object}	dto.Customer		"Клиент найден"
//	@Failure		401		{object}	dto.ErrorResponse	"Отсутствует или недействителен access-токен"
//	@Failure		403		{object}	dto.ErrorResponse	"Профиль принадлежит другому пользователю"
//	@Failure		404		{object}	dto.ErrorResponse	"Клиент не найден"
//	@Router			/customer/{user_id} [get]
func (h CustomerHandler) GetCustomer(c *gin.Context) {
	resp, err := h.service.Customer.GetCustomer(c.Request.Context(), targetUserID(c))
	if err != nil {
		writeError(c, err)

		return
	}

	c.JSON(http.StatusOK, customerResponse(resp))
}

// UpdateProfile godoc
//
//	@Summary		Обновление профиля клиента
//	@Description	Записывает анкетные данные. Поле birth_date передаётся строкой в формате YYYY-MM-DD и может быть пустым. Когда профиль заполнен целиком, клиент переходит в статус profile_filled и дальнейшее редактирование запрещается.
//	@Tags			customer
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			user_id	path		string				true	"UUID пользователя или me - текущий пользователь"
//	@Param			request	body		dto.UpdateProfile	true	"Поля профиля"
//	@Success		200		{object}	dto.Customer		"Профиль обновлён"
//	@Failure		400		{object}	dto.ErrorResponse	"Некорректное тело запроса или профиль больше не редактируется"
//	@Failure		401		{object}	dto.ErrorResponse	"Отсутствует или недействителен access-токен"
//	@Failure		403		{object}	dto.ErrorResponse	"Профиль принадлежит другому пользователю"
//	@Failure		404		{object}	dto.ErrorResponse	"Клиент не найден"
//	@Router			/customer/{user_id} [post]
func (h CustomerHandler) UpdateProfile(c *gin.Context) {
	var req dto.UpdateProfile
	if err := c.ShouldBindJSON(&req); err != nil {
		writeBindError(c, err)

		return
	}

	birthDate, err := parseBirthDate(req.BirthDate)
	if err != nil {
		writeError(c, err)

		return
	}

	out, err := h.service.Customer.UpdateProfile(c.Request.Context(), targetUserID(c), &customerv1.Profile{
		FirstName:   req.FirstName,
		LastName:    req.LastName,
		BirthDate:   birthDate,
		Citizenship: req.Citizenship,
		Phone:       req.Phone,
	})
	if err != nil {
		writeError(c, err)

		return
	}

	c.JSON(http.StatusOK, customerResponse(out))
}

// GetCustomerStatus godoc
//
//	@Summary		Статус клиента
//	@Description	Возвращает текущий статус и момент его последней смены. Возможные значения: new, profile_filled, on_kyc, active, rejected, blocked.
//	@Tags			customer
//	@Produce		json
//	@Security		BearerAuth
//	@Param			user_id	path		string	true	"UUID пользователя или me - текущий пользователь"
//	@Success		200		{object}	dto.CustomerStatus	"Текущий статус"
//	@Failure		401		{object}	dto.ErrorResponse	"Отсутствует или недействителен access-токен"
//	@Failure		403		{object}	dto.ErrorResponse	"Профиль принадлежит другому пользователю"
//	@Failure		404		{object}	dto.ErrorResponse	"Клиент не найден"
//	@Router			/customer/{user_id}/status [get]
func (h CustomerHandler) GetCustomerStatus(c *gin.Context) {
	resp, err := h.service.Customer.GetCustomerStatus(c.Request.Context(), targetUserID(c))
	if err != nil {
		writeError(c, err)

		return
	}

	c.JSON(http.StatusOK, dto.CustomerStatus{
		Status:    statusName(resp.Status),
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
//	@Param			offset	query		int					false	"Сколько записей пропустить"		minimum(0)
//	@Success		200		{object}	dto.ListCustomers	"Страница списка"
//	@Failure		400		{object}	dto.ErrorResponse	"Некорректные параметры запроса"
//	@Failure		401		{object}	dto.ErrorResponse	"Отсутствует или недействителен access-токен"
//	@Failure		403		{object}	dto.ErrorResponse	"Роль не officer и не admin"
//	@Router			/customer [get]
func (h CustomerHandler) ListCustomers(c *gin.Context) {
	var query dto.ListCustomersQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		writeBindError(c, err)

		return
	}

	resp, err := h.service.Customer.ListCustomers(c.Request.Context(), &customerv1.ListCustomersRequest{
		Limit:  query.Limit,
		Offset: query.Offset,
	})
	if err != nil {
		writeError(c, err)

		return
	}

	customers := make([]dto.Customer, 0, len(resp))
	for _, customer := range resp {
		customers = append(customers, customerResponse(customer))
	}

	c.JSON(http.StatusOK, dto.ListCustomers{
		Customers: customers,
	})
}

// targetUserID - чей профиль запрошен. Литерал "me" в пути подменяется
// идентификатором из токена: так клиенту не нужно знать свой UUID, а вложенные
// пути вроде /customer/me/status работают без отдельных маршрутов.
func targetUserID(c *gin.Context) string {
	raw := c.Param(domain.USER_ID_PARAM)
	if raw != domain.SELF_ALIAS {
		return raw
	}

	actor, ok := ActorFrom(c)
	if !ok {
		return raw
	}

	return actor.ID
}

func customerResponse(customer *customerv1.Customer) dto.Customer {
	return dto.Customer{
		ID:        customer.Id,
		UserID:    customer.UserId,
		Status:    statusName(customer.Status),
		Profile:   profileResponse(customer.Profile),
		CreatedAt: customer.CreatedAt.AsTime(),
		UpdatedAt: customer.UpdatedAt.AsTime(),
	}
}

func profileResponse(profile *customerv1.Profile) dto.Profile {
	if profile == nil {
		return dto.Profile{}
	}

	return dto.Profile{
		FirstName:   profile.FirstName,
		LastName:    profile.LastName,
		BirthDate:   formatBirthDate(profile.BirthDate),
		Citizenship: profile.Citizenship,
		Phone:       profile.Phone,
	}
}

// statusName отдаёт статус в том же виде, в котором он живёт в БД клиента
// ("profile_filled"), а не именем значения enum ("PROFILE_FILLED").
func statusName(status customerv1.CustomerStatus) string {
	return strings.ToLower(status.String())
}

// parseBirthDate принимает дату в YYYY-MM-DD. Пустая строка - это "поле не
// заполнено", законное состояние незавершённой анкеты, а не ошибка.
func parseBirthDate(raw string) (*date.Date, error) {
	if raw == "" {
		return nil, nil
	}

	parsed, err := time.Parse(time.DateOnly, raw)
	if err != nil {
		return nil, domain.ErrInvalidBirthDate
	}

	return &date.Date{
		Year:  int32(parsed.Year()),
		Month: int32(parsed.Month()),
		Day:   int32(parsed.Day()),
	}, nil
}

// formatBirthDate - обратная операция к parseBirthDate. Без неё наружу уезжал
// бы protobuf-текст вида "year:1990 month:5 day:17": у date.Date нет своего
// String() в формате даты.
func formatBirthDate(in *date.Date) string {
	if in == nil {
		return ""
	}

	return time.Date(int(in.Year), time.Month(in.Month), int(in.Day), 0, 0, 0, 0, time.UTC).
		Format(time.DateOnly)
}
