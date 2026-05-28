-- Events table (central event log)
CREATE TABLE IF NOT EXISTS events (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    type         TEXT NOT NULL,
    source       TEXT NOT NULL,
    payload      JSONB NOT NULL DEFAULT '{}',
    occurred_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    processed_at TIMESTAMPTZ,
    error        TEXT
);

CREATE INDEX idx_events_workspace_type
    ON events(workspace_id, type);
CREATE INDEX idx_events_workspace_occurred
    ON events(workspace_id, occurred_at DESC);
CREATE INDEX idx_events_type_occurred
    ON events(type, occurred_at DESC);

-- Flow blueprints (automations)
CREATE TABLE IF NOT EXISTS flow_blueprints (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id   UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    name           TEXT NOT NULL,
    description    TEXT NOT NULL DEFAULT '',
    trigger_type   TEXT NOT NULL,
    trigger_config JSONB NOT NULL DEFAULT '{}',
    actions        JSONB NOT NULL DEFAULT '[]',
    is_active      BOOLEAN NOT NULL DEFAULT true,
    run_count      INTEGER NOT NULL DEFAULT 0,
    last_run_at    TIMESTAMPTZ,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_blueprints_workspace
    ON flow_blueprints(workspace_id);
CREATE INDEX idx_blueprints_trigger
    ON flow_blueprints(workspace_id, trigger_type)
    WHERE is_active = true;
CREATE INDEX idx_blueprints_active
    ON flow_blueprints(workspace_id, is_active);

-- Blueprint execution log
CREATE TABLE IF NOT EXISTS blueprint_executions (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    blueprint_id UUID NOT NULL REFERENCES flow_blueprints(id) ON DELETE CASCADE,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    event_id     UUID REFERENCES events(id) ON DELETE SET NULL,
    status       TEXT NOT NULL DEFAULT 'running',
    actions_run  INTEGER NOT NULL DEFAULT 0,
    actions_failed INTEGER NOT NULL DEFAULT 0,
    error        TEXT,
    started_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ
);

CREATE INDEX idx_executions_blueprint
    ON blueprint_executions(blueprint_id);
CREATE INDEX idx_executions_workspace
    ON blueprint_executions(workspace_id, started_at DESC);

-- AI conversations
CREATE TABLE IF NOT EXISTS ai_conversations (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id  UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    user_id       UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title         TEXT NOT NULL DEFAULT 'New conversation',
    message_count INTEGER NOT NULL DEFAULT 0,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_ai_conversations_workspace
    ON ai_conversations(workspace_id);
CREATE INDEX idx_ai_conversations_user
    ON ai_conversations(user_id);

-- AI messages
CREATE TABLE IF NOT EXISTS ai_messages (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id UUID NOT NULL REFERENCES ai_conversations(id) ON DELETE CASCADE,
    role            TEXT NOT NULL,
    content         TEXT NOT NULL,
    tool_calls      JSONB NOT NULL DEFAULT '[]',
    tool_results    JSONB NOT NULL DEFAULT '[]',
    tokens_used     INTEGER NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_ai_messages_conversation
    ON ai_messages(conversation_id);
CREATE INDEX idx_ai_messages_created
    ON ai_messages(conversation_id, created_at ASC);

-- AI usage tracking
CREATE TABLE IF NOT EXISTS ai_usage (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    user_id      UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    command      TEXT NOT NULL,
    tokens_used  INTEGER NOT NULL DEFAULT 0,
    tools_called JSONB NOT NULL DEFAULT '[]',
    success      BOOLEAN NOT NULL DEFAULT true,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_ai_usage_workspace
    ON ai_usage(workspace_id);
CREATE INDEX idx_ai_usage_created
    ON ai_usage(workspace_id, created_at DESC);
