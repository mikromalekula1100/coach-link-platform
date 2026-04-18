package builder

import (
	"fmt"
	"testing"
	"time"

	"github.com/coach-link/platform/services/bdui-service/internal/model"
)

// --- BuildCoachDashboard ---

func TestBuildCoachDashboard_Schema(t *testing.T) {
	schema := BuildCoachDashboard(CoachDashboardData{})

	if schema.ScreenID != "coach-dashboard" {
		t.Errorf("ScreenID = %q, want %q", schema.ScreenID, "coach-dashboard")
	}
	if schema.Version != "1.0.0" {
		t.Errorf("Version = %q, want %q", schema.Version, "1.0.0")
	}
	if schema.TTLSeconds != 300 {
		t.Errorf("TTLSeconds = %d, want 300", schema.TTLSeconds)
	}
	if schema.Root.Type != "scroll_view" {
		t.Errorf("Root.Type = %q, want %q", schema.Root.Type, "scroll_view")
	}
}

func TestBuildCoachDashboard_GreetingDefault(t *testing.T) {
	schema := BuildCoachDashboard(CoachDashboardData{})
	greeting := schema.Root.Children[0]
	if greeting.Type != "text" {
		t.Fatalf("Children[0].Type = %q, want text", greeting.Type)
	}
	if greeting.Props["text"] != "Добро пожаловать!" {
		t.Errorf("greeting = %q, want %q", greeting.Props["text"], "Добро пожаловать!")
	}
}

func TestBuildCoachDashboard_GreetingWithProfile(t *testing.T) {
	data := CoachDashboardData{
		Profile: &model.UserProfile{FullName: "Иванов Иван"},
	}
	schema := BuildCoachDashboard(data)
	greeting := schema.Root.Children[0]
	want := "Добрый день, Иванов Иван!"
	if greeting.Props["text"] != want {
		t.Errorf("greeting = %q, want %q", greeting.Props["text"], want)
	}
}

func TestBuildCoachDashboard_NoPendingRequests(t *testing.T) {
	data := CoachDashboardData{Athletes: []model.AthleteInfo{{}, {}}}
	schema := BuildCoachDashboard(data)
	// Children[2] = row, Children[2].Children[1] = pending-requests card
	row := schema.Root.Children[2]
	pendingCard := row.Children[1]
	if pendingCard.ID != "pending-requests" {
		t.Fatalf("expected pending-requests card, got ID=%q", pendingCard.ID)
	}
	if pendingCard.Props["subtitle"] != "Нет новых" {
		t.Errorf("subtitle = %q, want %q", pendingCard.Props["subtitle"], "Нет новых")
	}
	if _, hasBadge := pendingCard.Props["badge"]; hasBadge {
		t.Error("should not have badge when no pending requests")
	}
}

func TestBuildCoachDashboard_WithPendingRequests(t *testing.T) {
	data := CoachDashboardData{
		PendingReqs: []model.ConnectionRequest{{}, {}, {}},
	}
	schema := BuildCoachDashboard(data)
	row := schema.Root.Children[2]
	pendingCard := row.Children[1]
	if pendingCard.Props["badge"] != 3 {
		t.Errorf("badge = %v, want 3", pendingCard.Props["badge"])
	}
}

func TestBuildCoachDashboard_AthletesCard(t *testing.T) {
	data := CoachDashboardData{
		Athletes: []model.AthleteInfo{{}, {}, {}, {}, {}},
	}
	schema := BuildCoachDashboard(data)
	row := schema.Root.Children[2]
	athletesCard := row.Children[0]
	if athletesCard.ID != "athletes-count" {
		t.Fatalf("expected athletes-count card, got ID=%q", athletesCard.ID)
	}
	want := "5 человек"
	if athletesCard.Props["subtitle"] != want {
		t.Errorf("subtitle = %q, want %q", athletesCard.Props["subtitle"], want)
	}
}

func TestBuildCoachDashboard_RecentReports(t *testing.T) {
	dist := 10.5
	data := CoachDashboardData{
		RecentRpts: []model.ReportWithPlan{
			{
				ID:              "r1",
				AssignmentID:    "a1",
				Title:           "Бег 5км",
				PerceivedEffort: 7,
				DurationMinutes: 30,
				DistanceKm:      &dist,
				CreatedAt:       time.Now().Add(-2 * time.Hour),
			},
		},
	}
	schema := BuildCoachDashboard(data)
	// Children[4] = list recent-reports
	reportsList := schema.Root.Children[4]
	if reportsList.ID != "recent-reports" {
		t.Fatalf("expected recent-reports list, got ID=%q", reportsList.ID)
	}
	if len(reportsList.Children) != 1 {
		t.Fatalf("expected 1 report tile, got %d", len(reportsList.Children))
	}
	tile := reportsList.Children[0]
	if tile.Props["title"] != "Бег 5км" {
		t.Errorf("tile title = %q, want %q", tile.Props["title"], "Бег 5км")
	}
	if tile.Action == nil || tile.Action.Route != "/coach/assignments/a1/report" {
		t.Errorf("unexpected action: %+v", tile.Action)
	}
}

func TestBuildCoachDashboard_UpcomingAssignments(t *testing.T) {
	data := CoachDashboardData{
		UpcomingAsns: []model.AssignmentListItem{
			{ID: "asn1", Title: "ОФП тренировка", ScheduledDate: "2026-04-20"},
		},
	}
	schema := BuildCoachDashboard(data)
	// Children[6] = list upcoming-assignments
	list := schema.Root.Children[6]
	if list.ID != "upcoming-assignments" {
		t.Fatalf("expected upcoming-assignments list, got ID=%q", list.ID)
	}
	tile := list.Children[0]
	if tile.Props["leading_icon"] != "fitness_center" {
		t.Errorf("icon = %q, want fitness_center for ОФП", tile.Props["leading_icon"])
	}
}

// --- formatReportSubtitle ---

func TestFormatReportSubtitle(t *testing.T) {
	dist := 5.2
	tests := []struct {
		name string
		r    model.ReportWithPlan
		want string
	}{
		{
			name: "all fields",
			r:    model.ReportWithPlan{PerceivedEffort: 8, DurationMinutes: 45, DistanceKm: &dist},
			want: "RPE: 8/10 · 45 мин · 5.2 км",
		},
		{
			name: "only RPE",
			r:    model.ReportWithPlan{PerceivedEffort: 5},
			want: "RPE: 5/10",
		},
		{
			name: "only duration",
			r:    model.ReportWithPlan{DurationMinutes: 60},
			want: "60 мин",
		},
		{
			name: "no fields",
			r:    model.ReportWithPlan{},
			want: "Нет деталей",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := formatReportSubtitle(tc.r)
			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

// --- formatRelativeTime ---

func TestFormatRelativeTime(t *testing.T) {
	tests := []struct {
		name   string
		offset time.Duration
		want   string
	}{
		{"30 min ago → Только что", -30 * time.Minute, "Только что"},
		{"5 hours ago → 5 ч. назад", -5 * time.Hour, "5 ч. назад"},
		{"25 hours ago → Вчера", -25 * time.Hour, "Вчера"},
		{"3 days ago → 3 дня назад", -72 * time.Hour, "3 дня назад"},
		{"10 days ago → formatted date", -240 * time.Hour, time.Now().Add(-240 * time.Hour).Format("02.01.2006")},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := formatRelativeTime(time.Now().Add(tc.offset))
			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

// --- pluralAthletes ---

func TestPluralAthletes(t *testing.T) {
	tests := []struct{ n int; want string }{
		{0, "человек"},
		{1, "человек"},
		{2, "человека"},
		{3, "человека"},
		{4, "человека"},
		{5, "человек"},
		{11, "человек"},
		{21, "человек"},
		{22, "человека"},
	}
	for _, tc := range tests {
		t.Run(fmt.Sprintf("%d", tc.n), func(t *testing.T) {
			got := pluralAthletes(tc.n)
			if got != tc.want {
				t.Errorf("pluralAthletes(%d) = %q, want %q", tc.n, got, tc.want)
			}
		})
	}
}

// --- pluralDays ---

func TestPluralDays(t *testing.T) {
	tests := []struct{ n int; want string }{
		{1, "день"},
		{2, "дня"},
		{5, "дней"},
		{11, "дней"},
		{21, "день"},
	}
	for _, tc := range tests {
		t.Run(fmt.Sprintf("%d", tc.n), func(t *testing.T) {
			got := pluralDays(tc.n)
			if got != tc.want {
				t.Errorf("pluralDays(%d) = %q, want %q", tc.n, got, tc.want)
			}
		})
	}
}
