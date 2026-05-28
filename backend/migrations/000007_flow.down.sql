-- Drop AI usage tracking
DROP TABLE IF EXISTS ai_usage CASCADE;

-- Drop AI messages
DROP TABLE IF EXISTS ai_messages CASCADE;

-- Drop AI conversations
DROP TABLE IF EXISTS ai_conversations CASCADE;

-- Drop blueprint execution log
DROP TABLE IF EXISTS blueprint_executions CASCADE;

-- Drop flow blueprints
DROP TABLE IF EXISTS flow_blueprints CASCADE;

-- Drop events table
DROP TABLE IF EXISTS events CASCADE;
