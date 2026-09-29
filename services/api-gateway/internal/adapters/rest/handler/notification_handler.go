package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/danila-kuryakin/banking_mini_cores/services/api-gateway/internal/adapters/rest/handler/dto"
	"github.com/danila-kuryakin/banking_mini_cores/services/api-gateway/internal/app/service"
	notificationv1 "github.com/danila-kuryakin/banking_mini_cores/services/api-gateway/internal/pb/gen/notification/v1"
)

type NotificationHandler struct {
	service *service.Service
}

func NewNotificationHandler(service *service.Service) *NotificationHandler {
	return &NotificationHandler{service: service}
}

// ListNotifications godoc
//
//	@Summary		Уведомления клиента
//	@Description	Уведомления, «отправленные» клиенту по событиям переводов и KYC, от новых к старым. Отправка - мок: уведомление только записывается.
//	@Tags			notification
//	@Produce		json
//	@Security		BearerAuth
//	@Param			user_id	path		string					true	"UUID пользователя или me - текущий пользователь"
//	@Param			limit	query		int						false	"Размер страницы"
//	@Param			offset	query		int						false	"Смещение"
//	@Success		200		{object}	dto.ListNotifications	"Уведомления"
//	@Failure		400		{object}	dto.ErrorResponse		"Некорректная пагинация"
//	@Failure		401		{object}	dto.ErrorResponse		"Отсутствует или недействителен access-токен"
//	@Failure		403		{object}	dto.ErrorResponse		"Чужой клиент"
//	@Router			/customer/{user_id}/notifications [get]
func (h NotificationHandler) ListNotifications(c *gin.Context) {
	var query dto.ListNotificationsQuery

	if err := c.ShouldBindQuery(&query); err != nil {
		writeBindError(c, err)

		return
	}

	notifications, err := h.service.Notification.ListNotifications(c.Request.Context(), &notificationv1.ListNotificationsRequest{
		UserId: targetUserID(c),
		Limit:  query.Limit,
		Offset: query.Offset,
	})
	if err != nil {
		writeError(c, err)

		return
	}

	items := make([]dto.Notification, 0, len(notifications))
	for _, n := range notifications {
		items = append(items, dto.Notification{
			ID:        n.Id,
			EventID:   n.EventId,
			EventType: n.EventType,
			Text:      n.Text,
			CreatedAt: n.CreatedAt.AsTime(),
		})
	}

	c.JSON(http.StatusOK, dto.ListNotifications{Notifications: items})
}
