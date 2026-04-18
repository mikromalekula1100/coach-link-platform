package builder

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/coach-link/platform/services/bdui-service/internal/model"
)

type CoachDashboardData struct {
	Profile      *model.UserProfile
	Athletes     []model.AthleteInfo
	PendingReqs  []model.ConnectionRequest
	RecentRpts   []model.ReportWithPlan
	UpcomingAsns []model.AssignmentListItem
}

func BuildCoachDashboard(data CoachDashboardData) model.BduiSchema {
	var children []model.BduiComponent

	// 1. Приветствие
	greeting := "Добро пожаловать!"
	if data.Profile != nil && data.Profile.FullName != "" {
		greeting = fmt.Sprintf("Добрый день, %s!", data.Profile.FullName)
	}
	children = append(children, Text(greeting, "headline"))
	children = append(children, Spacer(16))

	// 2. Карточки-счётчики
	athleteCount := len(data.Athletes)
	pendingCount := len(data.PendingReqs)

	statsCards := []model.BduiComponent{
		Card(
			"athletes-count",
			"Спортсмены",
			fmt.Sprintf("%d %s", athleteCount, pluralAthletes(athleteCount)),
			"people",
			"#1976D2",
			Navigate("/coach/athletes"),
		),
	}

	if pendingCount > 0 {
		statsCards = append(statsCards, CardWithBadge(
			"pending-requests",
			"Заявки",
			"Ожидают решения",
			"person_add",
			"#FF9800",
			pendingCount,
			Navigate("/coach/requests"),
		))
	} else {
		statsCards = append(statsCards, Card(
			"pending-requests",
			"Заявки",
			"Нет новых",
			"person_add",
			"#4CAF50",
			Navigate("/coach/requests"),
		))
	}

	children = append(children, Row(12, statsCards))
	children = append(children, Spacer(24))

	// 3. Последние отчёты
	reportTiles := make([]model.BduiComponent, 0, len(data.RecentRpts))
	for _, r := range data.RecentRpts {
		subtitle := formatReportSubtitle(r)
		trailingText := formatRelativeTime(r.CreatedAt)
		reportTiles = append(reportTiles, ListTile(
			r.Title,
			subtitle,
			"description",
			trailingText,
			Navigate(fmt.Sprintf("/coach/assignments/%s/report", r.AssignmentID)),
		))
	}
	children = append(children, List(
		"recent-reports",
		"Последние отчёты",
		"Отчётов пока нет",
		true,
		reportTiles,
	))
	children = append(children, Spacer(24))

	// 4. Ближайшие задания
	assignmentTiles := make([]model.BduiComponent, 0, len(data.UpcomingAsns))
	for _, a := range data.UpcomingAsns {
		icon := "directions_run"
		if strings.Contains(strings.ToLower(a.Title), "офп") || strings.Contains(strings.ToLower(a.Title), "круговая") {
			icon = "fitness_center"
		}
		assignmentTiles = append(assignmentTiles, ListTile(
			a.Title,
			a.ScheduledDate,
			icon,
			"",
			Navigate(fmt.Sprintf("/coach/assignments/%s", a.ID)),
		))
	}
	children = append(children, List(
		"upcoming-assignments",
		"Ближайшие задания",
		"Нет запланированных заданий",
		true,
		assignmentTiles,
	))
	children = append(children, Spacer(24))

	// 5. Совет дня
	children = append(children, Container(
		16, "#FFF3E0", 12,
		[]model.BduiComponent{
			Text("💡 Не забывайте про восстановительные тренировки между интенсивными блоками!", "tip"),
		},
	))

	return model.BduiSchema{
		ScreenID:   "coach-dashboard",
		Version:    "1.0.0",
		TTLSeconds: 300,
		Root:       ScrollView(16, children),
	}
}

func formatReportSubtitle(r model.ReportWithPlan) string {
	var parts []string
	if r.PerceivedEffort > 0 {
		parts = append(parts, fmt.Sprintf("RPE: %d/10", r.PerceivedEffort))
	}
	if r.DurationMinutes > 0 {
		parts = append(parts, fmt.Sprintf("%d мин", r.DurationMinutes))
	}
	if r.DistanceKm != nil && *r.DistanceKm > 0 {
		parts = append(parts, fmt.Sprintf("%.1f км", *r.DistanceKm))
	}
	if len(parts) == 0 {
		return "Нет деталей"
	}
	return strings.Join(parts, " · ")
}

func formatRelativeTime(t time.Time) string {
	diff := time.Since(t)
	hours := diff.Hours()

	switch {
	case hours < 1:
		return "Только что"
	case hours < 24:
		return fmt.Sprintf("%d ч. назад", int(hours))
	case hours < 48:
		return "Вчера"
	default:
		days := int(math.Round(hours / 24))
		if days < 7 {
			return fmt.Sprintf("%d %s назад", days, pluralDays(days))
		}
		return t.Format("02.01.2006")
	}
}

func pluralAthletes(n int) string {
	mod := n % 10
	mod100 := n % 100
	if mod100 >= 11 && mod100 <= 14 {
		return "человек"
	}
	if mod == 1 {
		return "человек"
	}
	if mod >= 2 && mod <= 4 {
		return "человека"
	}
	return "человек"
}

func pluralDays(n int) string {
	mod := n % 10
	mod100 := n % 100
	if mod100 >= 11 && mod100 <= 14 {
		return "дней"
	}
	if mod == 1 {
		return "день"
	}
	if mod >= 2 && mod <= 4 {
		return "дня"
	}
	return "дней"
}
