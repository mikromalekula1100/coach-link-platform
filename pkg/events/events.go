package events

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// NATS subject constants
const (
	SubjectUserRegistered    = "coachlink.user.registered"
	SubjectConnectionRequested = "coachlink.connection.requested"
	SubjectConnectionAccepted  = "coachlink.connection.accepted"
	SubjectConnectionRejected  = "coachlink.connection.rejected"
	SubjectTrainingAssigned    = "coachlink.training.assigned"
	SubjectTrainingDeleted     = "coachlink.training.deleted"
	SubjectReportSubmitted     = "coachlink.report.submitted"
	SubjectGroupAthleteAdded   = "coachlink.group.athlete_added"
	SubjectGroupAthleteRemoved = "coachlink.group.athlete_removed"

	StreamName = "COACHLINK"
)

type Event struct {
	EventID   string      `json:"event_id"`
	EventType string      `json:"event_type"`
	Timestamp time.Time   `json:"timestamp"`
	Payload   interface{} `json:"payload"`
}

func NewEvent(eventType string, payload interface{}) Event {
	return Event{
		EventID:   uuid.New().String(),
		EventType: eventType,
		Timestamp: time.Now().UTC(),
		Payload:   payload,
	}
}

func (e Event) Marshal() ([]byte, error) {
	return json.Marshal(e)
}

// Payload types

type UserRegisteredPayload struct {
	UserID   string `json:"user_id"`
	Login    string `json:"login"`
	FullName string `json:"full_name"`
	Email    string `json:"email"`
	Role     string `json:"role"`
}

type ConnectionRequestedPayload struct {
	RequestID       string `json:"request_id"`
	AthleteID       string `json:"athlete_id"`
	AthleteFullName string `json:"athlete_full_name"`
	AthleteLogin    string `json:"athlete_login"`
	CoachID         string `json:"coach_id"`
}

type ConnectionAcceptedPayload struct {
	RequestID       string `json:"request_id"`
	AthleteID       string `json:"athlete_id"`
	AthleteFullName string `json:"athlete_full_name"`
	CoachID         string `json:"coach_id"`
	CoachFullName   string `json:"coach_full_name"`
}

type ConnectionRejectedPayload struct {
	RequestID       string `json:"request_id"`
	AthleteID       string `json:"athlete_id"`
	CoachID         string `json:"coach_id"`
	CoachFullName   string `json:"coach_full_name"`
}

type TrainingAssignedPayload struct {
	AssignmentID string `json:"assignment_id"`
	AthleteID    string `json:"athlete_id"`
	CoachID      string `json:"coach_id"`
	CoachFullName string `json:"coach_full_name"`
	Title        string `json:"title"`
	ScheduledDate string `json:"scheduled_date"`
}

type TrainingDeletedPayload struct {
	AssignmentID string `json:"assignment_id"`
	AthleteID    string `json:"athlete_id"`
	CoachID      string `json:"coach_id"`
	Title        string `json:"title"`
}

type ReportSubmittedPayload struct {
	AssignmentID    string `json:"assignment_id"`
	AthleteID       string `json:"athlete_id"`
	AthleteFullName string `json:"athlete_full_name"`
	CoachID         string `json:"coach_id"`
	Title           string `json:"title"`
}

type GroupAthleteAddedPayload struct {
	GroupID         string `json:"group_id"`
	GroupName       string `json:"group_name"`
	AthleteID       string `json:"athlete_id"`
	CoachID         string `json:"coach_id"`
}

type GroupAthleteRemovedPayload struct {
	GroupID         string `json:"group_id"`
	GroupName       string `json:"group_name"`
	AthleteID       string `json:"athlete_id"`
	CoachID         string `json:"coach_id"`
}
