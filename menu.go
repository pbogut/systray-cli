package main

import "github.com/godbus/dbus/v5"

type MenuProperties struct {
	Type     string
	Enabled  bool
	Visible  bool
	Label    string
	HasLabel bool
}

type MenuItem struct {
	ID         int32
	Properties MenuProperties
	Children   []MenuItem
}

type rawMenuItem struct {
	ID         int32
	Properties map[string]dbus.Variant
	Children   []rawMenuItem
}

type GetLayoutResponse struct {
	Version   int32
	RootProps map[string]dbus.Variant
	Items     []MenuItem
}

type rawGetLayoutResponse struct {
	Version   int32
	RootProps map[string]dbus.Variant
	Items     []rawMenuItem
}

func convertLayout(raw rawGetLayoutResponse) GetLayoutResponse {
	items := make([]MenuItem, len(raw.Items))
	for i, item := range raw.Items {
		items[i] = convertMenuItem(item)
	}

	return GetLayoutResponse{
		Version:   raw.Version,
		RootProps: raw.RootProps,
		Items:     items,
	}
}

func convertMenuItem(raw rawMenuItem) MenuItem {
	children := make([]MenuItem, len(raw.Children))
	for i, child := range raw.Children {
		children[i] = convertMenuItem(child)
	}

	return MenuItem{
		ID:         raw.ID,
		Properties: newMenuProperties(raw.Properties),
		Children:   children,
	}
}

func newMenuProperties(props map[string]dbus.Variant) MenuProperties {
	r := MenuProperties{
		Enabled: true,
		Visible: true,
	}

	if props == nil {
		return r
	}

	if v, ok := props["type"]; ok {
		if value, ok := v.Value().(string); ok {
			r.Type = value
		}
	}

	if v, ok := props["enabled"]; ok {
		if value, ok := v.Value().(bool); ok {
			r.Enabled = value
		}
	}

	if v, ok := props["visible"]; ok {
		if value, ok := v.Value().(bool); ok {
			r.Visible = value
		}
	}

	if v, ok := props["label"]; ok {
		if value, ok := v.Value().(string); ok {
			r.Label = value
			r.HasLabel = true
		}
	}

	return r
}
