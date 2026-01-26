package hooks

import (
	"encoding/json"
	"path/filepath"
	"time"

	"github.com/example/microforge/internal/util"
)

type AgentHeartbeat struct {
	Timestamp    string `json:"timestamp"`
	Status       string `json:"status"`
	AssignmentID string `json:"assignment_id,omitempty"`
	TurnID       string `json:"turn_id,omitempty"`
	Message      string `json:"message,omitempty"`
}

func UpdateHeartbeat(identity AgentIdentity, status, assignmentID, turnID, message string) {
	if identity.RigHome == "" || identity.RigName == "" || identity.CellName == "" || identity.Role == "" {
		return
	}
	base := filepath.Join(identity.RigHome, "rigs", identity.RigName, "agents", identity.CellName, identity.Role)
	_ = util.EnsureDir(base)
	hb := AgentHeartbeat{
		Timestamp:    time.Now().UTC().Format(time.RFC3339),
		Status:       status,
		AssignmentID: assignmentID,
		TurnID:       turnID,
		Message:      message,
	}
	b, err := json.MarshalIndent(hb, "", "  ")
	if err != nil {
		return
	}
	_ = util.AtomicWriteFile(filepath.Join(base, "heartbeat.json"), b, 0o644)
}
