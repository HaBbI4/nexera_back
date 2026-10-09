-- Услуги компании, отображаемые на сайте (с ценой, описанием и порядком показа).
CREATE TABLE services (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    description VARCHAR(5000),
    price INTEGER CHECK (price >= 0),
    position INTEGER NOT NULL DEFAULT 0 CHECK (position >= 0),
    is_published BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX services_published_position_idx
    ON services (position, id) WHERE is_published;