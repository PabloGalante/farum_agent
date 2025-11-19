package domain

import "time"

type SessionID string
type UserID string
type MessageID string

type Role string

const (
	RoleUser  Role = "user"
	RoleAgent Role = "agent"
)

type InteractionMode string

const (
	ModeCheckIn    InteractionMode = "check_in"    // Short conversation, Emotional Status
	ModeDeepDive   InteractionMode = "deep_dive"   // Deeper Exploration
	ModeActionPlan InteractionMode = "action_plan" // Goal-Oriented
)

type Timestamp = time.Time
