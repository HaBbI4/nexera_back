package admin

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"nexera_back/internal/httpresponse"
	"nexera_back/internal/logger"
	"nexera_back/internal/media"
	"nexera_back/internal/models"
	"nexera_back/internal/redis"
	"nexera_back/internal/repository"
	"nexera_back/internal/services"
)

// maxBannerFormMemory ограничивает объем текстовых полей формы, буферизуемых в памяти
// ParseMultipartForm; сам файл изображения при превышении лимита спулится на диск во
// временный файл (поведение net/http, не наше).
const maxBannerFormMemory = 10 << 20 // 10MB

// bannerImageCategory — поддиректория/префикс в media.Storage для изображений баннеров
// (см. media.UploadImage).
const bannerImageCategory = "banners"

// allowedBannerSortFields — белый список значений query-параметра ?sort= для админского
// списка баннеров (тот же набор, что в публичном internal/handlers/public).
var allowedBannerSortFields = map[string]repository.BannerSortField{
	"position":   repository.BannerSortPosition,
	"created_at": repository.BannerSortCreatedAt,
}

// BannerHandler управляет баннерами из админ-панели: просмотр, создание, обновление,
// удаление. В отличие от публичного internal/handlers/public.BannerHandler, чтение здесь
// показывает все баннеры — и выключенные, и вне окна показа — и без кеша.
type BannerHandler struct {
	bannerService *services.BannerService
	redisClient   *redis.Client
	mediaStorage  media.Storage
	maxFileSizeMB int64
	log           *logger.Logger
}

// NewBannerHandler создает новый обработчик баннеров админ-панели
func NewBannerHandler(
	bannerService *services.BannerService,
	redisClient *redis.Client,
	mediaStorage media.Storage,
	maxFileSizeMB int64,
	log *logger.Logger,
) *BannerHandler {
	return &BannerHandler{
		bannerService: bannerService,
		redisClient:   redisClient,
		mediaStorage:  mediaStorage,
		maxFileSizeMB: maxFileSizeMB,
		log:           log,
	}
}

// GetBanners возвращает список всех баннеров для админ-панели с фильтрацией, сортировкой
// и пагинацией — вне зависимости от видимости на сайте.
//
// Как и в публичном GET /api/v1/banners: limit/offset при некорректном значении молча
// заменяются значением по умолчанию, а фильтры и сортировка — строже, некорректное значение — 400.
//
// @Summary      Список баннеров (админка)
// @Description  Возвращает все баннеры, включая выключенные и находящиеся вне окна показа. Требует авторизации.
// @Tags         admin-banners
// @Produce      json
// @Param        page_id    query     int     false  "ID страницы — вернуть баннеры только этой страницы"
// @Param        is_active  query     bool    false  "true — только включенные, false — только выключенные"
// @Param        sort       query     string  false  "Поле сортировки: position (по умолчанию) или created_at"
// @Param        order      query     string  false  "Направление сортировки: asc (по умолчанию) или desc"
// @Param        limit      query     int     false  "Количество записей (по умолчанию 50, максимум 100)"
// @Param        offset     query     int     false  "Смещение (по умолчанию 0)"
// @Success      200  {array}   models.Banner
// @Failure      400  {object}  httpresponse.ErrorResponse
// @Failure      401  {object}  httpresponse.ErrorResponse
// @Failure      500  {object}  httpresponse.ErrorResponse
// @Router       /api/v1/admin/banners [get]
func (h *BannerHandler) GetBanners(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpresponse.WriteError(w, http.StatusMethodNotAllowed, "Method Not Allowed")
		return
	}

	query := r.URL.Query()

	filter := services.ListAdminBannersFilter{
		Sort:  repository.BannerSortPosition,
		Order: repository.SortAsc,
	}

	if pageIDStr := query.Get("page_id"); pageIDStr != "" {
		pageID, err := strconv.Atoi(pageIDStr)
		if err != nil || pageID <= 0 {
			httpresponse.WriteError(w, http.StatusBadRequest, "invalid page_id")
			return
		}
		filter.PageID = &pageID
	}

	if isActiveStr := query.Get("is_active"); isActiveStr != "" {
		isActive, err := strconv.ParseBool(isActiveStr)
		if err != nil {
			httpresponse.WriteError(w, http.StatusBadRequest, "invalid is_active")
			return
		}
		filter.IsActive = &isActive
	}

	if sortStr := query.Get("sort"); sortStr != "" {
		sortField, ok := allowedBannerSortFields[sortStr]
		if !ok {
			httpresponse.WriteError(w, http.StatusBadRequest, "invalid sort, allowed: position, created_at")
			return
		}
		filter.Sort = sortField
	}

	if orderStr := query.Get("order"); orderStr != "" {
		switch repository.SortOrder(orderStr) {
		case repository.SortAsc, repository.SortDesc:
			filter.Order = repository.SortOrder(orderStr)
		default:
			httpresponse.WriteError(w, http.StatusBadRequest, "invalid order, allowed: asc, desc")
			return
		}
	}

	// Пагинация — по тому же правилу, что в остальных списочных эндпоинтах проекта:
	// некорректное значение не ошибка, а молчаливый возврат к дефолту.
	filter.Limit = 50 // По умолчанию
	if limitStr := query.Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			filter.Limit = l
		}
	}

	if offsetStr := query.Get("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			filter.Offset = o
		}
	}

	banners, err := h.bannerService.List(filter)
	if err != nil {
		h.log.WithError(err).Error("failed to get banners")
		httpresponse.WriteError(w, http.StatusInternalServerError, "failed to get banners")
		return
	}

	httpresponse.WriteJSON(w, http.StatusOK, banners)
}

// GetBanner возвращает баннер по ID для админ-панели — вне зависимости от видимости на сайте.
// @Summary      Получить баннер (админка)
// @Description  Возвращает баннер по ID, включая выключенный и находящийся вне окна показа. Требует авторизации.
// @Tags         admin-banners
// @Produce      json
// @Param        id   path      int  true  "ID баннера"
// @Success      200  {object}  models.Banner
// @Failure      400  {object}  httpresponse.ErrorResponse
// @Failure      401  {object}  httpresponse.ErrorResponse
// @Failure      404  {object}  httpresponse.ErrorResponse
// @Failure      500  {object}  httpresponse.ErrorResponse
// @Router       /api/v1/admin/banners/{id} [get]
func (h *BannerHandler) GetBanner(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpresponse.WriteError(w, http.StatusMethodNotAllowed, "Method Not Allowed")
		return
	}

	bannerID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || bannerID <= 0 {
		httpresponse.WriteError(w, http.StatusBadRequest, "invalid banner id")
		return
	}

	banner, err := h.bannerService.GetByID(bannerID)
	if err != nil {
		h.writeBannerError(w, err)
		return
	}

	httpresponse.WriteJSON(w, http.StatusOK, banner)
}

// CreateBanner создает баннер. Принимает multipart/form-data с текстовыми полями и картинкой
// в одном запросе: картинка — либо файл в поле "image" (загружается в media.Storage,
// image_url выставляется автоматически по результату загрузки), либо готовая ссылка в
// текстовом поле "image_url" — одно из двух обязательно, оба сразу — ошибка.
// @Summary      Создать баннер
// @Description  Создает новый баннер. Картинка — либо файл (поле image), либо готовый URL (поле image_url). Требует авторизации.
// @Tags         admin-banners
// @Accept       multipart/form-data
// @Produce      json
// @Param        page_id      formData  int     true   "ID страницы"
// @Param        title        formData  string  true   "Заголовок"
// @Param        subtitle     formData  string  false  "Подзаголовок"
// @Param        description  formData  string  false  "Описание"
// @Param        link_title   formData  string  false  "Текст ссылки"
// @Param        link_url     formData  string  false  "URL ссылки"
// @Param        image        formData  file    false  "Файл изображения (альтернатива image_url)"
// @Param        image_url    formData  string  false  "Готовый URL изображения (альтернатива файлу)"
// @Param        position     formData  int     false  "Порядок показа"
// @Param        starts_at    formData  string  false  "Начало показа, RFC3339"
// @Param        ends_at      formData  string  false  "Конец показа, RFC3339"
// @Param        is_active    formData  bool    false  "Активен"
// @Success      201  {object}  models.Banner
// @Failure      400  {object}  httpresponse.ErrorResponse
// @Failure      401  {object}  httpresponse.ErrorResponse
// @Failure      415  {object}  httpresponse.ErrorResponse
// @Failure      500  {object}  httpresponse.ErrorResponse
// @Router       /api/v1/admin/banners [post]
func (h *BannerHandler) CreateBanner(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpresponse.WriteError(w, http.StatusMethodNotAllowed, "Method Not Allowed")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, h.maxFileSizeMB*1024*1024+maxBannerFormMemory)
	if err := r.ParseMultipartForm(maxBannerFormMemory); err != nil {
		httpresponse.WriteError(w, http.StatusBadRequest, "invalid multipart form or request too large")
		return
	}
	defer r.MultipartForm.RemoveAll()

	form := r.PostForm
	var req models.CreateBannerRequest

	if pageIDStr := form.Get("page_id"); pageIDStr != "" {
		pageID, err := strconv.Atoi(pageIDStr)
		if err != nil {
			httpresponse.WriteError(w, http.StatusBadRequest, "invalid page_id")
			return
		}
		req.PageID = pageID
	}

	req.Title = form.Get("title")
	req.SubTitle = formStringPtr(form, "subtitle")
	req.Description = formStringPtr(form, "description")
	req.LinkTitle = formStringPtr(form, "link_title")
	req.LinkURL = formStringPtr(form, "link_url")

	if positionStr := form.Get("position"); positionStr != "" {
		position, err := strconv.Atoi(positionStr)
		if err != nil {
			httpresponse.WriteError(w, http.StatusBadRequest, "invalid position")
			return
		}
		req.Position = position
	}

	startsAt, err := formTimePtr(form, "starts_at")
	if err != nil {
		httpresponse.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	req.StartsAt = startsAt

	endsAt, err := formTimePtr(form, "ends_at")
	if err != nil {
		httpresponse.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	req.EndsAt = endsAt

	if isActiveStr := form.Get("is_active"); isActiveStr != "" {
		isActive, err := strconv.ParseBool(isActiveStr)
		if err != nil {
			httpresponse.WriteError(w, http.StatusBadRequest, "invalid is_active")
			return
		}
		req.IsActive = isActive
	}

	imageURL, err := h.resolveImageURL(r, form.Get("image_url"))
	if err != nil {
		h.writeBannerError(w, err)
		return
	}
	req.ImageURL = imageURL

	banner, err := h.bannerService.Create(req)
	if err != nil {
		h.writeBannerError(w, err)
		return
	}

	h.invalidatePageCache(r, banner.PageID)

	httpresponse.WriteJSON(w, http.StatusCreated, banner)
}

// UpdateBanner частично обновляет баннер — multipart/form-data, где отправляются только
// изменяемые поля (отсутствующее или пустое поле формы = не менять, см. models.UpdateBannerRequest).
// Картинка обновляется так же, как при создании: новый файл в поле "image" либо новый
// image_url текстом; если не прислано ни то, ни другое — картинка остается прежней.
// @Summary      Обновить баннер
// @Description  Частично обновляет баннер по ID. Требует авторизации.
// @Tags         admin-banners
// @Accept       multipart/form-data
// @Produce      json
// @Param        id           path      int     true   "ID баннера"
// @Param        page_id      formData  int     false  "ID страницы"
// @Param        title        formData  string  false  "Заголовок"
// @Param        subtitle     formData  string  false  "Подзаголовок"
// @Param        description  formData  string  false  "Описание"
// @Param        link_title   formData  string  false  "Текст ссылки"
// @Param        link_url     formData  string  false  "URL ссылки"
// @Param        image        formData  file    false  "Новый файл изображения (альтернатива image_url)"
// @Param        image_url    formData  string  false  "Новый URL изображения (альтернатива файлу)"
// @Param        position     formData  int     false  "Порядок показа"
// @Param        starts_at    formData  string  false  "Начало показа, RFC3339"
// @Param        ends_at      formData  string  false  "Конец показа, RFC3339"
// @Param        is_active    formData  bool    false  "Активен"
// @Success      200  {object}  models.Banner
// @Failure      400  {object}  httpresponse.ErrorResponse
// @Failure      401  {object}  httpresponse.ErrorResponse
// @Failure      404  {object}  httpresponse.ErrorResponse
// @Failure      415  {object}  httpresponse.ErrorResponse
// @Failure      500  {object}  httpresponse.ErrorResponse
// @Router       /api/v1/admin/banners/{id} [patch]
func (h *BannerHandler) UpdateBanner(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		httpresponse.WriteError(w, http.StatusMethodNotAllowed, "Method Not Allowed")
		return
	}

	bannerID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || bannerID <= 0 {
		httpresponse.WriteError(w, http.StatusBadRequest, "invalid banner id")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, h.maxFileSizeMB*1024*1024+maxBannerFormMemory)
	if err := r.ParseMultipartForm(maxBannerFormMemory); err != nil {
		httpresponse.WriteError(w, http.StatusBadRequest, "invalid multipart form or request too large")
		return
	}
	defer r.MultipartForm.RemoveAll()

	form := r.PostForm
	var req models.UpdateBannerRequest

	pageID, err := formIntPtr(form, "page_id")
	if err != nil {
		httpresponse.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	req.PageID = pageID

	req.Title = formStringPtr(form, "title")
	req.SubTitle = formStringPtr(form, "subtitle")
	req.Description = formStringPtr(form, "description")
	req.LinkTitle = formStringPtr(form, "link_title")
	req.LinkURL = formStringPtr(form, "link_url")

	position, err := formIntPtr(form, "position")
	if err != nil {
		httpresponse.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	req.Position = position

	startsAt, err := formTimePtr(form, "starts_at")
	if err != nil {
		httpresponse.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	req.StartsAt = startsAt

	endsAt, err := formTimePtr(form, "ends_at")
	if err != nil {
		httpresponse.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	req.EndsAt = endsAt

	isActive, err := formBoolPtr(form, "is_active")
	if err != nil {
		httpresponse.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	req.IsActive = isActive

	imageURL, err := h.resolveOptionalImageURL(r, form)
	if err != nil {
		h.writeBannerError(w, err)
		return
	}
	req.ImageURL = imageURL

	if req == (models.UpdateBannerRequest{}) {
		httpresponse.WriteError(w, http.StatusBadRequest, "at least one field is required")
		return
	}

	banner, err := h.bannerService.Update(bannerID, req)
	if err != nil {
		h.writeBannerError(w, err)
		return
	}

	// Инвалидация кеша: следующий публичный GET /banners/{id} перечитает из БД.
	cacheKey := redis.GenerateKey(redis.KeyPrefixBanner, strconv.Itoa(bannerID))
	if err := h.redisClient.Delete(r.Context(), cacheKey); err != nil {
		h.log.WithError(err).Error("failed to invalidate banner cache")
	}
	// Баннер встроен в кеш своей страницы (GET /pages/{id}) — его тоже нужно сбросить,
	// иначе правка баннера не будет видна внутри страницы до истечения TTL. Используем
	// итоговый (после Update) banner.PageID: если в этом же запросе banner "переехал" на
	// другую страницу, кеш старой страницы останется устаревшим до TTL — редкий случай,
	// отдельно не обрабатываем.
	h.invalidatePageCache(r, banner.PageID)

	httpresponse.WriteJSON(w, http.StatusOK, banner)
}

// DeleteBanner удаляет баннер по ID.
// @Summary      Удалить баннер
// @Description  Удаляет баннер по ID. Требует авторизации.
// @Tags         admin-banners
// @Param        id  path  int  true  "ID баннера"
// @Success      204  "No Content"
// @Failure      400  {object}  httpresponse.ErrorResponse
// @Failure      401  {object}  httpresponse.ErrorResponse
// @Failure      404  {object}  httpresponse.ErrorResponse
// @Failure      500  {object}  httpresponse.ErrorResponse
// @Router       /api/v1/admin/banners/{id} [delete]
func (h *BannerHandler) DeleteBanner(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		httpresponse.WriteError(w, http.StatusMethodNotAllowed, "Method Not Allowed")
		return
	}

	bannerID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || bannerID <= 0 {
		httpresponse.WriteError(w, http.StatusBadRequest, "invalid banner id")
		return
	}

	// Баннер нужно прочитать до удаления — иначе после Delete неоткуда взять его PageID,
	// чтобы сбросить кеш страницы, в которую он был встроен (см. invalidatePageCache).
	banner, err := h.bannerService.GetByID(bannerID)
	if err != nil {
		h.writeBannerError(w, err)
		return
	}

	if err := h.bannerService.Delete(bannerID); err != nil {
		h.writeBannerError(w, err)
		return
	}

	cacheKey := redis.GenerateKey(redis.KeyPrefixBanner, strconv.Itoa(bannerID))
	if err := h.redisClient.Delete(r.Context(), cacheKey); err != nil {
		h.log.WithError(err).Error("failed to invalidate banner cache")
	}
	h.invalidatePageCache(r, banner.PageID)

	w.WriteHeader(http.StatusNoContent)
}

// invalidatePageCache сбрасывает кеш страницы pageID (GET /api/v1/pages/{id}), в которую
// встроен список баннеров — любая запись в banners делает этот кеш потенциально устаревшим.
func (h *BannerHandler) invalidatePageCache(r *http.Request, pageID int) {
	cacheKey := redis.GenerateKey(redis.KeyPrefixPage, strconv.Itoa(pageID))
	if err := h.redisClient.Delete(r.Context(), cacheKey); err != nil {
		h.log.WithError(err).Error("failed to invalidate page cache")
	}
}

// resolveImageURL решает, откуда взять image_url для Create: если в запросе есть файл в поле
// "image" — загружает его через media.UploadImage (универсальная точка входа для загрузки
// картинок, см. internal/media/image.go) и возвращает URL загруженного файла; если файла нет —
// использует urlFromForm как есть (баннер ссылается на уже где-то размещенную картинку).
// Оба способа сразу — ошибка: непонятно, какой из них в приоритете.
func (h *BannerHandler) resolveImageURL(r *http.Request, urlFromForm string) (string, error) {
	files := r.MultipartForm.File["image"]
	if len(files) > 0 && urlFromForm != "" {
		return "", fmt.Errorf("%w: specify either image file or image_url, not both", services.ErrInvalidBanner)
	}
	if len(files) == 0 {
		return urlFromForm, nil
	}

	f, err := files[0].Open()
	if err != nil {
		return "", fmt.Errorf("failed to open uploaded image: %w", err)
	}
	defer f.Close()

	return media.UploadImage(r.Context(), h.mediaStorage, bannerImageCategory, files[0].Size, f)
}

// resolveOptionalImageURL — то же самое, что resolveImageURL, но для PATCH: если не прислано
// ни файла, ни image_url, возвращает nil — существующий image_url баннера остается прежним.
func (h *BannerHandler) resolveOptionalImageURL(r *http.Request, form url.Values) (*string, error) {
	files := r.MultipartForm.File["image"]
	urlFromForm := form.Get("image_url")

	if len(files) > 0 && urlFromForm != "" {
		return nil, fmt.Errorf("%w: specify either image file or image_url, not both", services.ErrInvalidBanner)
	}
	if len(files) == 0 {
		if urlFromForm == "" {
			return nil, nil
		}
		return &urlFromForm, nil
	}

	f, err := files[0].Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open uploaded image: %w", err)
	}
	defer f.Close()

	uploadedURL, err := media.UploadImage(r.Context(), h.mediaStorage, bannerImageCategory, files[0].Size, f)
	if err != nil {
		return nil, err
	}
	return &uploadedURL, nil
}

// formStringPtr читает текстовое поле формы: отсутствующее или пустое значение — nil
// ("не менять" для PATCH; для Create используется напрямую, required-проверка — в validateBanner).
func formStringPtr(form url.Values, key string) *string {
	v := form.Get(key)
	if v == "" {
		return nil
	}
	return &v
}

// formIntPtr читает числовое поле формы по тем же правилам, что formStringPtr.
func formIntPtr(form url.Values, key string) (*int, error) {
	raw := form.Get(key)
	if raw == "" {
		return nil, nil
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return nil, fmt.Errorf("invalid %s", key)
	}
	return &v, nil
}

// formBoolPtr читает булево поле формы по тем же правилам, что formStringPtr.
func formBoolPtr(form url.Values, key string) (*bool, error) {
	raw := form.Get(key)
	if raw == "" {
		return nil, nil
	}
	v, err := strconv.ParseBool(raw)
	if err != nil {
		return nil, fmt.Errorf("invalid %s", key)
	}
	return &v, nil
}

// formTimePtr читает поле формы с датой/временем (RFC3339) по тем же правилам, что formStringPtr.
func formTimePtr(form url.Values, key string) (*time.Time, error) {
	raw := form.Get(key)
	if raw == "" {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return nil, fmt.Errorf("invalid %s, expected RFC3339", key)
	}
	return &t, nil
}

// writeBannerError сопоставляет ошибки сервисного/медиа-слоя с HTTP статус-кодами через
// errors.Is/errors.As по сигнальным ошибкам, а не разбором текста сообщения — это надежнее
// и не ломается при изменении формулировок.
func (h *BannerHandler) writeBannerError(w http.ResponseWriter, err error) {
	var unsupportedFormat *media.ErrUnsupportedFormat

	switch {
	case errors.Is(err, services.ErrInvalidBanner):
		httpresponse.WriteError(w, http.StatusBadRequest, err.Error())
	case errors.As(err, &unsupportedFormat):
		httpresponse.WriteError(w, http.StatusUnsupportedMediaType, err.Error())
	case errors.Is(err, repository.ErrBannerNotFound):
		httpresponse.WriteError(w, http.StatusNotFound, "banner not found")
	case errors.Is(err, repository.ErrPageNotFound):
		httpresponse.WriteError(w, http.StatusBadRequest, "page not found")
	default:
		h.log.WithError(err).Error("banner operation failed")
		httpresponse.WriteError(w, http.StatusInternalServerError, "internal server error")
	}
}
