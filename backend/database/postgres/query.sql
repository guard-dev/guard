
-------------------- UserInfo Queries --------------------

-- name: AddUser :one
INSERT INTO user_info (email, full_name) VALUES ($1, $2) RETURNING *;

-- name: GetUserByEmail :one
SELECT * FROM user_info WHERE email = $1 LIMIT 1;

-- name: GetUserById :one
SELECT * FROM user_info WHERE user_id = $1 LIMIT 1;

-- name: GetExternalIdByEmail :one
SELECT external_id FROM user_info WHERE email = $1;



-------------------- Team Queries --------------------

-- name: GetTeamByTeamSlug :one
SELECT * FROM team WHERE team_slug = $1 LIMIT 1;

-- name: GetTeamByTeamId :one
SELECT * FROM team WHERE team_id = $1 LIMIT 1;

-- name: CreateNewTeam :one
INSERT INTO team (team_slug, team_name) VALUES ($1, $2) RETURNING *;

-- name: GetTeamsByUserId :many
SELECT team.*
FROM team
JOIN team_membership on team.team_id = team_membership.team_id
WHERE team_membership.user_id = $1
ORDER BY team.created;

-- name: UpdateTeamStripeCustomerIdByTeamId :one
UPDATE team SET stripe_customer_id = $2 WHERE team_id = $1 RETURNING *;

-- name: GetTeamByStripeCustomerId :one
SELECT * FROM team WHERE stripe_customer_id = $1;





-------------------- TeamMembership Queries --------------------

-- name: AddTeamMembership :one
INSERT INTO team_membership (team_id, user_id, membership_type)
VALUES ($1, $2, $3) RETURNING *;

-- name: GetTeamMembershipByTeamIdUserId :one
SELECT * FROM team_membership WHERE team_id = $1 AND user_id = $2 LIMIT 1;

-- name: GetTeamMembershipsByTeamId :many
SELECT * FROM team_membership WHERE team_id = $1 ORDER BY created;




-------------------- Project Queries --------------------

-- name: CreateNewProject :one
INSERT INTO project (
  team_id,
  project_slug,
  project_name
) VALUES ($1, $2, $3) RETURNING *;

-- name: GetProjectsByTeamId :many
SELECT * FROM project WHERE team_id = $1;

-- name: GetProjectByTeamIdAndProjectSlug :one
SELECT * FROM project WHERE team_id = $1 AND project_slug = $2;

-- name: DeleteProjectByTeamIdAndProjectSlug :one
DELETE FROM project WHERE team_id = $1 AND project_slug = $2 RETURNING *;

-- name: GetProjectByProjectId :one
SELECT * FROM project WHERE project_id = $1;


-------------------- Account Connection Queries --------------------

-- name: CreateAccountConnection :one
INSERT INTO account_connection (
  project_id,
  external_id,
  account_id
) VALUES ($1, $2, $3) RETURNING *;

-- name: GetConnectionsByProjectId :many
SELECT * FROM account_connection WHERE project_id = $1;


-------------------- Scan Queries --------------------

-- name: CreateNewScan :one
INSERT INTO scan (project_id, region_count, service_count, services, regions) VALUES ($1, $2, $3, $4, $5) RETURNING *;

-- name: IncrementScanResourceCostByScanId :one
UPDATE scan SET resource_cost = resource_cost + $1 WHERE scan_id = $2 RETURNING *;


-- name: GetScansByProjectId :many
SELECT * FROM scan WHERE project_id = $1;

-- name: UpdateScanCompletedStatus :one
UPDATE scan SET scan_completed = $1 WHERE scan_id = $2 RETURNING *;

-- name: GetScanByScanIdProjectId :one
SELECT * FROM scan WHERE project_id = $1 AND scan_id = $2;

-------------------- Scan Item Queries --------------------

-- name: GetScanItemByScanItemId :one
SELECT * FROM scan_item WHERE scan_id = $1;

-- name: CreateNewScanItem :one
INSERT INTO scan_item (
  scan_id,
  service,
  region,
  findings,
  summary,
  remedy
) VALUES ($1, $2, $3, $4, $5, $6) RETURNING *;

-- name: IncrementScanItemResourceCostByScanItemid :one
UPDATE scan_item SET resource_cost = resource_cost + $1 WHERE scan_item_id = $2 RETURNING *;

-- name: GetScanItemsByScanId :many
SELECT * FROM scan_item WHERE scan_id = $1;

-------------------- Scan Item Entry Queries --------------------

-- name: CreateNewScanItemEntry :one
INSERT INTO scan_item_entry (
  scan_item_id,
  findings,
  title,
  summary,
  remedy,
  commands
) VALUES ($1, $2, $3, $4, $5, $6) RETURNING *;

-- name: GetScanItemEntriesByScanId :many
SELECT * FROM scan_item_entry WHERE scan_item_id = $1;



-------------------- Subscription Plan Queries --------------------

-- name: IncrementSubscriptionResourcesUsedByTeamId :one
UPDATE subscription_plan SET resources_used = resources_used + $1 WHERE team_id = $2 RETURNING *;

-- name: CreateSubscription :one
INSERT INTO subscription_plan
(team_id, stripe_subscription_id, resources_included)
VALUES ($1, $2, $3) RETURNING *;

-- name: GetSubscriptionByTeamId :one
SELECT * FROM subscription_plan WHERE team_id = $1 ORDER BY created LIMIT 1;

-- name: GetSubscriptionByTeamIdSubscriptionId :one
SELECT * FROM subscription_plan WHERE team_id = $1 AND id = $2 LIMIT 1;

-- name: GetSubscriptionById :one
SELECT * FROM subscription_plan WHERE id = $1 LIMIT 1;

-- name: GetSubscriptionByStripeSubscriptionId :one
SELECT * FROM subscription_plan WHERE stripe_subscription_id = $1 LIMIT 1;

-- name: SetSubscriptionStripeIdByTeamId :one
UPDATE subscription_plan SET stripe_subscription_id = $2 WHERE team_id = $1 RETURNING *;

-- name: DeleteSubscriptionByStripeSubscriptionId :one
DELETE FROM subscription_plan WHERE stripe_subscription_id = $1 RETURNING *;

-- name: ResetSubscriptionResourcesUsed :one
UPDATE subscription_plan
SET resources_used = 0
WHERE team_id = $1
RETURNING *;

