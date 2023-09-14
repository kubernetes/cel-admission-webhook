package alertmanager

type AlertInfo struct {
	Name        string
	Severity    string
	Resource    string
	Instance    string
	Description string
	Namespace   string
}
