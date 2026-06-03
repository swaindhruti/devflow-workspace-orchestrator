package projects

// project structure to hold project details
type Project struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Path        string   `json:"path"`
	TechStack   []string `json:"tech_stack"`
	Favorite    bool     `json:"favorite"`

	RunCommands []Command `json:"commands"`
}

// command structure to hold command details
type Command struct {
	ID          string `json:"id"`
	ProjectID   string `json:"project_id"`
	Name        string `json:"name"`
	Command     string `json:"command"`
	Path        string `json:"path"`
	Description string `json:"description"`
}
