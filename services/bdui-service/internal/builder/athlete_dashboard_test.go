package builder

import (
	"testing"

	"github.com/coach-link/platform/services/bdui-service/internal/model"
)

// --- BuildAthleteDashboard ---

func TestBuildAthleteDashboard_Schema(t *testing.T) {
	schema := BuildAthleteDashboard(AthleteDashboardData{})

	if schema.ScreenID != "athlete-dashboard" {
		t.Errorf("ScreenID = %q, want athlete-dashboard", schema.ScreenID)
	}
	if schema.Version != "1.0.0" {
		t.Errorf("Version = %q, want 1.0.0", schema.Version)
	}
	if schema.Root.Type != "scroll_view" {
		t.Errorf("Root.Type = %q, want scroll_view", schema.Root.Type)
	}
}

func TestBuildAthleteDashboard_GreetingDefault(t *testing.T) {
	schema := BuildAthleteDashboard(AthleteDashboardData{})
	greeting := schema.Root.Children[0]
	if greeting.Props["text"] != "Привет!" {
		t.Errorf("greeting = %q, want %q", greeting.Props["text"], "Привет!")
	}
}

func TestBuildAthleteDashboard_GreetingWithProfile(t *testing.T) {
	data := AthleteDashboardData{
		Profile: &model.UserProfile{FullName: "Петров Алексей Сергеевич"},
	}
	schema := BuildAthleteDashboard(data)
	greeting := schema.Root.Children[0]
	want := "Привет, Алексей!"
	if greeting.Props["text"] != want {
		t.Errorf("greeting = %q, want %q", greeting.Props["text"], want)
	}
}

func TestBuildAthleteDashboard_WithCoach(t *testing.T) {
	data := AthleteDashboardData{
		Coach: &model.CoachInfo{FullName: "Сидоров Виктор"},
	}
	schema := BuildAthleteDashboard(data)
	// Children[2] = coach card
	coachCard := schema.Root.Children[2]
	if coachCard.ID != "my-coach" {
		t.Errorf("card ID = %q, want my-coach", coachCard.ID)
	}
	if coachCard.Props["subtitle"] != "Сидоров Виктор" {
		t.Errorf("subtitle = %q, want %q", coachCard.Props["subtitle"], "Сидоров Виктор")
	}
	if coachCard.Action == nil || coachCard.Action.Route != "/athlete/my-coach" {
		t.Errorf("unexpected action: %+v", coachCard.Action)
	}
}

func TestBuildAthleteDashboard_WithoutCoach(t *testing.T) {
	schema := BuildAthleteDashboard(AthleteDashboardData{})
	coachCard := schema.Root.Children[2]
	if coachCard.ID != "no-coach" {
		t.Errorf("card ID = %q, want no-coach", coachCard.ID)
	}
	if coachCard.Props["subtitle"] != "Не подключён" {
		t.Errorf("subtitle = %q, want %q", coachCard.Props["subtitle"], "Не подключён")
	}
	if coachCard.Action == nil || coachCard.Action.Route != "/athlete/find-coach" {
		t.Errorf("unexpected action: %+v", coachCard.Action)
	}
}

func TestBuildAthleteDashboard_UpcomingAssignments(t *testing.T) {
	data := AthleteDashboardData{
		UpcomingAsns: []model.AssignmentListItem{
			{ID: "a1", Title: "Круговая тренировка", ScheduledDate: "2026-04-21"},
			{ID: "a2", Title: "Бег 10км", ScheduledDate: "2026-04-22"},
		},
	}
	schema := BuildAthleteDashboard(data)
	// Children[4] = list upcoming-assignments
	list := schema.Root.Children[4]
	if list.ID != "upcoming-assignments" {
		t.Fatalf("expected upcoming-assignments list, got ID=%q", list.ID)
	}
	if len(list.Children) != 2 {
		t.Fatalf("expected 2 tiles, got %d", len(list.Children))
	}
	// "Круговая" → fitness_center icon
	if list.Children[0].Props["leading_icon"] != "fitness_center" {
		t.Errorf("icon = %q, want fitness_center", list.Children[0].Props["leading_icon"])
	}
	// Бег → directions_run
	if list.Children[1].Props["leading_icon"] != "directions_run" {
		t.Errorf("icon = %q, want directions_run", list.Children[1].Props["leading_icon"])
	}
	if list.Children[0].Action.Route != "/athlete/assignments/a1" {
		t.Errorf("route = %q, want /athlete/assignments/a1", list.Children[0].Action.Route)
	}
}

func TestBuildAthleteDashboard_AllAssignmentsButton(t *testing.T) {
	schema := BuildAthleteDashboard(AthleteDashboardData{})
	// Last child = button
	children := schema.Root.Children
	btn := children[len(children)-1]
	if btn.Type != "button" {
		t.Fatalf("last child type = %q, want button", btn.Type)
	}
	if btn.Action == nil || btn.Action.Route != "/athlete/assignments" {
		t.Errorf("button action: %+v", btn.Action)
	}
}

// --- extractFirstName ---

func TestExtractFirstName(t *testing.T) {
	tests := []struct {
		fullName string
		want     string
	}{
		{"Иванов Иван Иванович", "Иван"},
		{"Петров Алексей", "Алексей"},
		{"Одно", "Одно"},
		{"", ""},
	}
	for _, tc := range tests {
		t.Run(tc.fullName, func(t *testing.T) {
			got := extractFirstName(tc.fullName)
			if got != tc.want {
				t.Errorf("extractFirstName(%q) = %q, want %q", tc.fullName, got, tc.want)
			}
		})
	}
}
