-- Страницы сайта, на которых можно показывать баннеры (например "home", "pricing", "about").
-- Отдельная таблица вместо enum — чтобы новую страницу можно было добавить через админку,
-- без миграции и деплоя.
CREATE TABLE pages (
    id BIGSERIAL PRIMARY KEY,
    slug VARCHAR(100) NOT NULL,
    name VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX pages_slug_uq ON pages (slug);

-- Баннеры сайта-визитки. Один баннер показывается на одной странице (banners.page_id),
-- на одной странице может быть несколько баннеров — порядок задается полем position.
CREATE TABLE banners (
    id BIGSERIAL PRIMARY KEY,
    page_id BIGINT NOT NULL REFERENCES pages(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    subtitle VARCHAR(255),
    description VARCHAR(255),
    link_title VARCHAR(255),
    link_url TEXT,
    image_url TEXT NOT NULL,
    position INTEGER NOT NULL DEFAULT 0 CHECK (position >= 0),
    starts_at TIMESTAMPTZ,
    ends_at TIMESTAMPTZ,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX banners_page_position_idx ON banners (page_id, position);
CREATE INDEX banners_active_idx ON banners (is_active);
