-- Drop password reset tokens
DROP TABLE IF EXISTS password_resets CASCADE;

-- Drop email verification tokens
DROP TABLE IF EXISTS email_verifications CASCADE;

-- Drop usage records
DROP TABLE IF EXISTS usage_records CASCADE;

-- Drop subscriptions
DROP TABLE IF EXISTS subscriptions CASCADE;

-- Drop workspace invitations
DROP TABLE IF EXISTS workspace_invitations CASCADE;

-- Drop workspace members
DROP TABLE IF EXISTS workspace_members CASCADE;

-- Drop workspaces
DROP TABLE IF EXISTS workspaces CASCADE;

-- Drop users
DROP TABLE IF EXISTS users CASCADE;

-- Drop pgcrypto extension
DROP EXTENSION IF EXISTS "pgcrypto" CASCADE;
