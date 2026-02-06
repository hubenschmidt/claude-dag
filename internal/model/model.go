package model

type TaskStatus string

const (
	StatusPending   TaskStatus = "pending"
	StatusRunning   TaskStatus = "running"
	StatusCompleted TaskStatus = "completed"
	StatusFailed    TaskStatus = "failed"
	StatusRejected  TaskStatus = "rejected"
)

type AgentRole string

const (
	RoleArchitect  AgentRole = "architect"
	RoleBackend    AgentRole = "backend"
	RoleFrontend   AgentRole = "frontend"
	RoleDatabase   AgentRole = "database"
	RoleMigrator   AgentRole = "migrator"
	RoleReviewer   AgentRole = "reviewer"
	RoleIntegrator AgentRole = "integrator"
)

const MaxAttempts = 3

type Task struct {
	ID           string     `json:"id"`
	Role         AgentRole  `json:"role"`
	Description  string     `json:"description"`
	Status       TaskStatus `json:"status"`
	DependsOn    []string   `json:"depends_on"`
	ArtifactDirs []string   `json:"artifact_dirs"`
	OutputDir    string     `json:"output_dir"`
	Result       string     `json:"result,omitempty"`
	Error        string     `json:"error,omitempty"`
	Feedback     string     `json:"feedback,omitempty"`
	Attempts     int        `json:"attempts"`
	ReviewTaskID string     `json:"review_task_id,omitempty"`
	PaneID       string     `json:"pane_id,omitempty"`
	StartedAt    int64      `json:"started_at,omitempty"`
}
