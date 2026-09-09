package handler

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/danila-kuryakin/banking_mini_cores/services/api-gateway/internal/adapters/rest/handler/dto"
	"github.com/danila-kuryakin/banking_mini_cores/services/api-gateway/internal/app/service"
	"github.com/danila-kuryakin/banking_mini_cores/services/api-gateway/internal/domain"
	filestorev1 "github.com/danila-kuryakin/banking_mini_cores/services/api-gateway/internal/pb/gen/filestore/v1"
)

type FilestoreHandler struct {
	service *service.Service
}

func NewFilestoreHandler(service *service.Service) *FilestoreHandler {
	return &FilestoreHandler{
		service: service,
	}
}

// InitUpload godoc
//
//	@Summary		Заявка на загрузку файла
//	@Description	Заводит запись о файле и возвращает presigned-ссылку. Файл клиент кладёт сам: PUT по upload_url с телом файла, gateway байты не принимает. После заливки нужно вызвать confirm. Типы: passport, selfie, proof_of_address.
//	@Tags			files
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			user_id	path		string				true	"UUID пользователя или me - текущий пользователь"
//	@Param			request	body		dto.InitUpload		true	"Тип файла и исходное имя"
//	@Success		200		{object}	dto.UploadTicket	"Ссылка на загрузку"
//	@Failure		400		{object}	dto.ErrorResponse	"Некорректное тело запроса или неизвестный тип файла"
//	@Failure		401		{object}	dto.ErrorResponse	"Отсутствует или недействителен access-токен"
//	@Failure		403		{object}	dto.ErrorResponse	"Файлы принадлежат другому пользователю"
//	@Router			/customer/{user_id}/files [post]
func (h FilestoreHandler) InitUpload(c *gin.Context) {
	var req dto.InitUpload
	if err := c.ShouldBindJSON(&req); err != nil {
		writeBindError(c, err)

		return
	}

	fileType, err := fileTypeFromName(req.Type)
	if err != nil {
		writeError(c, err)

		return
	}

	resp, err := h.service.Filestore.InitUpload(c.Request.Context(), targetUserID(c), fileType, req.Filename)
	if err != nil {
		writeError(c, err)

		return
	}

	c.JSON(http.StatusOK, dto.UploadTicket{
		FileID:    resp.FileId,
		UploadURL: resp.UploadUrl,
		ExpiresAt: resp.ExpiresAt.AsTime(),
	})
}

// ConfirmUpload godoc
//
//	@Summary		Подтверждение загрузки
//	@Description	Проверяет залитый файл: размер не больше 10 МБ, тип определяется по содержимому (jpeg, png, pdf), считается sha256. Только после этого файл становится доступен для скачивания. Повторный вызов по подтверждённому файлу ничего не меняет.
//	@Tags			files
//	@Produce		json
//	@Security		BearerAuth
//	@Param			user_id	path		string				true	"UUID пользователя или me - текущий пользователь"
//	@Param			file_id	path		string				true	"UUID файла из ответа на заявку"
//	@Success		200		{object}	dto.File			"Файл подтверждён"
//	@Failure		400		{object}	dto.ErrorResponse	"Файл пустой, больше 10 МБ или неподдерживаемого типа"
//	@Failure		401		{object}	dto.ErrorResponse	"Отсутствует или недействителен access-токен"
//	@Failure		403		{object}	dto.ErrorResponse	"Файл принадлежит другому пользователю"
//	@Failure		404		{object}	dto.ErrorResponse	"Файл не найден"
//	@Failure		409		{object}	dto.ErrorResponse	"Файл этого типа уже подтверждён"
//	@Router			/customer/{user_id}/files/{file_id}/confirm [post]
func (h FilestoreHandler) ConfirmUpload(c *gin.Context) {
	resp, err := h.service.Filestore.ConfirmUpload(c.Request.Context(), targetUserID(c), c.Param(domain.FILE_ID_PARAM))
	if err != nil {
		writeError(c, err)

		return
	}

	c.JSON(http.StatusOK, fileResponse(resp))
}

// GetDownloadURL godoc
//
//	@Summary		Ссылка на скачивание
//	@Description	Возвращает presigned-ссылку со сроком жизни 5 минут. Скачивает по ней клиент сам, gateway файл не отдаёт. Работает только для подтверждённых файлов.
//	@Tags			files
//	@Produce		json
//	@Security		BearerAuth
//	@Param			user_id	path		string				true	"UUID пользователя или me - текущий пользователь"
//	@Param			file_id	path		string				true	"UUID файла"
//	@Success		200		{object}	dto.DownloadTicket	"Ссылка на скачивание"
//	@Failure		400		{object}	dto.ErrorResponse	"Файл ещё не подтверждён"
//	@Failure		401		{object}	dto.ErrorResponse	"Отсутствует или недействителен access-токен"
//	@Failure		403		{object}	dto.ErrorResponse	"Файл принадлежит другому пользователю"
//	@Failure		404		{object}	dto.ErrorResponse	"Файл не найден"
//	@Router			/customer/{user_id}/files/{file_id}/url [get]
func (h FilestoreHandler) GetDownloadURL(c *gin.Context) {
	resp, err := h.service.Filestore.GetDownloadURL(c.Request.Context(), targetUserID(c), c.Param(domain.FILE_ID_PARAM))
	if err != nil {
		writeError(c, err)

		return
	}

	c.JSON(http.StatusOK, dto.DownloadTicket{
		URL:       resp.Url,
		ExpiresAt: resp.ExpiresAt.AsTime(),
	})
}

// ListFiles godoc
//
//	@Summary		Список файлов клиента
//	@Description	Постраничный список файлов, отсортированный по дате создания по убыванию. Незавершённые загрузки тоже видны - у них status = uploaded.
//	@Tags			files
//	@Produce		json
//	@Security		BearerAuth
//	@Param			user_id	path		string				true	"UUID пользователя или me - текущий пользователь"
//	@Param			limit	query		int					false	"Размер страницы, по умолчанию 20"	minimum(0)	maximum(100)
//	@Param			offset	query		int					false	"Сколько записей пропустить"		minimum(0)
//	@Success		200		{object}	dto.ListFiles		"Страница списка"
//	@Failure		400		{object}	dto.ErrorResponse	"Некорректные параметры запроса"
//	@Failure		401		{object}	dto.ErrorResponse	"Отсутствует или недействителен access-токен"
//	@Failure		403		{object}	dto.ErrorResponse	"Файлы принадлежат другому пользователю"
//	@Router			/customer/{user_id}/files [get]
func (h FilestoreHandler) ListFiles(c *gin.Context) {
	var query dto.ListFilesQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		writeBindError(c, err)

		return
	}

	resp, err := h.service.Filestore.ListFiles(c.Request.Context(), &filestorev1.ListFilesRequest{
		UserId: targetUserID(c),
		Limit:  query.Limit,
		Offset: query.Offset,
	})
	if err != nil {
		writeError(c, err)

		return
	}

	files := make([]dto.File, 0, len(resp))
	for _, file := range resp {
		files = append(files, fileResponse(file))
	}

	c.JSON(http.StatusOK, dto.ListFiles{
		Files: files,
	})
}

// HasRequiredFiles godoc
//
//	@Summary		Комплектность файлов
//	@Description	Загружены ли подтверждённые файлы обязательных типов (паспорт и селфи). В missing перечислено недостающее.
//	@Tags			files
//	@Produce		json
//	@Security		BearerAuth
//	@Param			user_id	path		string				true	"UUID пользователя или me - текущий пользователь"
//	@Success		200		{object}	dto.RequiredFiles	"Состояние комплекта"
//	@Failure		401		{object}	dto.ErrorResponse	"Отсутствует или недействителен access-токен"
//	@Failure		403		{object}	dto.ErrorResponse	"Файлы принадлежат другому пользователю"
//	@Router			/customer/{user_id}/files/required [get]
func (h FilestoreHandler) HasRequiredFiles(c *gin.Context) {
	resp, err := h.service.Filestore.HasRequiredFiles(c.Request.Context(), targetUserID(c))
	if err != nil {
		writeError(c, err)

		return
	}

	missing := make([]string, 0, len(resp.Missing))
	for _, fileType := range resp.Missing {
		missing = append(missing, fileTypeName(fileType))
	}

	c.JSON(http.StatusOK, dto.RequiredFiles{
		OK:      resp.Ok,
		Missing: missing,
	})
}

func fileResponse(file *filestorev1.FileMetadata) dto.File {
	return dto.File{
		ID:          file.Id,
		UserID:      file.UserId,
		Type:        fileTypeName(file.Type),
		Status:      fileStatusName(file.Status),
		Filename:    file.Filename,
		ContentType: file.ContentType,
		SizeBytes:   file.SizeBytes,
		SHA256:      file.Sha256,
		CreatedAt:   file.CreatedAt.AsTime(),
		ConfirmedAt: confirmedAt(file),
	}
}

// confirmedAt отсутствует, пока файл не подтверждён. Отдавать вместо этого
// нулевую дату нельзя: клиент прочитает её как 1 января первого года.
func confirmedAt(file *filestorev1.FileMetadata) *time.Time {
	if file.ConfirmedAt == nil {
		return nil
	}

	at := file.ConfirmedAt.AsTime()

	return &at
}

// fileTypeName и fileStatusName отдают доменное имя без префикса enum:
// "passport" вместо "FILE_TYPE_PASSPORT".
func fileTypeName(fileType filestorev1.FileType) string {
	return strings.ToLower(strings.TrimPrefix(fileType.String(), domain.FILE_TYPE_ENUM_PREFIX))
}

func fileStatusName(status filestorev1.FileStatus) string {
	return strings.ToLower(strings.TrimPrefix(status.String(), domain.FILE_STATUS_ENUM_PREFIX))
}

// fileTypeFromName принимает доменное имя ("passport"), а не имя значения enum.
// Неизвестное значение отсекается здесь, а не в filestore-service: тот на
// UNSPECIFIED ответит тем же 400, но лишним сетевым вызовом.
func fileTypeFromName(raw string) (filestorev1.FileType, error) {
	name := domain.FILE_TYPE_ENUM_PREFIX + strings.ToUpper(strings.TrimSpace(raw))

	value, ok := filestorev1.FileType_value[name]
	if !ok || filestorev1.FileType(value) == filestorev1.FileType_FILE_TYPE_UNSPECIFIED {
		return filestorev1.FileType_FILE_TYPE_UNSPECIFIED, domain.ErrFileTypeUnknown
	}

	return filestorev1.FileType(value), nil
}
