// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.27.0

package postgres

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type MembershipType string

const (
	MembershipTypeOWNER  MembershipType = "OWNER"
	MembershipTypeADMIN  MembershipType = "ADMIN"
	MembershipTypeMEMBER MembershipType = "MEMBER"
)

func (e *MembershipType) Scan(src interface{}) error {
	switch s := src.(type) {
	case []byte:
		*e = MembershipType(s)
	case string:
		*e = MembershipType(s)
	default:
		return fmt.Errorf("unsupported scan type for MembershipType: %T", src)
	}
	return nil
}

type NullMembershipType struct {
	MembershipType MembershipType
	Valid          bool // Valid is true if MembershipType is not NULL
}

// Scan implements the Scanner interface.
func (ns *NullMembershipType) Scan(value interface{}) error {
	if value == nil {
		ns.MembershipType, ns.Valid = "", false
		return nil
	}
	ns.Valid = true
	return ns.MembershipType.Scan(value)
}

// Value implements the driver Valuer interface.
func (ns NullMembershipType) Value() (driver.Value, error) {
	if !ns.Valid {
		return nil, nil
	}
	return string(ns.MembershipType), nil
}

type AccountConnection struct {
	ConnectionID int64
	ProjectID    int64
	ExternalID   uuid.UUID
	AccountID    string
	Created      time.Time
}

type Project struct {
	ProjectID   int64
	TeamID      int64
	ProjectSlug string
	ProjectName string
	Created     time.Time
}

type Scan struct {
	ScanID        uuid.UUID
	ProjectID     int64
	ScanCompleted bool
	Regions       []string
	Services      []string
	ServiceCount  int32
	RegionCount   int32
	ResourceCost  int32
	Created       time.Time
}

type ScanItem struct {
	ScanItemID   int64
	ScanID       uuid.UUID
	Service      string
	Region       string
	ResourceCost int32
	Findings     []string
	Summary      string
	Remedy       string
	Created      time.Time
}

type ScanItemEntry struct {
	ScanItemEntryID int64
	ScanItemID      int64
	Findings        []string
	Title           string
	Summary         string
	Remedy          string
	Commands        []string
	ResourceCost    int32
	Created         time.Time
}

type SubscriptionPlan struct {
	ID                   int64
	TeamID               int64
	StripeSubscriptionID sql.NullString
	ResourcesIncluded    int32
	ResourcesUsed        int32
	Created              time.Time
}

type Team struct {
	TeamID           int64
	TeamSlug         string
	TeamName         string
	StripeCustomerID sql.NullString
	Created          time.Time
}

type TeamInvite struct {
	TeamInviteID int64
	InviteCode   string
	TeamID       int64
	InviteeEmail string
	Created      time.Time
}

type TeamMembership struct {
	TeamMembershipID int64
	TeamID           int64
	UserID           int64
	MembershipType   MembershipType
	Created          time.Time
}

type UserInfo struct {
	UserID     int64
	Email      string
	FullName   string
	ExternalID uuid.UUID
	Created    time.Time
}
