package model

import (
	"bytes"
	"fmt"
	"text/template"
)

func (m *Model) LoadTemplate(system, user string) {
	systpl, err := template.ParseFiles(system)
	if err != nil {
		panic(err)
	}
	m.systemTpl = systpl

	usertpl, err := template.ParseFiles(user)
	if err != nil {
		panic(err)
	}
	m.userTpl = usertpl
}

func (m *Model) ExecuteUserTemplate(message string, args map[string]any) (string, error) {
	var buf bytes.Buffer
	var argsMap = make(map[string]any)
	argsMap["Message"] = message
	if args != nil {
		for k, v := range args {
			argsMap[k] = v
		}
	}
	if err := m.userTpl.Execute(&buf, argsMap); err != nil {
		return "", fmt.Errorf("error executing template: %w", err)
	}
	return buf.String(), nil
}

func (m *Model) ExecuteSystemTemplate(args map[string]any) (string, error) {
	var buf bytes.Buffer
	var argsMap = make(map[string]any)
	if args != nil {
		for k, v := range args {
			argsMap[k] = v
		}
	}
	if err := m.systemTpl.Execute(&buf, argsMap); err != nil {
		return "", fmt.Errorf("error executing template: %w", err)
	}
	return buf.String(), nil
}
