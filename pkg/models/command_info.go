package models

type CommandInfo struct {
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	Triggers     []string `json:"triggers"`
	Usages       []string `json:"usages"`
	RequiresAuth bool     `json:"requires_auth"`
	AllowDM      bool     `json:"allow_dm"`
}

type CommandUsage struct {
	Name  string `firestore:"name" json:"name"`
	Count int    `firestore:"count" json:"count"`
}
