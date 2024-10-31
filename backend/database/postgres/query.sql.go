// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.27.0
// source: query.sql

package postgres

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

const addTeamMembership = `-- name: AddTeamMembership :one

INSERT INTO team_membership (team_id, user_id, membership_type)
VALUES ($1, $2, $3) RETURNING team_membership_id, team_id, user_id, membership_type, created
`

type AddTeamMembershipParams struct {
	TeamID         int64
	UserID         int64
	MembershipType MembershipType
}

// ------------------ TeamMembership Queries --------------------
func (q *Queries) AddTeamMembership(ctx context.Context, arg AddTeamMembershipParams) (TeamMembership, error) {
	row := q.db.QueryRowContext(ctx, addTeamMembership, arg.TeamID, arg.UserID, arg.MembershipType)
	var i TeamMembership
	err := row.Scan(
		&i.TeamMembershipID,
		&i.TeamID,
		&i.UserID,
		&i.MembershipType,
		&i.Created,
	)
	return i, err
}

const addUser = `-- name: AddUser :one

INSERT INTO user_info (email, full_name) VALUES ($1, $2) RETURNING user_id, email, full_name, external_id, created
`

type AddUserParams struct {
	Email    string
	FullName string
}

// ------------------ UserInfo Queries --------------------
func (q *Queries) AddUser(ctx context.Context, arg AddUserParams) (UserInfo, error) {
	row := q.db.QueryRowContext(ctx, addUser, arg.Email, arg.FullName)
	var i UserInfo
	err := row.Scan(
		&i.UserID,
		&i.Email,
		&i.FullName,
		&i.ExternalID,
		&i.Created,
	)
	return i, err
}

const createAccountConnection = `-- name: CreateAccountConnection :one

INSERT INTO account_connection (
  project_id,
  external_id,
  account_id
) VALUES ($1, $2, $3) RETURNING connection_id, project_id, external_id, account_id, created
`

type CreateAccountConnectionParams struct {
	ProjectID  int64
	ExternalID uuid.UUID
	AccountID  string
}

// ------------------ Account Connection Queries --------------------
func (q *Queries) CreateAccountConnection(ctx context.Context, arg CreateAccountConnectionParams) (AccountConnection, error) {
	row := q.db.QueryRowContext(ctx, createAccountConnection, arg.ProjectID, arg.ExternalID, arg.AccountID)
	var i AccountConnection
	err := row.Scan(
		&i.ConnectionID,
		&i.ProjectID,
		&i.ExternalID,
		&i.AccountID,
		&i.Created,
	)
	return i, err
}

const createNewProject = `-- name: CreateNewProject :one

INSERT INTO project (
  team_id,
  project_slug,
  project_name
) VALUES ($1, $2, $3) RETURNING project_id, team_id, project_slug, project_name, created
`

type CreateNewProjectParams struct {
	TeamID      int64
	ProjectSlug string
	ProjectName string
}

// ------------------ Project Queries --------------------
func (q *Queries) CreateNewProject(ctx context.Context, arg CreateNewProjectParams) (Project, error) {
	row := q.db.QueryRowContext(ctx, createNewProject, arg.TeamID, arg.ProjectSlug, arg.ProjectName)
	var i Project
	err := row.Scan(
		&i.ProjectID,
		&i.TeamID,
		&i.ProjectSlug,
		&i.ProjectName,
		&i.Created,
	)
	return i, err
}

const createNewScan = `-- name: CreateNewScan :one

INSERT INTO scan (project_id, region_count, service_count) VALUES ($1, $2, $3) RETURNING scan_id, project_id, scan_completed, service_count, region_count, resource_cost, created
`

type CreateNewScanParams struct {
	ProjectID    int64
	RegionCount  int32
	ServiceCount int32
}

// ------------------ Scan Queries --------------------
func (q *Queries) CreateNewScan(ctx context.Context, arg CreateNewScanParams) (Scan, error) {
	row := q.db.QueryRowContext(ctx, createNewScan, arg.ProjectID, arg.RegionCount, arg.ServiceCount)
	var i Scan
	err := row.Scan(
		&i.ScanID,
		&i.ProjectID,
		&i.ScanCompleted,
		&i.ServiceCount,
		&i.RegionCount,
		&i.ResourceCost,
		&i.Created,
	)
	return i, err
}

const createNewScanItem = `-- name: CreateNewScanItem :one
INSERT INTO scan_item (
  scan_id,
  service,
  region,
  findings,
  summary,
  remedy
) VALUES ($1, $2, $3, $4, $5, $6) RETURNING scan_item_id, scan_id, service, region, resource_cost, findings, summary, remedy, created
`

type CreateNewScanItemParams struct {
	ScanID   uuid.UUID
	Service  string
	Region   string
	Findings []string
	Summary  string
	Remedy   string
}

func (q *Queries) CreateNewScanItem(ctx context.Context, arg CreateNewScanItemParams) (ScanItem, error) {
	row := q.db.QueryRowContext(ctx, createNewScanItem,
		arg.ScanID,
		arg.Service,
		arg.Region,
		pq.Array(arg.Findings),
		arg.Summary,
		arg.Remedy,
	)
	var i ScanItem
	err := row.Scan(
		&i.ScanItemID,
		&i.ScanID,
		&i.Service,
		&i.Region,
		&i.ResourceCost,
		pq.Array(&i.Findings),
		&i.Summary,
		&i.Remedy,
		&i.Created,
	)
	return i, err
}

const createNewScanItemEntry = `-- name: CreateNewScanItemEntry :one

INSERT INTO scan_item_entry (
  scan_item_id,
  findings,
  title,
  summary,
  remedy,
  commands
) VALUES ($1, $2, $3, $4, $5, $6) RETURNING scan_item_entry_id, scan_item_id, findings, title, summary, remedy, commands, resource_cost, created
`

type CreateNewScanItemEntryParams struct {
	ScanItemID int64
	Findings   []string
	Title      string
	Summary    string
	Remedy     string
	Commands   []string
}

// ------------------ Scan Item Entry Queries --------------------
func (q *Queries) CreateNewScanItemEntry(ctx context.Context, arg CreateNewScanItemEntryParams) (ScanItemEntry, error) {
	row := q.db.QueryRowContext(ctx, createNewScanItemEntry,
		arg.ScanItemID,
		pq.Array(arg.Findings),
		arg.Title,
		arg.Summary,
		arg.Remedy,
		pq.Array(arg.Commands),
	)
	var i ScanItemEntry
	err := row.Scan(
		&i.ScanItemEntryID,
		&i.ScanItemID,
		pq.Array(&i.Findings),
		&i.Title,
		&i.Summary,
		&i.Remedy,
		pq.Array(&i.Commands),
		&i.ResourceCost,
		&i.Created,
	)
	return i, err
}

const createNewTeam = `-- name: CreateNewTeam :one
INSERT INTO team (team_slug, team_name) VALUES ($1, $2) RETURNING team_id, team_slug, team_name, stripe_customer_id, created
`

type CreateNewTeamParams struct {
	TeamSlug string
	TeamName string
}

func (q *Queries) CreateNewTeam(ctx context.Context, arg CreateNewTeamParams) (Team, error) {
	row := q.db.QueryRowContext(ctx, createNewTeam, arg.TeamSlug, arg.TeamName)
	var i Team
	err := row.Scan(
		&i.TeamID,
		&i.TeamSlug,
		&i.TeamName,
		&i.StripeCustomerID,
		&i.Created,
	)
	return i, err
}

const createSubscription = `-- name: CreateSubscription :one
INSERT INTO subscription_plan
(team_id, stripe_subscription_id, resources_included)
VALUES ($1, $2, $3) RETURNING id, team_id, stripe_subscription_id, resources_included, resources_used, created
`

type CreateSubscriptionParams struct {
	TeamID               int64
	StripeSubscriptionID sql.NullString
	ResourcesIncluded    int32
}

func (q *Queries) CreateSubscription(ctx context.Context, arg CreateSubscriptionParams) (SubscriptionPlan, error) {
	row := q.db.QueryRowContext(ctx, createSubscription, arg.TeamID, arg.StripeSubscriptionID, arg.ResourcesIncluded)
	var i SubscriptionPlan
	err := row.Scan(
		&i.ID,
		&i.TeamID,
		&i.StripeSubscriptionID,
		&i.ResourcesIncluded,
		&i.ResourcesUsed,
		&i.Created,
	)
	return i, err
}

const deleteProjectByTeamIdAndProjectSlug = `-- name: DeleteProjectByTeamIdAndProjectSlug :one
DELETE FROM project WHERE team_id = $1 AND project_slug = $2 RETURNING project_id, team_id, project_slug, project_name, created
`

type DeleteProjectByTeamIdAndProjectSlugParams struct {
	TeamID      int64
	ProjectSlug string
}

func (q *Queries) DeleteProjectByTeamIdAndProjectSlug(ctx context.Context, arg DeleteProjectByTeamIdAndProjectSlugParams) (Project, error) {
	row := q.db.QueryRowContext(ctx, deleteProjectByTeamIdAndProjectSlug, arg.TeamID, arg.ProjectSlug)
	var i Project
	err := row.Scan(
		&i.ProjectID,
		&i.TeamID,
		&i.ProjectSlug,
		&i.ProjectName,
		&i.Created,
	)
	return i, err
}

const deleteSubscriptionByStripeSubscriptionId = `-- name: DeleteSubscriptionByStripeSubscriptionId :one
DELETE FROM subscription_plan WHERE stripe_subscription_id = $1 RETURNING id, team_id, stripe_subscription_id, resources_included, resources_used, created
`

func (q *Queries) DeleteSubscriptionByStripeSubscriptionId(ctx context.Context, stripeSubscriptionID sql.NullString) (SubscriptionPlan, error) {
	row := q.db.QueryRowContext(ctx, deleteSubscriptionByStripeSubscriptionId, stripeSubscriptionID)
	var i SubscriptionPlan
	err := row.Scan(
		&i.ID,
		&i.TeamID,
		&i.StripeSubscriptionID,
		&i.ResourcesIncluded,
		&i.ResourcesUsed,
		&i.Created,
	)
	return i, err
}

const getConnectionsByProjectId = `-- name: GetConnectionsByProjectId :many
SELECT connection_id, project_id, external_id, account_id, created FROM account_connection WHERE project_id = $1
`

func (q *Queries) GetConnectionsByProjectId(ctx context.Context, projectID int64) ([]AccountConnection, error) {
	rows, err := q.db.QueryContext(ctx, getConnectionsByProjectId, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []AccountConnection
	for rows.Next() {
		var i AccountConnection
		if err := rows.Scan(
			&i.ConnectionID,
			&i.ProjectID,
			&i.ExternalID,
			&i.AccountID,
			&i.Created,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const getExternalIdByEmail = `-- name: GetExternalIdByEmail :one
SELECT external_id FROM user_info WHERE email = $1
`

func (q *Queries) GetExternalIdByEmail(ctx context.Context, email string) (uuid.UUID, error) {
	row := q.db.QueryRowContext(ctx, getExternalIdByEmail, email)
	var external_id uuid.UUID
	err := row.Scan(&external_id)
	return external_id, err
}

const getProjectByProjectId = `-- name: GetProjectByProjectId :one
SELECT project_id, team_id, project_slug, project_name, created FROM project WHERE project_id = $1
`

func (q *Queries) GetProjectByProjectId(ctx context.Context, projectID int64) (Project, error) {
	row := q.db.QueryRowContext(ctx, getProjectByProjectId, projectID)
	var i Project
	err := row.Scan(
		&i.ProjectID,
		&i.TeamID,
		&i.ProjectSlug,
		&i.ProjectName,
		&i.Created,
	)
	return i, err
}

const getProjectByTeamIdAndProjectSlug = `-- name: GetProjectByTeamIdAndProjectSlug :one
SELECT project_id, team_id, project_slug, project_name, created FROM project WHERE team_id = $1 AND project_slug = $2
`

type GetProjectByTeamIdAndProjectSlugParams struct {
	TeamID      int64
	ProjectSlug string
}

func (q *Queries) GetProjectByTeamIdAndProjectSlug(ctx context.Context, arg GetProjectByTeamIdAndProjectSlugParams) (Project, error) {
	row := q.db.QueryRowContext(ctx, getProjectByTeamIdAndProjectSlug, arg.TeamID, arg.ProjectSlug)
	var i Project
	err := row.Scan(
		&i.ProjectID,
		&i.TeamID,
		&i.ProjectSlug,
		&i.ProjectName,
		&i.Created,
	)
	return i, err
}

const getProjectsByTeamId = `-- name: GetProjectsByTeamId :many
SELECT project_id, team_id, project_slug, project_name, created FROM project WHERE team_id = $1
`

func (q *Queries) GetProjectsByTeamId(ctx context.Context, teamID int64) ([]Project, error) {
	rows, err := q.db.QueryContext(ctx, getProjectsByTeamId, teamID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []Project
	for rows.Next() {
		var i Project
		if err := rows.Scan(
			&i.ProjectID,
			&i.TeamID,
			&i.ProjectSlug,
			&i.ProjectName,
			&i.Created,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const getScanByScanIdProjectId = `-- name: GetScanByScanIdProjectId :one
SELECT scan_id, project_id, scan_completed, service_count, region_count, resource_cost, created FROM scan WHERE project_id = $1 AND scan_id = $2
`

type GetScanByScanIdProjectIdParams struct {
	ProjectID int64
	ScanID    uuid.UUID
}

func (q *Queries) GetScanByScanIdProjectId(ctx context.Context, arg GetScanByScanIdProjectIdParams) (Scan, error) {
	row := q.db.QueryRowContext(ctx, getScanByScanIdProjectId, arg.ProjectID, arg.ScanID)
	var i Scan
	err := row.Scan(
		&i.ScanID,
		&i.ProjectID,
		&i.ScanCompleted,
		&i.ServiceCount,
		&i.RegionCount,
		&i.ResourceCost,
		&i.Created,
	)
	return i, err
}

const getScanItemByScanItemId = `-- name: GetScanItemByScanItemId :one

SELECT scan_item_id, scan_id, service, region, resource_cost, findings, summary, remedy, created FROM scan_item WHERE scan_id = $1
`

// ------------------ Scan Item Queries --------------------
func (q *Queries) GetScanItemByScanItemId(ctx context.Context, scanID uuid.UUID) (ScanItem, error) {
	row := q.db.QueryRowContext(ctx, getScanItemByScanItemId, scanID)
	var i ScanItem
	err := row.Scan(
		&i.ScanItemID,
		&i.ScanID,
		&i.Service,
		&i.Region,
		&i.ResourceCost,
		pq.Array(&i.Findings),
		&i.Summary,
		&i.Remedy,
		&i.Created,
	)
	return i, err
}

const getScanItemEntriesByScanId = `-- name: GetScanItemEntriesByScanId :many
SELECT scan_item_entry_id, scan_item_id, findings, title, summary, remedy, commands, resource_cost, created FROM scan_item_entry WHERE scan_item_id = $1
`

func (q *Queries) GetScanItemEntriesByScanId(ctx context.Context, scanItemID int64) ([]ScanItemEntry, error) {
	rows, err := q.db.QueryContext(ctx, getScanItemEntriesByScanId, scanItemID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []ScanItemEntry
	for rows.Next() {
		var i ScanItemEntry
		if err := rows.Scan(
			&i.ScanItemEntryID,
			&i.ScanItemID,
			pq.Array(&i.Findings),
			&i.Title,
			&i.Summary,
			&i.Remedy,
			pq.Array(&i.Commands),
			&i.ResourceCost,
			&i.Created,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const getScanItemsByScanId = `-- name: GetScanItemsByScanId :many
SELECT scan_item_id, scan_id, service, region, resource_cost, findings, summary, remedy, created FROM scan_item WHERE scan_id = $1
`

func (q *Queries) GetScanItemsByScanId(ctx context.Context, scanID uuid.UUID) ([]ScanItem, error) {
	rows, err := q.db.QueryContext(ctx, getScanItemsByScanId, scanID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []ScanItem
	for rows.Next() {
		var i ScanItem
		if err := rows.Scan(
			&i.ScanItemID,
			&i.ScanID,
			&i.Service,
			&i.Region,
			&i.ResourceCost,
			pq.Array(&i.Findings),
			&i.Summary,
			&i.Remedy,
			&i.Created,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const getScansByProjectId = `-- name: GetScansByProjectId :many
SELECT scan_id, project_id, scan_completed, service_count, region_count, resource_cost, created FROM scan WHERE project_id = $1
`

func (q *Queries) GetScansByProjectId(ctx context.Context, projectID int64) ([]Scan, error) {
	rows, err := q.db.QueryContext(ctx, getScansByProjectId, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []Scan
	for rows.Next() {
		var i Scan
		if err := rows.Scan(
			&i.ScanID,
			&i.ProjectID,
			&i.ScanCompleted,
			&i.ServiceCount,
			&i.RegionCount,
			&i.ResourceCost,
			&i.Created,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const getSubscriptionById = `-- name: GetSubscriptionById :one
SELECT id, team_id, stripe_subscription_id, resources_included, resources_used, created FROM subscription_plan WHERE id = $1 LIMIT 1
`

func (q *Queries) GetSubscriptionById(ctx context.Context, id int64) (SubscriptionPlan, error) {
	row := q.db.QueryRowContext(ctx, getSubscriptionById, id)
	var i SubscriptionPlan
	err := row.Scan(
		&i.ID,
		&i.TeamID,
		&i.StripeSubscriptionID,
		&i.ResourcesIncluded,
		&i.ResourcesUsed,
		&i.Created,
	)
	return i, err
}

const getSubscriptionByStripeSubscriptionId = `-- name: GetSubscriptionByStripeSubscriptionId :one
SELECT id, team_id, stripe_subscription_id, resources_included, resources_used, created FROM subscription_plan WHERE stripe_subscription_id = $1 LIMIT 1
`

func (q *Queries) GetSubscriptionByStripeSubscriptionId(ctx context.Context, stripeSubscriptionID sql.NullString) (SubscriptionPlan, error) {
	row := q.db.QueryRowContext(ctx, getSubscriptionByStripeSubscriptionId, stripeSubscriptionID)
	var i SubscriptionPlan
	err := row.Scan(
		&i.ID,
		&i.TeamID,
		&i.StripeSubscriptionID,
		&i.ResourcesIncluded,
		&i.ResourcesUsed,
		&i.Created,
	)
	return i, err
}

const getSubscriptionByTeamId = `-- name: GetSubscriptionByTeamId :one
SELECT id, team_id, stripe_subscription_id, resources_included, resources_used, created FROM subscription_plan WHERE team_id = $1 ORDER BY created LIMIT 1
`

func (q *Queries) GetSubscriptionByTeamId(ctx context.Context, teamID int64) (SubscriptionPlan, error) {
	row := q.db.QueryRowContext(ctx, getSubscriptionByTeamId, teamID)
	var i SubscriptionPlan
	err := row.Scan(
		&i.ID,
		&i.TeamID,
		&i.StripeSubscriptionID,
		&i.ResourcesIncluded,
		&i.ResourcesUsed,
		&i.Created,
	)
	return i, err
}

const getSubscriptionByTeamIdSubscriptionId = `-- name: GetSubscriptionByTeamIdSubscriptionId :one
SELECT id, team_id, stripe_subscription_id, resources_included, resources_used, created FROM subscription_plan WHERE team_id = $1 AND id = $2 LIMIT 1
`

type GetSubscriptionByTeamIdSubscriptionIdParams struct {
	TeamID int64
	ID     int64
}

func (q *Queries) GetSubscriptionByTeamIdSubscriptionId(ctx context.Context, arg GetSubscriptionByTeamIdSubscriptionIdParams) (SubscriptionPlan, error) {
	row := q.db.QueryRowContext(ctx, getSubscriptionByTeamIdSubscriptionId, arg.TeamID, arg.ID)
	var i SubscriptionPlan
	err := row.Scan(
		&i.ID,
		&i.TeamID,
		&i.StripeSubscriptionID,
		&i.ResourcesIncluded,
		&i.ResourcesUsed,
		&i.Created,
	)
	return i, err
}

const getTeamByStripeCustomerId = `-- name: GetTeamByStripeCustomerId :one
SELECT team_id, team_slug, team_name, stripe_customer_id, created FROM team WHERE stripe_customer_id = $1
`

func (q *Queries) GetTeamByStripeCustomerId(ctx context.Context, stripeCustomerID sql.NullString) (Team, error) {
	row := q.db.QueryRowContext(ctx, getTeamByStripeCustomerId, stripeCustomerID)
	var i Team
	err := row.Scan(
		&i.TeamID,
		&i.TeamSlug,
		&i.TeamName,
		&i.StripeCustomerID,
		&i.Created,
	)
	return i, err
}

const getTeamByTeamId = `-- name: GetTeamByTeamId :one
SELECT team_id, team_slug, team_name, stripe_customer_id, created FROM team WHERE team_id = $1 LIMIT 1
`

func (q *Queries) GetTeamByTeamId(ctx context.Context, teamID int64) (Team, error) {
	row := q.db.QueryRowContext(ctx, getTeamByTeamId, teamID)
	var i Team
	err := row.Scan(
		&i.TeamID,
		&i.TeamSlug,
		&i.TeamName,
		&i.StripeCustomerID,
		&i.Created,
	)
	return i, err
}

const getTeamByTeamSlug = `-- name: GetTeamByTeamSlug :one

SELECT team_id, team_slug, team_name, stripe_customer_id, created FROM team WHERE team_slug = $1 LIMIT 1
`

// ------------------ Team Queries --------------------
func (q *Queries) GetTeamByTeamSlug(ctx context.Context, teamSlug string) (Team, error) {
	row := q.db.QueryRowContext(ctx, getTeamByTeamSlug, teamSlug)
	var i Team
	err := row.Scan(
		&i.TeamID,
		&i.TeamSlug,
		&i.TeamName,
		&i.StripeCustomerID,
		&i.Created,
	)
	return i, err
}

const getTeamMembershipByTeamIdUserId = `-- name: GetTeamMembershipByTeamIdUserId :one
SELECT team_membership_id, team_id, user_id, membership_type, created FROM team_membership WHERE team_id = $1 AND user_id = $2 LIMIT 1
`

type GetTeamMembershipByTeamIdUserIdParams struct {
	TeamID int64
	UserID int64
}

func (q *Queries) GetTeamMembershipByTeamIdUserId(ctx context.Context, arg GetTeamMembershipByTeamIdUserIdParams) (TeamMembership, error) {
	row := q.db.QueryRowContext(ctx, getTeamMembershipByTeamIdUserId, arg.TeamID, arg.UserID)
	var i TeamMembership
	err := row.Scan(
		&i.TeamMembershipID,
		&i.TeamID,
		&i.UserID,
		&i.MembershipType,
		&i.Created,
	)
	return i, err
}

const getTeamMembershipsByTeamId = `-- name: GetTeamMembershipsByTeamId :many
SELECT team_membership_id, team_id, user_id, membership_type, created FROM team_membership WHERE team_id = $1 ORDER BY created
`

func (q *Queries) GetTeamMembershipsByTeamId(ctx context.Context, teamID int64) ([]TeamMembership, error) {
	rows, err := q.db.QueryContext(ctx, getTeamMembershipsByTeamId, teamID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []TeamMembership
	for rows.Next() {
		var i TeamMembership
		if err := rows.Scan(
			&i.TeamMembershipID,
			&i.TeamID,
			&i.UserID,
			&i.MembershipType,
			&i.Created,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const getTeamsByUserId = `-- name: GetTeamsByUserId :many
SELECT team.team_id, team.team_slug, team.team_name, team.stripe_customer_id, team.created
FROM team
JOIN team_membership on team.team_id = team_membership.team_id
WHERE team_membership.user_id = $1
ORDER BY team.created
`

func (q *Queries) GetTeamsByUserId(ctx context.Context, userID int64) ([]Team, error) {
	rows, err := q.db.QueryContext(ctx, getTeamsByUserId, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []Team
	for rows.Next() {
		var i Team
		if err := rows.Scan(
			&i.TeamID,
			&i.TeamSlug,
			&i.TeamName,
			&i.StripeCustomerID,
			&i.Created,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const getUserByEmail = `-- name: GetUserByEmail :one
SELECT user_id, email, full_name, external_id, created FROM user_info WHERE email = $1 LIMIT 1
`

func (q *Queries) GetUserByEmail(ctx context.Context, email string) (UserInfo, error) {
	row := q.db.QueryRowContext(ctx, getUserByEmail, email)
	var i UserInfo
	err := row.Scan(
		&i.UserID,
		&i.Email,
		&i.FullName,
		&i.ExternalID,
		&i.Created,
	)
	return i, err
}

const getUserById = `-- name: GetUserById :one
SELECT user_id, email, full_name, external_id, created FROM user_info WHERE user_id = $1 LIMIT 1
`

func (q *Queries) GetUserById(ctx context.Context, userID int64) (UserInfo, error) {
	row := q.db.QueryRowContext(ctx, getUserById, userID)
	var i UserInfo
	err := row.Scan(
		&i.UserID,
		&i.Email,
		&i.FullName,
		&i.ExternalID,
		&i.Created,
	)
	return i, err
}

const incrementScanItemResourceCostByScanItemid = `-- name: IncrementScanItemResourceCostByScanItemid :one
UPDATE scan_item SET resource_cost = resource_cost + $1 WHERE scan_item_id = $2 RETURNING scan_item_id, scan_id, service, region, resource_cost, findings, summary, remedy, created
`

type IncrementScanItemResourceCostByScanItemidParams struct {
	ResourceCost int32
	ScanItemID   int64
}

func (q *Queries) IncrementScanItemResourceCostByScanItemid(ctx context.Context, arg IncrementScanItemResourceCostByScanItemidParams) (ScanItem, error) {
	row := q.db.QueryRowContext(ctx, incrementScanItemResourceCostByScanItemid, arg.ResourceCost, arg.ScanItemID)
	var i ScanItem
	err := row.Scan(
		&i.ScanItemID,
		&i.ScanID,
		&i.Service,
		&i.Region,
		&i.ResourceCost,
		pq.Array(&i.Findings),
		&i.Summary,
		&i.Remedy,
		&i.Created,
	)
	return i, err
}

const incrementScanResourceCostByScanId = `-- name: IncrementScanResourceCostByScanId :one
UPDATE scan SET resource_cost = resource_cost + $1 WHERE scan_id = $2 RETURNING scan_id, project_id, scan_completed, service_count, region_count, resource_cost, created
`

type IncrementScanResourceCostByScanIdParams struct {
	ResourceCost int32
	ScanID       uuid.UUID
}

func (q *Queries) IncrementScanResourceCostByScanId(ctx context.Context, arg IncrementScanResourceCostByScanIdParams) (Scan, error) {
	row := q.db.QueryRowContext(ctx, incrementScanResourceCostByScanId, arg.ResourceCost, arg.ScanID)
	var i Scan
	err := row.Scan(
		&i.ScanID,
		&i.ProjectID,
		&i.ScanCompleted,
		&i.ServiceCount,
		&i.RegionCount,
		&i.ResourceCost,
		&i.Created,
	)
	return i, err
}

const incrementSubscriptionResourcesUsedByTeamId = `-- name: IncrementSubscriptionResourcesUsedByTeamId :one

UPDATE subscription_plan SET resources_used = resources_used + $1 WHERE team_id = $2 RETURNING id, team_id, stripe_subscription_id, resources_included, resources_used, created
`

type IncrementSubscriptionResourcesUsedByTeamIdParams struct {
	ResourcesUsed int32
	TeamID        int64
}

// ------------------ Subscription Plan Queries --------------------
func (q *Queries) IncrementSubscriptionResourcesUsedByTeamId(ctx context.Context, arg IncrementSubscriptionResourcesUsedByTeamIdParams) (SubscriptionPlan, error) {
	row := q.db.QueryRowContext(ctx, incrementSubscriptionResourcesUsedByTeamId, arg.ResourcesUsed, arg.TeamID)
	var i SubscriptionPlan
	err := row.Scan(
		&i.ID,
		&i.TeamID,
		&i.StripeSubscriptionID,
		&i.ResourcesIncluded,
		&i.ResourcesUsed,
		&i.Created,
	)
	return i, err
}

const resetSubscriptionResourcesUsed = `-- name: ResetSubscriptionResourcesUsed :one
UPDATE subscription_plan 
SET resources_used = 0 
WHERE team_id = $1 
RETURNING id, team_id, stripe_subscription_id, resources_included, resources_used, created
`

func (q *Queries) ResetSubscriptionResourcesUsed(ctx context.Context, teamID int64) (SubscriptionPlan, error) {
	row := q.db.QueryRowContext(ctx, resetSubscriptionResourcesUsed, teamID)
	var i SubscriptionPlan
	err := row.Scan(
		&i.ID,
		&i.TeamID,
		&i.StripeSubscriptionID,
		&i.ResourcesIncluded,
		&i.ResourcesUsed,
		&i.Created,
	)
	return i, err
}

const setSubscriptionStripeIdByTeamId = `-- name: SetSubscriptionStripeIdByTeamId :one
UPDATE subscription_plan SET stripe_subscription_id = $2 WHERE team_id = $1 RETURNING id, team_id, stripe_subscription_id, resources_included, resources_used, created
`

type SetSubscriptionStripeIdByTeamIdParams struct {
	TeamID               int64
	StripeSubscriptionID sql.NullString
}

func (q *Queries) SetSubscriptionStripeIdByTeamId(ctx context.Context, arg SetSubscriptionStripeIdByTeamIdParams) (SubscriptionPlan, error) {
	row := q.db.QueryRowContext(ctx, setSubscriptionStripeIdByTeamId, arg.TeamID, arg.StripeSubscriptionID)
	var i SubscriptionPlan
	err := row.Scan(
		&i.ID,
		&i.TeamID,
		&i.StripeSubscriptionID,
		&i.ResourcesIncluded,
		&i.ResourcesUsed,
		&i.Created,
	)
	return i, err
}

const updateScanCompletedStatus = `-- name: UpdateScanCompletedStatus :one
UPDATE scan SET scan_completed = $1 WHERE scan_id = $2 RETURNING scan_id, project_id, scan_completed, service_count, region_count, resource_cost, created
`

type UpdateScanCompletedStatusParams struct {
	ScanCompleted bool
	ScanID        uuid.UUID
}

func (q *Queries) UpdateScanCompletedStatus(ctx context.Context, arg UpdateScanCompletedStatusParams) (Scan, error) {
	row := q.db.QueryRowContext(ctx, updateScanCompletedStatus, arg.ScanCompleted, arg.ScanID)
	var i Scan
	err := row.Scan(
		&i.ScanID,
		&i.ProjectID,
		&i.ScanCompleted,
		&i.ServiceCount,
		&i.RegionCount,
		&i.ResourceCost,
		&i.Created,
	)
	return i, err
}

const updateTeamStripeCustomerIdByTeamId = `-- name: UpdateTeamStripeCustomerIdByTeamId :one
UPDATE team SET stripe_customer_id = $2 WHERE team_id = $1 RETURNING team_id, team_slug, team_name, stripe_customer_id, created
`

type UpdateTeamStripeCustomerIdByTeamIdParams struct {
	TeamID           int64
	StripeCustomerID sql.NullString
}

func (q *Queries) UpdateTeamStripeCustomerIdByTeamId(ctx context.Context, arg UpdateTeamStripeCustomerIdByTeamIdParams) (Team, error) {
	row := q.db.QueryRowContext(ctx, updateTeamStripeCustomerIdByTeamId, arg.TeamID, arg.StripeCustomerID)
	var i Team
	err := row.Scan(
		&i.TeamID,
		&i.TeamSlug,
		&i.TeamName,
		&i.StripeCustomerID,
		&i.Created,
	)
	return i, err
}
