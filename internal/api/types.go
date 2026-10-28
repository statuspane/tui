package api

import "time"

// Pointer fields are the ones the API documents as nullable.

type App struct {
	Name                   string  `json:"name"`
	Slug                   string  `json:"slug"`
	Status                 *string `json:"status"`
	WatchedComponentsCount int     `json:"watched_components_count"`
}

type AppDetail struct {
	App
	LastCheckedAt     *time.Time  `json:"last_checked_at"`
	WatchedComponents []Component `json:"watched_components"`
	ActiveIncidents   []Incident  `json:"active_incidents"`
}

type ComponentReference struct {
	Provider string `json:"provider"`
	ID       string `json:"id"`
	Name     string `json:"name"`
}

type Component struct {
	ComponentReference
	GroupName *string `json:"group_name"`
	Status    *string `json:"status"`
}

type Incident struct {
	ID                 string               `json:"id"`
	Provider           string               `json:"provider"`
	Title              string               `json:"title"`
	Impact             *string              `json:"impact"`
	Status             *string              `json:"status"`
	StartedAt          *time.Time           `json:"started_at"`
	URL                string               `json:"url"`
	LatestUpdate       *IncidentUpdate      `json:"latest_update"`
	AffectedComponents []ComponentReference `json:"affected_components"`
}

type IncidentUpdate struct {
	Status     *string    `json:"status"`
	Body       string     `json:"body"`
	OccurredAt *time.Time `json:"occurred_at"`
}

// AppHistory is the app's status per UTC day, oldest first. A day with a nil status has no data,
// which is not the same as operational. The API also sends each watched component's days, which
// this client does not read yet.
type AppHistory struct {
	TrackingSince *time.Time   `json:"tracking_since"`
	Days          []HistoryDay `json:"days"`
}

type HistoryDay struct {
	Date   string  `json:"date"`
	Status *string `json:"status"`
}
