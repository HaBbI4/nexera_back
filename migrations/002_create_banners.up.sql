-- Баннеры сайта-визитки. Один баннер может показываться на нескольких страницах
-- и в нескольких позициях одновременно (см. banner_pages / banner_positions).
CREATE TABLE banners (
    id BIGSERIAL PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    image_url TEXT NOT NULL,
    link_url TEXT,
    priority INTEGER NOT NULL DEFAULT 0,
    starts_at TIMESTAMPTZ,
    ends_at TIMESTAMPTZ,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Страницы сайта, на которых показывается баннер (например "home", "pricing", "about")
CREATE TABLE banner_pages (
    id BIGSERIAL PRIMARY KEY,
    banner_id BIGINT NOT NULL REFERENCES banners(id) ON DELETE CASCADE,
    page VARCHAR(100) NOT NULL
);

CREATE INDEX idx_banner_pages_page ON banner_pages(page);
CREATE UNIQUE INDEX idx_banner_pages_banner_page ON banner_pages(banner_id, page);

-- Позиции на странице (например "header", "sidebar", "footer")
CREATE TABLE banner_positions (
    id BIGSERIAL PRIMARY KEY,
    banner_id BIGINT NOT NULL REFERENCES banners(id) ON DELETE CASCADE,
    position VARCHAR(100) NOT NULL
);

CREATE INDEX idx_banner_positions_position ON banner_positions(position);
CREATE UNIQUE INDEX idx_banner_positions_banner_position ON banner_positions(banner_id, position);

CREATE INDEX idx_banners_active_priority ON banners(is_active, priority);
