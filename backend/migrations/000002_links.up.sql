-- Link folders
CREATE TABLE IF NOT EXISTS link_folders (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    name         TEXT NOT NULL,
    parent_id    UUID REFERENCES link_folders(id) ON DELETE SET NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_link_folders_workspace
    ON link_folders(workspace_id);

-- Links table
CREATE TABLE IF NOT EXISTS links (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id  UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    original_url  TEXT NOT NULL,
    short_code    TEXT UNIQUE NOT NULL,
    custom_slug   TEXT UNIQUE,
    custom_domain TEXT,
    title         TEXT NOT NULL DEFAULT '',
    description   TEXT NOT NULL DEFAULT '',
    tags          JSONB NOT NULL DEFAULT '[]',
    folder_id     UUID REFERENCES link_folders(id) ON DELETE SET NULL,
    password      TEXT,
    expires_at    TIMESTAMPTZ,
    click_limit   INTEGER,
    total_clicks  INTEGER NOT NULL DEFAULT 0,
    unique_clicks INTEGER NOT NULL DEFAULT 0,
    is_active     BOOLEAN NOT NULL DEFAULT true,
    utm_params    JSONB NOT NULL DEFAULT '{}',
    settings      JSONB NOT NULL DEFAULT '{}',
    created_by    UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_links_workspace
    ON links(workspace_id);
CREATE INDEX idx_links_short_code
    ON links(short_code);
CREATE INDEX idx_links_custom_slug
    ON links(custom_slug)
    WHERE custom_slug IS NOT NULL;
CREATE INDEX idx_links_created_at
    ON links(workspace_id, created_at DESC);

-- Bio link pages
CREATE TABLE IF NOT EXISTS bio_pages (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    username     TEXT UNIQUE NOT NULL,
    title        TEXT NOT NULL DEFAULT '',
    bio          TEXT NOT NULL DEFAULT '',
    avatar_url   TEXT,
    links        JSONB NOT NULL DEFAULT '[]',
    social_links JSONB NOT NULL DEFAULT '{}',
    theme        JSONB NOT NULL DEFAULT '{}',
    is_published BOOLEAN NOT NULL DEFAULT false,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_bio_pages_username
    ON bio_pages(username);
CREATE INDEX idx_bio_pages_workspace
    ON bio_pages(workspace_id);
