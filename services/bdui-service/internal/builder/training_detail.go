package builder

import (
	"fmt"
	"strings"

	"github.com/coach-link/platform/services/bdui-service/internal/model"
)

// BuildTrainingDetail собирает BDUI-схему для описания тренировки.
// Описание парсится как структурированный текст: секции разделены "\n\n".
func BuildTrainingDetail(assignment *model.AssignmentDetail) model.BduiSchema {
	var children []model.BduiComponent

	if assignment.Description != "" {
		sections := parseDescription(assignment.Description)
		for i, section := range sections {
			if i > 0 {
				children = append(children, Divider())
			}
			if section.title != "" {
				children = append(children, Text(section.title, "title"))
			}
			if section.body != "" {
				children = append(children, Text(section.body, "body"))
			}
		}
	} else {
		children = append(children, Text("Описание тренировки отсутствует", "body"))
	}

	return model.BduiSchema{
		ScreenID: fmt.Sprintf("training-detail/%s", assignment.ID),
		Version:  "1.0.0",
		Root:     ScrollView(16, children),
	}
}

type descriptionSection struct {
	title string
	body  string
}

func parseDescription(desc string) []descriptionSection {
	blocks := strings.Split(strings.TrimSpace(desc), "\n\n")
	sections := make([]descriptionSection, 0, len(blocks))

	for _, block := range blocks {
		block = strings.TrimSpace(block)
		if block == "" {
			continue
		}
		lines := strings.SplitN(block, "\n", 2)
		section := descriptionSection{title: strings.TrimSpace(lines[0])}
		if len(lines) > 1 {
			section.body = strings.TrimSpace(lines[1])
		}
		sections = append(sections, section)
	}

	return sections
}
