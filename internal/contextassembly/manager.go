package contextassembly

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"brain/internal/projectcontext"
)

type Manager struct {
	Context *projectcontext.Manager
}

type Request struct {
	ProjectDir string
	Task       string
	TaskSource string
	Limit      int
	Explain    bool
}

type Packet struct {
	Task          TaskInfo     `json:"task"`
	Summary       Summary      `json:"summary"`
	Selected      GroupedItems `json:"selected"`
	Ambiguities   []string     `json:"ambiguities"`
	OmittedNearby GroupedItems `json:"omitted_nearby"`
}

type TaskInfo struct {
	Text   string `json:"text"`
	Source string `json:"source"`
}

type Summary struct {
	Confidence    string      `json:"confidence"`
	SelectedCount int         `json:"selected_count"`
	GroupCounts   GroupCounts `json:"group_counts"`
}

type GroupCounts struct {
	DurableNotes     int `json:"durable_notes"`
	GeneratedContext int `json:"generated_context"`
	StructuralRepo   int `json:"structural_repo"`
	LiveWork         int `json:"live_work"`
	PolicyWorkflow   int `json:"policy_workflow"`
}

type GroupedItems struct {
	DurableNotes     []Item `json:"durable_notes"`
	GeneratedContext []Item `json:"generated_context"`
	StructuralRepo   []Item `json:"structural_repo"`
	LiveWork         []Item `json:"live_work"`
	PolicyWorkflow   []Item `json:"policy_workflow"`
}

type Item struct {
	Source          string           `json:"source"`
	Label           string           `json:"label"`
	Kind            string           `json:"kind"`
	Excerpt         string           `json:"excerpt"`
	Why             string           `json:"why"`
	Rank            int              `json:"rank,omitempty"`
	SelectionMethod string           `json:"selection_method,omitempty"`
	Diagnostics     *ItemDiagnostics `json:"diagnostics,omitempty"`
}

type ItemDiagnostics struct {
	SourceGroup string   `json:"source_group"`
	Notes       []string `json:"notes,omitempty"`
}

func New(contextManager *projectcontext.Manager) *Manager {
	return &Manager{Context: contextManager}
}

func (m *Manager) Assemble(req Request) (*Packet, error) {
	task := strings.TrimSpace(req.Task)
	if task == "" {
		return nil, errors.New("task context assembly requires a task")
	}
	taskSource := strings.TrimSpace(req.TaskSource)
	if taskSource == "" {
		taskSource = "flag"
	}
	packet := &Packet{
		Task: TaskInfo{
			Text:   task,
			Source: taskSource,
		},
		Summary: Summary{
			Confidence:    "low",
			SelectedCount: 0,
			GroupCounts:   GroupCounts{},
		},
		Selected:      newGroupedItems(),
		Ambiguities:   []string{},
		OmittedNearby: newGroupedItems(),
	}
	return packet, nil
}

func RenderHuman(w io.Writer, packet *Packet, explain bool) error {
	if packet == nil {
		return errors.New("task context packet is required")
	}
	groupCount := countNonEmptyGroups(packet.Selected)
	if _, err := fmt.Fprintf(w, "## Task Context\n\n- Task: `%s`\n- Source: `%s`\n- Confidence: `%s`\n- Selected: %d item(s) across %d group(s)\n\n",
		packet.Task.Text,
		packet.Task.Source,
		packet.Summary.Confidence,
		packet.Summary.SelectedCount,
		groupCount,
	); err != nil {
		return err
	}
	if _, err := io.WriteString(w, "## Selected Context\n\n"); err != nil {
		return err
	}
	if packet.Summary.SelectedCount == 0 {
		if _, err := io.WriteString(w, "- No selected context yet.\n"); err != nil {
			return err
		}
	} else {
		writeGroup := func(label string, items []Item) error {
			if len(items) == 0 {
				return nil
			}
			if _, err := fmt.Fprintf(w, "### %s\n\n", label); err != nil {
				return err
			}
			for _, item := range items {
				if _, err := fmt.Fprintf(w, "- `%s`: %s\n", item.Source, item.Why); err != nil {
					return err
				}
			}
			if _, err := io.WriteString(w, "\n"); err != nil {
				return err
			}
			return nil
		}
		for _, entry := range []struct {
			label string
			items []Item
		}{
			{label: "Durable Notes", items: packet.Selected.DurableNotes},
			{label: "Generated Context", items: packet.Selected.GeneratedContext},
			{label: "Structural Repo", items: packet.Selected.StructuralRepo},
			{label: "Live Work", items: packet.Selected.LiveWork},
			{label: "Policy Workflow", items: packet.Selected.PolicyWorkflow},
		} {
			if err := writeGroup(entry.label, entry.items); err != nil {
				return err
			}
		}
	}
	if len(packet.Ambiguities) > 0 {
		if _, err := io.WriteString(w, "\n## Ambiguities\n\n"); err != nil {
			return err
		}
		for _, ambiguity := range packet.Ambiguities {
			if _, err := fmt.Fprintf(w, "- %s\n", ambiguity); err != nil {
				return err
			}
		}
	}
	_ = explain
	return nil
}

func newGroupedItems() GroupedItems {
	return GroupedItems{
		DurableNotes:     []Item{},
		GeneratedContext: []Item{},
		StructuralRepo:   []Item{},
		LiveWork:         []Item{},
		PolicyWorkflow:   []Item{},
	}
}

func countNonEmptyGroups(groups GroupedItems) int {
	count := 0
	for _, items := range [][]Item{
		groups.DurableNotes,
		groups.GeneratedContext,
		groups.StructuralRepo,
		groups.LiveWork,
		groups.PolicyWorkflow,
	} {
		if len(items) > 0 {
			count++
		}
	}
	return count
}
