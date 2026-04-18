package builder

import "github.com/coach-link/platform/services/bdui-service/internal/model"

func Text(text, style string) model.BduiComponent {
	return model.BduiComponent{
		Type: "text",
		Props: map[string]interface{}{
			"text":  text,
			"style": style,
		},
	}
}

func Spacer(height float64) model.BduiComponent {
	return model.BduiComponent{
		Type:  "spacer",
		Props: map[string]interface{}{"height": height},
	}
}

func Divider() model.BduiComponent {
	return model.BduiComponent{Type: "divider"}
}

func Card(id, title, subtitle, icon, color string, action *model.BduiAction) model.BduiComponent {
	return model.BduiComponent{
		Type: "card",
		ID:   id,
		Props: map[string]interface{}{
			"title":    title,
			"subtitle": subtitle,
			"icon":     icon,
			"color":    color,
		},
		Action: action,
	}
}

func CardWithBadge(id, title, subtitle, icon, color string, badge int, action *model.BduiAction) model.BduiComponent {
	return model.BduiComponent{
		Type: "card",
		ID:   id,
		Props: map[string]interface{}{
			"title":    title,
			"subtitle": subtitle,
			"icon":     icon,
			"color":    color,
			"badge":    badge,
		},
		Action: action,
	}
}

func ListTile(title, subtitle, leadingIcon, trailingText string, action *model.BduiAction) model.BduiComponent {
	props := map[string]interface{}{
		"title":        title,
		"subtitle":     subtitle,
		"leading_icon": leadingIcon,
	}
	if trailingText != "" {
		props["trailing_text"] = trailingText
	}
	return model.BduiComponent{
		Type:   "list_tile",
		Props:  props,
		Action: action,
	}
}

func List(id, title, emptyText string, divider bool, children []model.BduiComponent) model.BduiComponent {
	return model.BduiComponent{
		Type: "list",
		ID:   id,
		Props: map[string]interface{}{
			"title":      title,
			"empty_text": emptyText,
			"divider":    divider,
		},
		Children: children,
	}
}

func Row(spacing float64, children []model.BduiComponent) model.BduiComponent {
	return model.BduiComponent{
		Type:     "row",
		Props:    map[string]interface{}{"spacing": spacing},
		Children: children,
	}
}

func ScrollView(padding float64, children []model.BduiComponent) model.BduiComponent {
	return model.BduiComponent{
		Type:     "scroll_view",
		Props:    map[string]interface{}{"padding": padding},
		Children: children,
	}
}

func Container(padding float64, color string, borderRadius float64, children []model.BduiComponent) model.BduiComponent {
	return model.BduiComponent{
		Type: "container",
		Props: map[string]interface{}{
			"padding":       padding,
			"color":         color,
			"border_radius": borderRadius,
		},
		Children: children,
	}
}

func Button(text, style, icon string, fullWidth bool, action *model.BduiAction) model.BduiComponent {
	props := map[string]interface{}{
		"text":       text,
		"style":      style,
		"full_width": fullWidth,
	}
	if icon != "" {
		props["icon"] = icon
	}
	return model.BduiComponent{
		Type:   "button",
		Props:  props,
		Action: action,
	}
}

func Navigate(route string) *model.BduiAction {
	return &model.BduiAction{
		Type:  "navigate",
		Route: route,
	}
}

func Refresh() *model.BduiAction {
	return &model.BduiAction{Type: "refresh"}
}
