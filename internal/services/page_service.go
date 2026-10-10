package services

import (
	"errors"
	"fmt"

	"nexera_back/internal/logger"
	"nexera_back/internal/models"
	"nexera_back/internal/repository"
)

// ErrInvalidPage оборачивает ошибки валидации данных страницы (см. validatePage) —
// обработчик по нему отличает "клиент прислал некорректные данные" (400) от прочих ошибок.
var ErrInvalidPage = errors.New("invalid page")

// PageService представляет сервис для работы со страницами сайта.
type PageService struct {
	repo          repository.PageRepository
	bannerService *BannerService
	log           *logger.Logger
}

// NewPageService создает новый сервис для работы со страницами.
// bannerService переиспользуется для получения баннеров страницы (ListPublic) — так
// правило видимости баннера (активен и в окне показа) одно на оба публичных эндпоинта
// (/banners и /pages/{id}), а не продублировано на уровне репозитория страниц.
func NewPageService(repo repository.PageRepository, bannerService *BannerService, log *logger.Logger) *PageService {
	return &PageService{
		repo:          repo,
		bannerService: bannerService,
		log:           log,
	}
}

// ListPublic возвращает все страницы сайта (без вложенных баннеров).
func (svc *PageService) ListPublic() ([]*models.Page, error) {
	return svc.repo.List()
}

// GetPublicByIDWithBanners возвращает страницу по ID вместе с ее баннерами, видимыми
// прямо сейчас, отсортированными по позиции.
func (svc *PageService) GetPublicByIDWithBanners(id int) (*models.Page, error) {
	page, err := svc.repo.GetByID(id)
	if err != nil {
		return nil, err
	}

	banners, err := svc.bannerService.ListPublic(ListBannersFilter{
		PageID: &page.ID,
		Sort:   repository.BannerSortPosition,
		Order:  repository.SortAsc,
	})
	if err != nil {
		return nil, err
	}
	page.Banners = banners

	return page, nil
}

// Create создает новую страницу. Вызывается только из админ-панели.
func (svc *PageService) Create(req models.CreatePageRequest) (*models.Page, error) {
	page := &models.Page{
		Slug: req.Slug,
		Name: req.Name,
	}

	if err := validatePage(page); err != nil {
		return nil, err
	}

	if err := svc.repo.Create(page); err != nil {
		return nil, err
	}

	svc.log.WithField("page_id", page.ID).Info("created new page")

	return page, nil
}

// Update частично обновляет страницу: подгружает текущее состояние, накладывает на него
// только присланные в запросе поля (nil в запросе — поле остается прежним) и сохраняет
// итог целиком — тот же merge-then-validate-then-persist, что и BannerService.Update.
func (svc *PageService) Update(id int, req models.UpdatePageRequest) (*models.Page, error) {
	page, err := svc.repo.GetByID(id)
	if err != nil {
		return nil, err
	}

	if req.Slug != nil {
		page.Slug = *req.Slug
	}
	if req.Name != nil {
		page.Name = *req.Name
	}

	if err := validatePage(page); err != nil {
		return nil, err
	}

	if err := svc.repo.Update(page); err != nil {
		return nil, err
	}

	svc.log.WithField("page_id", page.ID).Info("updated page")

	return page, nil
}

// Delete удаляет страницу по ID. Вместе со страницей каскадно удаляются и все ее баннеры
// (banners.page_id -> pages(id) ON DELETE CASCADE) — это ограничение БД, не сервиса.
func (svc *PageService) Delete(id int) error {
	if err := svc.repo.Delete(id); err != nil {
		return err
	}

	svc.log.WithField("page_id", id).Info("deleted page")
	return nil
}

// validatePage проверяет бизнес-правила, обязательные для страницы независимо от того,
// получила ли она текущий вид целиком при Create или слиянием при Update.
func validatePage(p *models.Page) error {
	switch {
	case p.Slug == "":
		return fmt.Errorf("%w: slug is required", ErrInvalidPage)
	case p.Name == "":
		return fmt.Errorf("%w: name is required", ErrInvalidPage)
	default:
		return nil
	}
}
