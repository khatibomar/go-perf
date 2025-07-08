// incident/response_team.go
package incident

import (
	"fmt"
	"sync"
	"time"
)

type ResponseTeam struct {
	responders map[string]*Responder
	escalationChain []EscalationLevel
	mu sync.RWMutex
}

type Responder struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Email        string    `json:"email"`
	Phone        string    `json:"phone"`
	Role         string    `json:"role"`
	Skills       []string  `json:"skills"`
	Available    bool      `json:"available"`
	CurrentLoad  int       `json:"current_load"`
	MaxLoad      int       `json:"max_load"`
	LastAssigned time.Time `json:"last_assigned"`
	Timezone     string    `json:"timezone"`
}

type EscalationLevel struct {
	Level       int           `json:"level"`
	Name        string        `json:"name"`
	Responders  []string      `json:"responders"`
	Timeout     time.Duration `json:"timeout"`
	Severities  []Severity    `json:"severities"`
	NotifyAll   bool          `json:"notify_all"`
}

func NewResponseTeam() *ResponseTeam {
	rt := &ResponseTeam{
		responders: make(map[string]*Responder),
	}

	// Initialize with default responders
	rt.initializeDefaultTeam()
	rt.setupEscalationChain()

	return rt
}

func (rt *ResponseTeam) initializeDefaultTeam() {
	defaultResponders := []*Responder{
		{
			ID:          "resp-001",
			Name:        "Alice Johnson",
			Email:       "alice@company.com",
			Phone:       "+1-555-0101",
			Role:        "SRE",
			Skills:      []string{"kubernetes", "monitoring", "databases"},
			Available:   true,
			CurrentLoad: 0,
			MaxLoad:     3,
			Timezone:    "UTC",
		},
		{
			ID:          "resp-002",
			Name:        "Bob Smith",
			Email:       "bob@company.com",
			Phone:       "+1-555-0102",
			Role:        "DevOps",
			Skills:      []string{"infrastructure", "networking", "security"},
			Available:   true,
			CurrentLoad: 0,
			MaxLoad:     2,
			Timezone:    "UTC",
		},
		{
			ID:          "resp-003",
			Name:        "Carol Davis",
			Email:       "carol@company.com",
			Phone:       "+1-555-0103",
			Role:        "Lead Engineer",
			Skills:      []string{"architecture", "performance", "debugging"},
			Available:   true,
			CurrentLoad: 0,
			MaxLoad:     2,
			Timezone:    "UTC",
		},
		{
			ID:          "resp-004",
			Name:        "David Wilson",
			Email:       "david@company.com",
			Phone:       "+1-555-0104",
			Role:        "Engineering Manager",
			Skills:      []string{"management", "escalation", "communication"},
			Available:   true,
			CurrentLoad: 0,
			MaxLoad:     1,
			Timezone:    "UTC",
		},
	}

	for _, responder := range defaultResponders {
		rt.responders[responder.ID] = responder
	}
}

func (rt *ResponseTeam) setupEscalationChain() {
	rt.escalationChain = []EscalationLevel{
		{
			Level:      1,
			Name:       "Primary Response",
			Responders: []string{"resp-001", "resp-002"},
			Timeout:    5 * time.Minute,
			Severities: []Severity{SeverityLow, SeverityMedium},
			NotifyAll:  false,
		},
		{
			Level:      2,
			Name:       "Senior Response",
			Responders: []string{"resp-003"},
			Timeout:    10 * time.Minute,
			Severities: []Severity{SeverityHigh},
			NotifyAll:  false,
		},
		{
			Level:      3,
			Name:       "Management Escalation",
			Responders: []string{"resp-004"},
			Timeout:    15 * time.Minute,
			Severities: []Severity{SeverityCritical},
			NotifyAll:  true,
		},
	}
}

func (rt *ResponseTeam) GetNextAvailableResponder(severity Severity) string {
	rt.mu.RLock()
	defer rt.mu.RUnlock()

	// Find appropriate escalation level for severity
	var targetLevel *EscalationLevel
	for _, level := range rt.escalationChain {
		for _, s := range level.Severities {
			if s == severity {
				targetLevel = &level
				break
			}
		}
		if targetLevel != nil {
			break
		}
	}

	if targetLevel == nil {
		// Default to level 1 if no specific level found
		targetLevel = &rt.escalationChain[0]
	}

	// Find best available responder
	var bestResponder *Responder
	for _, responderID := range targetLevel.Responders {
		responder := rt.responders[responderID]
		if responder != nil && responder.Available && responder.CurrentLoad < responder.MaxLoad {
			if bestResponder == nil || responder.CurrentLoad < bestResponder.CurrentLoad {
				bestResponder = responder
			}
		}
	}

	if bestResponder != nil {
		bestResponder.CurrentLoad++
		bestResponder.LastAssigned = time.Now()
		return bestResponder.ID
	}

	return ""
}

func (rt *ResponseTeam) NotifyIncident(incident *Incident) {
	// Find appropriate escalation level
	var targetLevel *EscalationLevel
	for _, level := range rt.escalationChain {
		for _, s := range level.Severities {
			if s == incident.Severity {
				targetLevel = &level
				break
			}
		}
		if targetLevel != nil {
			break
		}
	}

	if targetLevel == nil {
		targetLevel = &rt.escalationChain[0]
	}

	// Notify responders
	if targetLevel.NotifyAll {
		rt.notifyAllResponders(incident, targetLevel)
	} else {
		rt.notifyAssignedResponder(incident)
	}
}

func (rt *ResponseTeam) notifyAllResponders(incident *Incident, level *EscalationLevel) {
	for _, responderID := range level.Responders {
		responder := rt.responders[responderID]
		if responder != nil {
			rt.sendNotification(responder, incident, "broadcast")
		}
	}
}

func (rt *ResponseTeam) notifyAssignedResponder(incident *Incident) {
	if incident.Assignee != "" {
		responder := rt.responders[incident.Assignee]
		if responder != nil {
			rt.sendNotification(responder, incident, "assignment")
		}
	}
}

func (rt *ResponseTeam) sendNotification(responder *Responder, incident *Incident, notificationType string) {
	// Simulate notification sending
	message := fmt.Sprintf(
		"[%s] Incident %s: %s (Severity: %s)\nAssigned to: %s\nDescription: %s",
		notificationType,
		incident.ID[:8],
		incident.Title,
		incident.Severity,
		responder.Name,
		incident.Description,
	)

	// In a real implementation, this would send emails, SMS, Slack messages, etc.
	fmt.Printf("NOTIFICATION to %s (%s): %s\n", responder.Name, responder.Email, message)
}

func (rt *ResponseTeam) EscalateIncident(incident *Incident) {
	rt.mu.Lock()
	defer rt.mu.Unlock()

	// Find next escalation level
	currentLevel := rt.findCurrentEscalationLevel(incident.Severity)
	nextLevel := rt.findNextEscalationLevel(currentLevel)

	if nextLevel != nil {
		// Assign to next level
		newAssignee := rt.getNextLevelResponder(nextLevel)
		if newAssignee != "" {
			// Release current assignee
			if incident.Assignee != "" {
				if currentResponder := rt.responders[incident.Assignee]; currentResponder != nil {
					currentResponder.CurrentLoad--
				}
			}

			// Assign to new responder
			incident.Assignee = newAssignee
			if newResponder := rt.responders[newAssignee]; newResponder != nil {
				newResponder.CurrentLoad++
				newResponder.LastAssigned = time.Now()
			}

			// Notify new assignee
			rt.notifyAssignedResponder(incident)

			// If this is management level, notify all
			if nextLevel.NotifyAll {
				rt.notifyAllResponders(incident, nextLevel)
			}
		}
	}
}

func (rt *ResponseTeam) findCurrentEscalationLevel(severity Severity) *EscalationLevel {
	for _, level := range rt.escalationChain {
		for _, s := range level.Severities {
			if s == severity {
				return &level
			}
		}
	}
	return nil
}

func (rt *ResponseTeam) findNextEscalationLevel(currentLevel *EscalationLevel) *EscalationLevel {
	if currentLevel == nil {
		return &rt.escalationChain[0]
	}

	for i, level := range rt.escalationChain {
		if level.Level == currentLevel.Level && i+1 < len(rt.escalationChain) {
			return &rt.escalationChain[i+1]
		}
	}

	return nil
}

func (rt *ResponseTeam) getNextLevelResponder(level *EscalationLevel) string {
	for _, responderID := range level.Responders {
		responder := rt.responders[responderID]
		if responder != nil && responder.Available && responder.CurrentLoad < responder.MaxLoad {
			return responderID
		}
	}
	return ""
}

func (rt *ResponseTeam) ReleaseResponder(responderID string) {
	rt.mu.Lock()
	defer rt.mu.Unlock()

	responder := rt.responders[responderID]
	if responder != nil && responder.CurrentLoad > 0 {
		responder.CurrentLoad--
	}
}

func (rt *ResponseTeam) GetResponder(id string) *Responder {
	rt.mu.RLock()
	defer rt.mu.RUnlock()

	return rt.responders[id]
}

func (rt *ResponseTeam) ListResponders() []*Responder {
	rt.mu.RLock()
	defer rt.mu.RUnlock()

	responders := make([]*Responder, 0, len(rt.responders))
	for _, responder := range rt.responders {
		responders = append(responders, responder)
	}

	return responders
}

func (rt *ResponseTeam) UpdateResponderAvailability(id string, available bool) error {
	rt.mu.Lock()
	defer rt.mu.Unlock()

	responder := rt.responders[id]
	if responder == nil {
		return fmt.Errorf("responder not found: %s", id)
	}

	responder.Available = available
	return nil
}