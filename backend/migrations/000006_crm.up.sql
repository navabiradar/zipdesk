-- CRM pipelines
CREATE TABLE IF NOT EXISTS crm_pipelines (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    name         TEXT NOT NULL,
    is_default   BOOLEAN NOT NULL DEFAULT false,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_crm_pipelines_workspace
    ON crm_pipelines(workspace_id);

-- CRM stages
CREATE TABLE IF NOT EXISTS crm_stages (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    pipeline_id UUID NOT NULL REFERENCES crm_pipelines(id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    probability INTEGER NOT NULL DEFAULT 0,
    stage_order INTEGER NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_crm_stages_pipeline
    ON crm_stages(pipeline_id);

-- CRM companies
CREATE TABLE IF NOT EXISTS crm_companies (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    name         TEXT NOT NULL,
    industry     TEXT NOT NULL DEFAULT '',
    size         TEXT NOT NULL DEFAULT '',
    website      TEXT NOT NULL DEFAULT '',
    phone        TEXT NOT NULL DEFAULT '',
    address      JSONB NOT NULL DEFAULT '{}',
    owner_id     UUID REFERENCES users(id) ON DELETE SET NULL,
    tags         JSONB NOT NULL DEFAULT '[]',
    custom_fields JSONB NOT NULL DEFAULT '{}',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_crm_companies_workspace
    ON crm_companies(workspace_id);

-- CRM contacts
CREATE TABLE IF NOT EXISTS crm_contacts (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id  UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    first_name    TEXT NOT NULL DEFAULT '',
    last_name     TEXT NOT NULL DEFAULT '',
    email         TEXT NOT NULL DEFAULT '',
    phone         TEXT NOT NULL DEFAULT '',
    job_title     TEXT NOT NULL DEFAULT '',
    company_id    UUID REFERENCES crm_companies(id) ON DELETE SET NULL,
    lead_source   TEXT NOT NULL DEFAULT '',
    lead_status   TEXT NOT NULL DEFAULT 'new',
    lead_score    INTEGER NOT NULL DEFAULT 0,
    owner_id      UUID REFERENCES users(id) ON DELETE SET NULL,
    tags          JSONB NOT NULL DEFAULT '[]',
    custom_fields JSONB NOT NULL DEFAULT '{}',
    last_activity_at TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_crm_contacts_workspace
    ON crm_contacts(workspace_id);
CREATE INDEX idx_crm_contacts_email
    ON crm_contacts(workspace_id, email);
CREATE INDEX idx_crm_contacts_score
    ON crm_contacts(workspace_id, lead_score DESC);

-- CRM deals
CREATE TABLE IF NOT EXISTS crm_deals (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id   UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    title          TEXT NOT NULL,
    contact_id     UUID REFERENCES crm_contacts(id) ON DELETE SET NULL,
    company_id     UUID REFERENCES crm_companies(id) ON DELETE SET NULL,
    pipeline_id    UUID REFERENCES crm_pipelines(id) ON DELETE SET NULL,
    stage_id       UUID REFERENCES crm_stages(id) ON DELETE SET NULL,
    value          NUMERIC(15,2) NOT NULL DEFAULT 0,
    currency       TEXT NOT NULL DEFAULT 'USD',
    probability    INTEGER NOT NULL DEFAULT 0,
    expected_close DATE,
    owner_id       UUID REFERENCES users(id) ON DELETE SET NULL,
    lost_reason    TEXT,
    tags           JSONB NOT NULL DEFAULT '[]',
    custom_fields  JSONB NOT NULL DEFAULT '{}',
    won_at         TIMESTAMPTZ,
    lost_at        TIMESTAMPTZ,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_crm_deals_workspace
    ON crm_deals(workspace_id);
CREATE INDEX idx_crm_deals_pipeline
    ON crm_deals(pipeline_id, stage_id);
CREATE INDEX idx_crm_deals_contact
    ON crm_deals(contact_id);

-- CRM activities
CREATE TABLE IF NOT EXISTS crm_activities (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    type         TEXT NOT NULL,
    title        TEXT NOT NULL,
    description  TEXT NOT NULL DEFAULT '',
    contact_id   UUID REFERENCES crm_contacts(id) ON DELETE CASCADE,
    company_id   UUID REFERENCES crm_companies(id) ON DELETE CASCADE,
    deal_id      UUID REFERENCES crm_deals(id) ON DELETE CASCADE,
    owner_id     UUID REFERENCES users(id) ON DELETE SET NULL,
    due_at       TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    outcome      TEXT,
    duration_mins INTEGER NOT NULL DEFAULT 0,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_crm_activities_workspace
    ON crm_activities(workspace_id);
CREATE INDEX idx_crm_activities_contact
    ON crm_activities(contact_id);
CREATE INDEX idx_crm_activities_deal
    ON crm_activities(deal_id);

-- CRM notes
CREATE TABLE IF NOT EXISTS crm_notes (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    content      TEXT NOT NULL,
    contact_id   UUID REFERENCES crm_contacts(id) ON DELETE CASCADE,
    company_id   UUID REFERENCES crm_companies(id) ON DELETE CASCADE,
    deal_id      UUID REFERENCES crm_deals(id) ON DELETE CASCADE,
    author_id    UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_crm_notes_contact
    ON crm_notes(contact_id);
CREATE INDEX idx_crm_notes_deal
    ON crm_notes(deal_id);
