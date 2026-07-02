CREATE TABLE IF NOT EXISTS banners (
    id          BIGSERIAL PRIMARY KEY,
    description TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS slots (
    id          BIGSERIAL PRIMARY KEY,
    description TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS social_groups (
    id          BIGSERIAL PRIMARY KEY,
    description TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS slot_banners (
    slot_id   BIGINT NOT NULL REFERENCES slots(id) ON DELETE CASCADE,
    banner_id BIGINT NOT NULL REFERENCES banners(id) ON DELETE CASCADE,
    PRIMARY KEY (slot_id, banner_id)
);

CREATE TABLE IF NOT EXISTS stats (
    slot_id   BIGINT NOT NULL REFERENCES slots(id) ON DELETE CASCADE,
    banner_id BIGINT NOT NULL REFERENCES banners(id) ON DELETE CASCADE,
    group_id  BIGINT NOT NULL REFERENCES social_groups(id) ON DELETE CASCADE,
    shows     BIGINT NOT NULL DEFAULT 0,
    clicks    BIGINT NOT NULL DEFAULT 0,
    PRIMARY KEY (slot_id, banner_id, group_id)
);
