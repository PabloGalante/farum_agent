package domain

// Message represents a any message in a timeline (user or agent)
type Message struct {
	ID        MessageID
	SessionID SessionID
	Author    Role
	Text      string
	CreatedAt Timestamp

	// Metadata holds additional information about the message
	Tags        []string
	Mode        InteractionMode
	ReplyTo     *MessageID
	ContentType string // e.g., "text", "reflection", "task_list"
}

// Session represent a concrete "relationship" between a user and the agent (could last days)
type Session struct {
	ID        SessionID
	UserID    UserID
	CreatedAt Timestamp
	UpdatedAt Timestamp

	// Basic session's config
	PreferredMode InteractionMode
	Title         string
}
