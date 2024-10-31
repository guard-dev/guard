DROP TYPE IF EXISTS membership_type CASCADE;
CREATE TYPE membership_type AS ENUM ('OWNER', 'ADMIN', 'MEMBER');

DROP TABLE IF EXISTS user_info CASCADE;
CREATE TABLE user_info (
  user_id BIGSERIAL PRIMARY KEY NOT NULL,
  email TEXT UNIQUE NOT NULL,
  full_name TEXT NOT NULL,
  external_id UUID NOT NULL DEFAULT gen_random_uuid(),  -- New column for storing the External ID
  created TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

DROP TABLE IF EXISTS team CASCADE;
CREATE TABLE team (
  team_id BIGSERIAL PRIMARY KEY NOT NULL,
  team_slug TEXT UNIQUE NOT NULL,
  team_name TEXT NOT NULL,
  stripe_customer_id TEXT UNIQUE,
  created TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

DROP TABLE IF EXISTS team_membership CASCADE;
CREATE TABLE team_membership (
  team_membership_id BIGSERIAL PRIMARY KEY NOT NULL,
  team_id BIGINT REFERENCES team (team_id) ON DELETE CASCADE ON UPDATE CASCADE NOT NULL,
  user_id BIGINT REFERENCES user_info (user_id) ON DELETE CASCADE ON UPDATE CASCADE NOT NULL,
  membership_type MEMBERSHIP_TYPE NOT NULL,
  created TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE (team_id, user_id)
);

DROP TABLE IF EXISTS team_invite CASCADE;
CREATE TABLE team_invite (
  team_invite_id BIGSERIAL PRIMARY KEY NOT NULL,
  invite_code TEXT UNIQUE NOT NULL,
  team_id BIGINT REFERENCES team (team_id) ON DELETE CASCADE ON UPDATE CASCADE NOT NULL,
  invitee_email TEXT NOT NULL,
  created TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE (team_id, invitee_email)
);

DROP TABLE IF EXISTS project CASCADE;
CREATE TABLE project (
  project_id BIGSERIAL PRIMARY KEY NOT NULL,
  team_id BIGINT REFERENCES team (team_id) ON DELETE CASCADE ON UPDATE CASCADE NOT NULL,
  project_slug TEXT UNIQUE NOT NULL,
  project_name TEXT NOT NULL,
  created TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

DROP TABLE IF EXISTS account_connection CASCADE;
CREATE TABLE account_connection (
  connection_id BIGSERIAL PRIMARY KEY NOT NULL,
  project_id BIGINT REFERENCES project (project_id) ON DELETE CASCADE ON UPDATE CASCADE NOT NULL,
  external_id UUID NOT NULL,
  account_id TEXT NOT NULL,
  created TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

DROP TABLE IF EXISTS scan CASCADE;
CREATE TABLE scan (
  scan_id UUID PRIMARY KEY NOT NULL DEFAULT gen_random_uuid(),
  project_id BIGINT REFERENCES project (project_id) ON DELETE CASCADE ON UPDATE CASCADE NOT NULL,
  scan_completed BOOLEAN NOT NULL DEFAULT false,
  regions TEXT[],
  services TEXT[],
  service_count INT NOT NULL DEFAULT 0,
  region_count INT NOT NULL DEFAULT 0,
  resource_cost INT NOT NULL DEFAULT 0,  -- Total resource cost for the scan
  created TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

DROP TABLE IF EXISTS scan_item CASCADE;
CREATE TABLE scan_item (
  scan_item_id BIGSERIAL PRIMARY KEY NOT NULL,
  scan_id UUID REFERENCES scan (scan_id) ON DELETE CASCADE ON UPDATE CASCADE NOT NULL,
  service TEXT NOT NULL,
  region TEXT NOT NULL,
  resource_cost INT NOT NULL DEFAULT 0,  -- Total resource cost for each scan item
  findings TEXT[],
  summary TEXT NOT NULL,
  remedy TEXT NOT NULL,
  created TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

DROP TABLE IF EXISTS scan_item_entry CASCADE;
CREATE TABLE scan_item_entry (
  scan_item_entry_id BIGSERIAL PRIMARY KEY NOT NULL,
  scan_item_id BIGINT REFERENCES scan_item (scan_item_id) ON DELETE CASCADE ON UPDATE CASCADE NOT NULL,
  findings TEXT[],
  title TEXT NOT NULL,
  summary TEXT NOT NULL,
  remedy TEXT NOT NULL,
  commands TEXT[],
  resource_cost INT NOT NULL DEFAULT 1,  -- Resource cost for each scan item entry
  created TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

DROP TABLE IF EXISTS subscription_plan CASCADE;
CREATE TABLE subscription_plan (
  id BIGSERIAL PRIMARY KEY NOT NULL,
  team_id BIGINT REFERENCES team (team_id) ON DELETE CASCADE UNIQUE NOT NULL,
  stripe_subscription_id TEXT UNIQUE,
  resources_included INT NOT NULL DEFAULT 0,     -- Included resources per plan
  resources_used INT NOT NULL DEFAULT 0,                 -- Tracks total resources used
  created TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
