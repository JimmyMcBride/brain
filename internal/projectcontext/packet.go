package projectcontext

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

type CompiledTask struct {
	Text    string `json:"text"`
	Summary string `json:"summary"`
	Source  string `json:"source"`
}

type CompiledItem struct {
	ContextItem
	Reason string `json:"reason"`
}

type CompiledBoundary struct {
	Path               string   `json:"path"`
	Label              string   `json:"label"`
	Role               string   `json:"role"`
	Reason             string   `json:"reason"`
	AdjacentBoundaries []string `json:"adjacent_boundaries,omitempty"`
	Responsibilities   []string `json:"responsibilities,omitempty"`
}

type CompiledFile struct {
	Path   string `json:"path"`
	Status string `json:"status"`
	Source string `json:"source"`
	Reason string `json:"reason"`
}

type CompiledTest struct {
	Path     string `json:"path"`
	Relation string `json:"relation"`
	Reason   string `json:"reason"`
}

type VerificationHint struct {
	ID       string `json:"id"`
	Label    string `json:"label"`
	Command  string `json:"command,omitempty"`
	Summary  string `json:"summary"`
	Source   string `json:"source"`
	Strength string `json:"strength,omitempty"`
	Reason   string `json:"reason"`
}

type CompiledWorkingSet struct {
	Boundaries []CompiledBoundary `json:"boundaries"`
	Files      []CompiledFile     `json:"files"`
	Tests      []CompiledTest     `json:"tests"`
	Notes      []CompiledItem     `json:"notes"`
}

type PacketProvenance struct {
	ItemID  string        `json:"item_id"`
	Section string        `json:"section"`
	Anchor  ContextAnchor `json:"anchor"`
	Reason  string        `json:"reason"`
}

type CompiledPacket struct {
	Task         CompiledTask       `json:"task"`
	BaseContract []CompiledItem     `json:"base_contract"`
	WorkingSet   CompiledWorkingSet `json:"working_set"`
	Verification []VerificationHint `json:"verification"`
	Ambiguities  []string           `json:"ambiguities"`
	Provenance   []PacketProvenance `json:"provenance"`
}

func (p *CompiledPacket) Hash() string {
	if p == nil {
		return ""
	}
	body, err := json.Marshal(p)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}
