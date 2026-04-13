package contextassembly

import (
	"bytes"
	"strings"
	"testing"
)

func TestAssembleReturnsStableEmptyPacketShape(t *testing.T) {
	manager := New(nil)

	packet, err := manager.Assemble(Request{
		Task:       "tighten auth flow",
		TaskSource: "flag",
	})
	if err != nil {
		t.Fatal(err)
	}

	if packet.Task.Text != "tighten auth flow" || packet.Task.Source != "flag" {
		t.Fatalf("unexpected task payload: %#v", packet.Task)
	}
	if packet.Summary.Confidence != "low" || packet.Summary.SelectedCount != 0 {
		t.Fatalf("unexpected summary: %#v", packet.Summary)
	}
	if packet.Selected.DurableNotes == nil || packet.Selected.GeneratedContext == nil || packet.Selected.StructuralRepo == nil || packet.Selected.LiveWork == nil || packet.Selected.PolicyWorkflow == nil {
		t.Fatalf("expected empty selected groups to be initialized: %#v", packet.Selected)
	}
	if packet.OmittedNearby.DurableNotes == nil || packet.OmittedNearby.GeneratedContext == nil || packet.OmittedNearby.StructuralRepo == nil || packet.OmittedNearby.LiveWork == nil || packet.OmittedNearby.PolicyWorkflow == nil {
		t.Fatalf("expected empty omitted groups to be initialized: %#v", packet.OmittedNearby)
	}
}

func TestAssembleRequiresTask(t *testing.T) {
	manager := New(nil)
	if _, err := manager.Assemble(Request{}); err == nil {
		t.Fatal("expected task requirement error")
	}
}

func TestRenderHumanCompactPacket(t *testing.T) {
	packet := &Packet{
		Task: TaskInfo{Text: "tighten auth flow", Source: "flag"},
		Summary: Summary{
			Confidence:    "low",
			SelectedCount: 0,
		},
		Selected:      newGroupedItems(),
		Ambiguities:   []string{},
		OmittedNearby: newGroupedItems(),
	}

	var out bytes.Buffer
	if err := RenderHuman(&out, packet, false); err != nil {
		t.Fatal(err)
	}
	rendered := out.String()
	if !strings.Contains(rendered, "## Task Context") || !strings.Contains(rendered, "## Selected Context") {
		t.Fatalf("expected compact sections in human output:\n%s", rendered)
	}
	if strings.Contains(rendered, "## Why This Was Selected") {
		t.Fatalf("did not expect explain sections in compact output:\n%s", rendered)
	}
}
