package builder

import (
	"fmt"
	"testing"

	"github.com/coach-link/platform/services/bdui-service/internal/model"
)

// --- BuildTrainingDetail ---

func TestBuildTrainingDetail_ScreenID(t *testing.T) {
	assignment := &model.AssignmentDetail{ID: "abc-123"}
	schema := BuildTrainingDetail(assignment)

	want := "training-detail/abc-123"
	if schema.ScreenID != want {
		t.Errorf("ScreenID = %q, want %q", schema.ScreenID, want)
	}
	if schema.Version != "1.0.0" {
		t.Errorf("Version = %q, want 1.0.0", schema.Version)
	}
	if schema.Root.Type != "scroll_view" {
		t.Errorf("Root.Type = %q, want scroll_view", schema.Root.Type)
	}
}

func TestBuildTrainingDetail_EmptyDescription(t *testing.T) {
	assignment := &model.AssignmentDetail{ID: "x", Description: ""}
	schema := BuildTrainingDetail(assignment)

	children := schema.Root.Children
	if len(children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(children))
	}
	if children[0].Type != "text" {
		t.Errorf("child type = %q, want text", children[0].Type)
	}
	if children[0].Props["text"] != "Описание тренировки отсутствует" {
		t.Errorf("text = %q", children[0].Props["text"])
	}
}

func TestBuildTrainingDetail_SingleSection(t *testing.T) {
	assignment := &model.AssignmentDetail{
		ID:          "x",
		Description: "Разминка\n5 минут лёгкого бега",
	}
	schema := BuildTrainingDetail(assignment)
	children := schema.Root.Children

	// title text + body text = 2 children, no divider (only 1 section)
	if len(children) != 2 {
		t.Fatalf("expected 2 children, got %d", len(children))
	}
	if children[0].Props["text"] != "Разминка" || children[0].Props["style"] != "title" {
		t.Errorf("title component unexpected: %+v", children[0].Props)
	}
	if children[1].Props["text"] != "5 минут лёгкого бега" || children[1].Props["style"] != "body" {
		t.Errorf("body component unexpected: %+v", children[1].Props)
	}
}

func TestBuildTrainingDetail_MultipleSections(t *testing.T) {
	assignment := &model.AssignmentDetail{
		ID:          "x",
		Description: "Разминка\n5 мин бега\n\nОсновная часть\n3x10 приседаний",
	}
	schema := BuildTrainingDetail(assignment)
	children := schema.Root.Children

	// Section1: title + body
	// Divider
	// Section2: title + body
	// Total: 5 children
	if len(children) != 5 {
		t.Fatalf("expected 5 children, got %d\n%+v", len(children), children)
	}
	if children[2].Type != "divider" {
		t.Errorf("children[2].Type = %q, want divider", children[2].Type)
	}
	if children[3].Props["text"] != "Основная часть" {
		t.Errorf("section2 title = %q", children[3].Props["text"])
	}
}

func TestBuildTrainingDetail_TitleOnlySection(t *testing.T) {
	assignment := &model.AssignmentDetail{
		ID:          "x",
		Description: "Только заголовок",
	}
	schema := BuildTrainingDetail(assignment)
	children := schema.Root.Children

	// 1 section: title only, no body
	if len(children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(children))
	}
	if children[0].Props["style"] != "title" {
		t.Errorf("style = %q, want title", children[0].Props["style"])
	}
}

// --- parseDescription ---

func TestParseDescription(t *testing.T) {
	tests := []struct {
		desc     string
		sections []descriptionSection
	}{
		{
			desc:     "",
			sections: []descriptionSection{},
		},
		{
			desc:     "   \n\n   ",
			sections: []descriptionSection{},
		},
		{
			desc: "A\nB",
			sections: []descriptionSection{{title: "A", body: "B"}},
		},
		{
			desc: "A\nB\n\nC\nD",
			sections: []descriptionSection{
				{title: "A", body: "B"},
				{title: "C", body: "D"},
			},
		},
		{
			desc: "Только заголовок",
			sections: []descriptionSection{{title: "Только заголовок", body: ""}},
		},
	}
	for i, tc := range tests {
		t.Run(fmt.Sprintf("case_%d", i), func(t *testing.T) {
			got := parseDescription(tc.desc)
			if len(got) != len(tc.sections) {
				t.Fatalf("len(sections) = %d, want %d", len(got), len(tc.sections))
			}
			for j, s := range tc.sections {
				if got[j].title != s.title || got[j].body != s.body {
					t.Errorf("section[%d] = %+v, want %+v", j, got[j], s)
				}
			}
		})
	}
}
