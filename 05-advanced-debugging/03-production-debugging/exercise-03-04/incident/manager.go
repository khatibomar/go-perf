// incident/manager.go
package incident

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

type Manager struct {
	config    Config
	incidents map[string]*Incident
	mu        sync.RWMutex
	alertChan chan Alert
	responseTeam *ResponseTeam
	postMortems map[string]*PostMortem
}

type Config struct {
	DetectionInterval      time.Duration
	ResponseTimeout        time.Duration
	ResolutionTimeout      time.Duration
	MaxConcurrentIncidents int
	EnableAutoResponse     bool
	AlertChannels          []string
}

type Incident struct {
	ID          string                 `json:"id"`
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Severity    Severity               `json:"severity"`
	Status      Status                 `json:"status"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	ResolvedAt  *time.Time             `json:"resolved_at,omitempty"`
	Assignee    string                 `json:"assignee,omitempty"`
	Tags        []string               `json:"tags"`
	Metrics     map[string]interface{} `json:"metrics"`
	Timeline    []TimelineEvent        `json:"timeline"`
	RootCause   string                 `json:"root_cause,omitempty"`
	Resolution  string                 `json:"resolution,omitempty"`
}

type Severity string

const (
	SeverityLow      Severity = "low"
	SeverityMedium   Severity = "medium"
	SeverityHigh     Severity = "high"
	SeverityCritical Severity = "critical"
)

type Status string

const (
	StatusOpen       Status = "open"
	StatusInProgress Status = "in_progress"
	StatusResolved   Status = "resolved"
	StatusEscalated  Status = "escalated"
)

type TimelineEvent struct {
	Timestamp   time.Time `json:"timestamp"`
	Event       string    `json:"event"`
	Description string    `json:"description"`
	User        string    `json:"user,omitempty"`
}

type Alert struct {
	Type        string                 `json:"type"`
	Message     string                 `json:"message"`
	Severity    Severity               `json:"severity"`
	Timestamp   time.Time              `json:"timestamp"`
	Metrics     map[string]interface{} `json:"metrics"`
	Source      string                 `json:"source"`
}

type CreateRequest struct {
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Severity    Severity               `json:"severity"`
	Tags        []string               `json:"tags"`
	Metrics     map[string]interface{} `json:"metrics"`
}

type PostMortem struct {
	IncidentID    string                 `json:"incident_id"`
	Title         string                 `json:"title"`
	Summary       string                 `json:"summary"`
	RootCause     string                 `json:"root_cause"`
	Impact        string                 `json:"impact"`
	Resolution    string                 `json:"resolution"`
	LessonsLearned []string              `json:"lessons_learned"`
	ActionItems   []ActionItem           `json:"action_items"`
	Timeline      []TimelineEvent        `json:"timeline"`
	Metrics       map[string]interface{} `json:"metrics"`
	CreatedAt     time.Time              `json:"created_at"`
	CreatedBy     string                 `json:"created_by"`
}

type ActionItem struct {
	ID          string    `json:"id"`
	Description string    `json:"description"`
	Assignee    string    `json:"assignee"`
	DueDate     time.Time `json:"due_date"`
	Status      string    `json:"status"`
}

func NewManager(config Config) *Manager {
	return &Manager{
		config:      config,
		incidents:   make(map[string]*Incident),
		alertChan:   make(chan Alert, 100),
		responseTeam: NewResponseTeam(),
		postMortems: make(map[string]*PostMortem),
	}
}

func (m *Manager) Start(ctx context.Context) {
	go m.processAlerts(ctx)
	go m.monitorIncidents(ctx)
}

func (m *Manager) processAlerts(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case alert := <-m.alertChan:
			m.handleAlert(alert)
		}
	}
}

func (m *Manager) monitorIncidents(ctx context.Context) {
	ticker := time.NewTicker(m.config.DetectionInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.checkIncidentTimeouts()
		}
	}
}

func (m *Manager) HandleAlert(alert Alert) {
	select {
	case m.alertChan <- alert:
	default:
		// Alert channel is full, log error
		fmt.Printf("Alert channel full, dropping alert: %+v\n", alert)
	}
}

func (m *Manager) handleAlert(alert Alert) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if we should create a new incident or update existing one
	if existingIncident := m.findRelatedIncident(alert); existingIncident != nil {
		m.updateIncidentWithAlert(existingIncident, alert)
		return
	}

	// Create new incident
	incident := &Incident{
		ID:          uuid.New().String(),
		Title:       fmt.Sprintf("%s Alert", alert.Type),
		Description: alert.Message,
		Severity:    alert.Severity,
		Status:      StatusOpen,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Tags:        []string{alert.Type, alert.Source},
		Metrics:     alert.Metrics,
		Timeline: []TimelineEvent{
			{
				Timestamp:   time.Now(),
				Event:       "incident_created",
				Description: fmt.Sprintf("Incident created from %s alert", alert.Type),
			},
		},
	}

	m.incidents[incident.ID] = incident

	// Auto-assign if enabled
	if m.config.EnableAutoResponse {
		m.autoAssignIncident(incident)
	}

	// Notify response team
	m.responseTeam.NotifyIncident(incident)
}

func (m *Manager) findRelatedIncident(alert Alert) *Incident {
	// Simple correlation logic - find open incidents with same type
	for _, incident := range m.incidents {
		if incident.Status == StatusOpen || incident.Status == StatusInProgress {
			for _, tag := range incident.Tags {
				if tag == alert.Type {
					return incident
				}
			}
		}
	}
	return nil
}

func (m *Manager) updateIncidentWithAlert(incident *Incident, alert Alert) {
	incident.UpdatedAt = time.Now()
	incident.Timeline = append(incident.Timeline, TimelineEvent{
		Timestamp:   time.Now(),
		Event:       "alert_received",
		Description: fmt.Sprintf("Related %s alert received: %s", alert.Type, alert.Message),
	})

	// Update metrics
	for k, v := range alert.Metrics {
		incident.Metrics[k] = v
	}

	// Escalate if severity increased
	if m.shouldEscalate(incident.Severity, alert.Severity) {
		incident.Severity = alert.Severity
		m.escalateIncident(incident)
	}
}

func (m *Manager) shouldEscalate(currentSeverity, newSeverity Severity) bool {
	severityLevels := map[Severity]int{
		SeverityLow:      1,
		SeverityMedium:   2,
		SeverityHigh:     3,
		SeverityCritical: 4,
	}
	return severityLevels[newSeverity] > severityLevels[currentSeverity]
}

func (m *Manager) autoAssignIncident(incident *Incident) {
	assignee := m.responseTeam.GetNextAvailableResponder(incident.Severity)
	if assignee != "" {
		incident.Assignee = assignee
		incident.Status = StatusInProgress
		incident.Timeline = append(incident.Timeline, TimelineEvent{
			Timestamp:   time.Now(),
			Event:       "incident_assigned",
			Description: fmt.Sprintf("Auto-assigned to %s", assignee),
		})
	}
}

func (m *Manager) escalateIncident(incident *Incident) {
	incident.Status = StatusEscalated
	incident.Timeline = append(incident.Timeline, TimelineEvent{
		Timestamp:   time.Now(),
		Event:       "incident_escalated",
		Description: "Incident escalated due to increased severity",
	})
	m.responseTeam.EscalateIncident(incident)
}

func (m *Manager) checkIncidentTimeouts() {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	for _, incident := range m.incidents {
		if incident.Status == StatusOpen || incident.Status == StatusInProgress {
			// Check response timeout
			if incident.Status == StatusOpen && now.Sub(incident.CreatedAt) > m.config.ResponseTimeout {
				m.escalateIncident(incident)
			}

			// Check resolution timeout
			if now.Sub(incident.CreatedAt) > m.config.ResolutionTimeout {
				incident.Timeline = append(incident.Timeline, TimelineEvent{
					Timestamp:   time.Now(),
					Event:       "resolution_timeout",
					Description: "Incident exceeded resolution timeout",
				})
				m.escalateIncident(incident)
			}
		}
	}
}

func (m *Manager) CreateIncident(req CreateRequest) (*Incident, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.incidents) >= m.config.MaxConcurrentIncidents {
		return nil, fmt.Errorf("maximum concurrent incidents (%d) reached", m.config.MaxConcurrentIncidents)
	}

	incident := &Incident{
		ID:          uuid.New().String(),
		Title:       req.Title,
		Description: req.Description,
		Severity:    req.Severity,
		Status:      StatusOpen,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Tags:        req.Tags,
		Metrics:     req.Metrics,
		Timeline: []TimelineEvent{
			{
				Timestamp:   time.Now(),
				Event:       "incident_created",
				Description: "Incident manually created",
			},
		},
	}

	if incident.Metrics == nil {
		incident.Metrics = make(map[string]interface{})
	}

	m.incidents[incident.ID] = incident

	if m.config.EnableAutoResponse {
		m.autoAssignIncident(incident)
	}

	m.responseTeam.NotifyIncident(incident)

	return incident, nil
}

func (m *Manager) GetIncident(id string) (*Incident, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	incident, exists := m.incidents[id]
	if !exists {
		return nil, fmt.Errorf("incident not found: %s", id)
	}

	return incident, nil
}

func (m *Manager) ListIncidents() []*Incident {
	m.mu.RLock()
	defer m.mu.RUnlock()

	incidents := make([]*Incident, 0, len(m.incidents))
	for _, incident := range m.incidents {
		incidents = append(incidents, incident)
	}

	return incidents
}

func (m *Manager) ResolveIncident(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	incident, exists := m.incidents[id]
	if !exists {
		return fmt.Errorf("incident not found: %s", id)
	}

	if incident.Status == StatusResolved {
		return fmt.Errorf("incident already resolved: %s", id)
	}

	now := time.Now()
	incident.Status = StatusResolved
	incident.ResolvedAt = &now
	incident.UpdatedAt = now
	incident.Timeline = append(incident.Timeline, TimelineEvent{
		Timestamp:   now,
		Event:       "incident_resolved",
		Description: "Incident marked as resolved",
	})

	// Generate post-mortem for high/critical incidents
	if incident.Severity == SeverityHigh || incident.Severity == SeverityCritical {
		m.generatePostMortem(incident)
	}

	return nil
}

func (m *Manager) EscalateIncident(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	incident, exists := m.incidents[id]
	if !exists {
		return fmt.Errorf("incident not found: %s", id)
	}

	m.escalateIncident(incident)
	return nil
}

func (m *Manager) generatePostMortem(incident *Incident) {
	postMortem := &PostMortem{
		IncidentID:     incident.ID,
		Title:          fmt.Sprintf("Post-mortem: %s", incident.Title),
		Summary:        incident.Description,
		RootCause:      incident.RootCause,
		Resolution:     incident.Resolution,
		Timeline:       incident.Timeline,
		Metrics:        incident.Metrics,
		CreatedAt:      time.Now(),
		CreatedBy:      "system",
		LessonsLearned: []string{},
		ActionItems:    []ActionItem{},
	}

	// Add impact analysis
	if incident.ResolvedAt != nil {
		duration := incident.ResolvedAt.Sub(incident.CreatedAt)
		postMortem.Impact = fmt.Sprintf("Incident duration: %v", duration)
	}

	m.postMortems[incident.ID] = postMortem
}

func (m *Manager) GetPostMortem(incidentID string) (*PostMortem, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	postMortem, exists := m.postMortems[incidentID]
	if !exists {
		return nil, fmt.Errorf("post-mortem not found for incident: %s", incidentID)
	}

	return postMortem, nil
}