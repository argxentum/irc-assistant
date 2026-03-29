package models

type CommandInfo struct {
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	Triggers     []string `json:"triggers"`
	RequiresAuth bool     `json:"requires_auth"`
}

type CommandUsage struct {
	Name  string `firestore:"name" json:"name"`
	Count int    `firestore:"count" json:"count"`
}
