-- Document templates
CREATE TABLE IF NOT EXISTS document_templates (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID REFERENCES workspaces(id) ON DELETE CASCADE,
    title        TEXT NOT NULL,
    description  TEXT NOT NULL DEFAULT '',
    category     TEXT NOT NULL DEFAULT 'other',
    thumbnail_url TEXT,
    content      JSONB NOT NULL DEFAULT '{}',
    is_public    BOOLEAN NOT NULL DEFAULT false,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_doc_templates_workspace
    ON document_templates(workspace_id);
CREATE INDEX idx_doc_templates_public
    ON document_templates(is_public)
    WHERE is_public = true;

-- Documents table
CREATE TABLE IF NOT EXISTS documents (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id     UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    title            TEXT NOT NULL,
    slug             TEXT UNIQUE NOT NULL,
    type             TEXT NOT NULL DEFAULT 'other',
    status           TEXT NOT NULL DEFAULT 'draft',
    content          JSONB NOT NULL DEFAULT '{}',
    template_id      UUID REFERENCES document_templates(id) ON DELETE SET NULL,
    pdf_url          TEXT,
    pdf_generated_at TIMESTAMPTZ,
    settings         JSONB NOT NULL DEFAULT '{}',
    is_published     BOOLEAN NOT NULL DEFAULT false,
    expires_at       TIMESTAMPTZ,
    created_by       UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_documents_workspace
    ON documents(workspace_id);
CREATE INDEX idx_documents_slug
    ON documents(slug);
CREATE INDEX idx_documents_created_at
    ON documents(workspace_id, created_at DESC);

-- Document views
CREATE TABLE IF NOT EXISTS document_views (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    document_id  UUID NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    viewer_email TEXT,
    ip_address   TEXT,
    device       TEXT,
    browser      TEXT,
    country      TEXT,
    duration_sec INTEGER NOT NULL DEFAULT 0,
    pages_viewed JSONB NOT NULL DEFAULT '[]',
    viewed_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_doc_views_document
    ON document_views(document_id);
CREATE INDEX idx_doc_views_date
    ON document_views(document_id, viewed_at DESC);

-- Document versions
CREATE TABLE IF NOT EXISTS document_versions (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    document_id    UUID NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    version_number INTEGER NOT NULL,
    content        JSONB NOT NULL DEFAULT '{}',
    created_by     UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_doc_versions_document
    ON document_versions(document_id);
