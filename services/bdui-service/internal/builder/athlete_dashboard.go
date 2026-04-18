package builder

import (
	"fmt"
	"strings"

	"github.com/coach-link/platform/services/bdui-service/internal/model"
)

type AthleteDashboardData struct {
	Profile      *model.UserProfile
	Coach        *model.CoachInfo
	UpcomingAsns []model.AssignmentListItem
}

func BuildAthleteDashboard(data AthleteDashboardData) model.BduiSchema {
	var children []model.BduiComponent

	// 1. Приветствие
	greeting := "Привет!"
	if data.Profile != nil && data.Profile.FullName != "" {
		greeting = fmt.Sprintf("Привет, %s!", extractFirstName(data.Profile.FullName))
	}
	children = append(children, Text(greeting, "headline"))
	children = append(children, Spacer(16))

	// 2. Карточка тренера
	if data.Coach != nil {
		children = append(children, Card(
			"my-coach",
			"Мой тренер",
			data.Coach.FullName,
			"person",
			"#1976D2",
			Navigate("/athlete/my-coach"),
		))
	} else {
		children = append(children, Card(
			"no-coach",
			"Тренер",
			"Не подключён",
			"person_add",
			"#9E9E9E",
			Navigate("/athlete/find-coach"),
		))
	}
	children = append(children, Spacer(24))

	// 3. Ближайшие задания
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
			Navigate(fmt.Sprintf("/athlete/assignments/%s", a.ID)),
		))
	}
	children = append(children, List(
		"upcoming-assignments",
		"Ближайшие задания",
		"Нет запланированных заданий",
		true,
		assignmentTiles,
	))
	children = append(children, Spacer(16))

	// 4. Кнопка «Все задания»
	children = append(children, Button(
		"Все задания",
		"outlined",
		"arrow_forward",
		true,
		Navigate("/athlete/assignments"),
	))

	return model.BduiSchema{
		ScreenID:   "athlete-dashboard",
		Version:    "1.0.0",
		TTLSeconds: 300,
		Root:       ScrollView(16, children),
	}
}

// extractFirstName берёт имя из ФИО формата "Фамилия Имя [Отчество]".
func extractFirstName(fullName string) string {
	parts := strings.Fields(fullName)
	if len(parts) >= 2 {
		return parts[1]
	}
	return fullName
}
