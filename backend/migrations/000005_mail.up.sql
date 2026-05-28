-- Mail lists
CREATE TABLE IF NOT EXISTS mail_lists (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id  UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    name          TEXT NOT NULL,
    description   TEXT NOT NULL DEFAULT '',
    contact_count INTEGER NOT NULL DEFAULT 0,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_mail_lists_workspace
    ON mail_lists(workspace_id);

-- Mail contacts
CREATE TABLE IF NOT EXISTS mail_contacts (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id    UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    email           TEXT NOT NULL,
    first_name      TEXT NOT NULL DEFAULT '',
    last_name       TEXT NOT NULL DEFAULT '',
    company         TEXT NOT NULL DEFAULT '',
    phone           TEXT NOT NULL DEFAULT '',
    tags            JSONB NOT NULL DEFAULT '[]',
    custom_fields   JSONB NOT NULL DEFAULT '{}',
    status          TEXT NOT NULL DEFAULT 'subscribed',
    source          TEXT NOT NULL DEFAULT 'manual',
    subscribed_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    unsubscribed_at TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(workspace_id, email)
);

CREATE INDEX idx_mail_contacts_workspace
    ON mail_contacts(workspace_id);
CREATE INDEX idx_mail_contacts_email
    ON mail_contacts(workspace_id, email);
CREATE INDEX idx_mail_contacts_status
    ON mail_contacts(workspace_id, status);

-- Mail list contacts (junction)
CREATE TABLE IF NOT EXISTS mail_list_contacts (
    list_id      UUID NOT NULL REFERENCES mail_lists(id) ON DELETE CASCADE,
    contact_id   UUID NOT NULL REFERENCES mail_contacts(id) ON DELETE CASCADE,
    status       TEXT NOT NULL DEFAULT 'subscribed',
    subscribed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY(list_id, contact_id)
);

CREATE INDEX idx_list_contacts_contact
    ON mail_list_contacts(contact_id);

-- Mail campaigns
CREATE TABLE IF NOT EXISTS mail_campaigns (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    name         TEXT NOT NULL,
    subject      TEXT NOT NULL,
    preview_text TEXT NOT NULL DEFAULT '',
    from_name    TEXT NOT NULL,
    from_email   TEXT NOT NULL,
    content      JSONB NOT NULL DEFAULT '{}',
    list_id      UUID REFERENCES mail_lists(id) ON DELETE SET NULL,
    status       TEXT NOT NULL DEFAULT 'draft',
    scheduled_at TIMESTAMPTZ,
    sent_at      TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_campaigns_workspace
    ON mail_campaigns(workspace_id);
CREATE INDEX idx_campaigns_status
    ON mail_campaigns(workspace_id, status);

-- Campaign stats
CREATE TABLE IF NOT EXISTS mail_campaign_stats (
    campaign_id   UUID PRIMARY KEY REFERENCES mail_campaigns(id) ON DELETE CASCADE,
    sent          INTEGER NOT NULL DEFAULT 0,
    delivered     INTEGER NOT NULL DEFAULT 0,
    opened        INTEGER NOT NULL DEFAULT 0,
    clicked       INTEGER NOT NULL DEFAULT 0,
    bounced       INTEGER NOT NULL DEFAULT 0,
    unsubscribed  INTEGER NOT NULL DEFAULT 0,
    spam_reported INTEGER NOT NULL DEFAULT 0,
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Mail automations
CREATE TABLE IF NOT EXISTS mail_automations (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    name         TEXT NOT NULL,
    trigger      JSONB NOT NULL DEFAULT '{}',
    steps        JSONB NOT NULL DEFAULT '[]',
    is_active    BOOLEAN NOT NULL DEFAULT false,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_automations_workspace
    ON mail_automations(workspace_id);
CREATE INDEX idx_automations_active
    ON mail_automations(workspace_id, is_active)
    WHERE is_active = true;
