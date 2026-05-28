-- Forms table
CREATE TABLE IF NOT EXISTS forms (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    title        TEXT NOT NULL,
    description  TEXT NOT NULL DEFAULT '',
    slug         TEXT UNIQUE NOT NULL,
    settings     JSONB NOT NULL DEFAULT '{}',
    is_published BOOLEAN NOT NULL DEFAULT false,
    published_at TIMESTAMPTZ,
    created_by   UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_forms_workspace
    ON forms(workspace_id);
CREATE INDEX idx_forms_slug
    ON forms(slug);
CREATE INDEX idx_forms_created_at
    ON forms(workspace_id, created_at DESC);

-- Form fields
CREATE TABLE IF NOT EXISTS form_fields (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    form_id      UUID NOT NULL REFERENCES forms(id) ON DELETE CASCADE,
    type         TEXT NOT NULL,
    label        TEXT NOT NULL,
    placeholder  TEXT NOT NULL DEFAULT '',
    helper_text  TEXT NOT NULL DEFAULT '',
    required     BOOLEAN NOT NULL DEFAULT false,
    options      JSONB NOT NULL DEFAULT '[]',
    validation   JSONB NOT NULL DEFAULT '{}',
    logic        JSONB NOT NULL DEFAULT '[]',
    field_order  INTEGER NOT NULL DEFAULT 0,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_form_fields_form
    ON form_fields(form_id);
CREATE INDEX idx_form_fields_order
    ON form_fields(form_id, field_order);

-- Form responses
CREATE TABLE IF NOT EXISTS form_responses (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    form_id         UUID NOT NULL REFERENCES forms(id) ON DELETE CASCADE,
    workspace_id    UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    data            JSONB NOT NULL DEFAULT '{}',
    score           INTEGER,
    ip_address      TEXT,
    user_agent      TEXT,
    referrer        TEXT,
    completion_time INTEGER NOT NULL DEFAULT 0,
    is_complete     BOOLEAN NOT NULL DEFAULT true,
    submitted_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_form_responses_form
    ON form_responses(form_id);
CREATE INDEX idx_form_responses_workspace
    ON form_responses(workspace_id);
CREATE INDEX idx_form_responses_submitted
    ON form_responses(form_id, submitted_at DESC);

-- Form views (analytics)
CREATE TABLE IF NOT EXISTS form_views (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    form_id    UUID NOT NULL REFERENCES forms(id) ON DELETE CASCADE,
    ip_address TEXT,
    device     TEXT,
    referrer   TEXT,
    viewed_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_form_views_form
    ON form_views(form_id);
CREATE INDEX idx_form_views_date
    ON form_views(form_id, viewed_at DESC);
