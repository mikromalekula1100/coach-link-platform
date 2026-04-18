package model

// BduiSchema — корневая модель BDUI-экрана.
type BduiSchema struct {
	ScreenID   string        `json:"screen_id"`
	Version    string        `json:"version"`
	TTLSeconds int           `json:"ttl_seconds,omitempty"`
	Root       BduiComponent `json:"root"`
}

// BduiComponent — узел дерева UI-компонентов.
type BduiComponent struct {
	Type     string                 `json:"type"`
	ID       string                 `json:"id,omitempty"`
	Props    map[string]interface{} `json:"props,omitempty"`
	Children []BduiComponent        `json:"children,omitempty"`
	Action   *BduiAction            `json:"action,omitempty"`
}

// BduiAction — действие при взаимодействии с компонентом.
type BduiAction struct {
	Type   string                 `json:"type"`
	Route  string                 `json:"route,omitempty"`
	Method string                 `json:"method,omitempty"`
	URL    string                 `json:"url,omitempty"`
	Body   map[string]interface{} `json:"body,omitempty"`
}
