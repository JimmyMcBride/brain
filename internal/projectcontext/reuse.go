package projectcontext

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
)

type PacketCacheStatus string

const (
	PacketCacheStatusFresh  PacketCacheStatus = "fresh"
	PacketCacheStatusReused PacketCacheStatus = "reused"
	PacketCacheStatusDelta  PacketCacheStatus = "delta"
)

type PacketCacheMetadata struct {
	CacheStatus         PacketCacheStatus `json:"cache_status"`
	Fingerprint         string            `json:"fingerprint"`
	ReusedFrom          string            `json:"reused_from,omitempty"`
	DeltaFrom           string            `json:"delta_from,omitempty"`
	ChangedSections     []string          `json:"changed_sections,omitempty"`
	ChangedItemIDs      []string          `json:"changed_item_ids,omitempty"`
	InvalidationReasons []string          `json:"invalidation_reasons,omitempty"`
	FullPacketIncluded  bool              `json:"full_packet_included"`
	FallbackReason      string            `json:"fallback_reason,omitempty"`
}

type CompileResponse struct {
	PacketHash string       `json:"packet_hash"`
	Task       CompiledTask `json:"task"`
	Budget     PacketBudget `json:"budget"`
	PacketCacheMetadata
	BaseContract []CompiledItem      `json:"base_contract,omitempty"`
	WorkingSet   *CompiledWorkingSet `json:"working_set,omitempty"`
	Verification *[]VerificationHint `json:"verification,omitempty"`
	Ambiguities  *[]string           `json:"ambiguities,omitempty"`
	Provenance   *[]PacketProvenance `json:"provenance,omitempty"`
}

type PacketFingerprintInputs struct {
	TaskText                 string   `json:"task_text"`
	TaskSummary              string   `json:"task_summary"`
	TaskSource               string   `json:"task_source"`
	BudgetPreset             string   `json:"budget_preset,omitempty"`
	BudgetTarget             int      `json:"budget_target"`
	ChangedFiles             []string `json:"changed_files,omitempty"`
	TouchedBoundaries        []string `json:"touched_boundaries,omitempty"`
	DurableSearchSignals     []string `json:"durable_search_signals,omitempty"`
	SourceSummaryHashes      []string `json:"source_summary_hashes,omitempty"`
	VerificationRequirements []string `json:"verification_requirements,omitempty"`
}

func NewCompileResponse(packet *CompiledPacket, meta PacketCacheMetadata) *CompileResponse {
	if packet == nil {
		return nil
	}
	response := &CompileResponse{
		PacketHash:          packet.Hash(),
		Task:                packet.Task,
		Budget:              packet.Budget,
		PacketCacheMetadata: normalizePacketCacheMetadata(meta),
	}
	if response.FullPacketIncluded {
		response.BaseContract = append([]CompiledItem(nil), packet.BaseContract...)
		workingSet := packet.WorkingSet
		response.WorkingSet = &workingSet
		verification := append([]VerificationHint{}, packet.Verification...)
		response.Verification = &verification
		ambiguities := append([]string{}, packet.Ambiguities...)
		response.Ambiguities = &ambiguities
		provenance := append([]PacketProvenance{}, packet.Provenance...)
		response.Provenance = &provenance
	}
	return response
}

func (r *CompileResponse) ToCompiledPacket() *CompiledPacket {
	if r == nil || !r.FullPacketIncluded || r.WorkingSet == nil {
		return nil
	}
	return &CompiledPacket{
		Task:         r.Task,
		Budget:       r.Budget,
		BaseContract: append([]CompiledItem(nil), r.BaseContract...),
		WorkingSet:   *r.WorkingSet,
		Verification: cloneVerificationHints(r.Verification),
		Ambiguities:  cloneStrings(r.Ambiguities),
		Provenance:   cloneProvenance(r.Provenance),
	}
}

func (p PacketFingerprintInputs) Hash() string {
	body, err := json.Marshal(p.normalized())
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}

func (p PacketFingerprintInputs) InvalidationReasons(previous PacketFingerprintInputs) []string {
	current := p.normalized()
	prior := previous.normalized()
	reasons := []string{}
	if current.TaskText != prior.TaskText {
		reasons = append(reasons, "task text changed")
	}
	if current.TaskSummary != prior.TaskSummary {
		reasons = append(reasons, "task summary changed")
	}
	if current.TaskSource != prior.TaskSource {
		reasons = append(reasons, "task source changed")
	}
	if current.BudgetPreset != prior.BudgetPreset || current.BudgetTarget != prior.BudgetTarget {
		reasons = append(reasons, "compile budget changed")
	}
	if !equalStrings(current.ChangedFiles, prior.ChangedFiles) {
		reasons = append(reasons, "changed files changed")
	}
	if !equalStrings(current.TouchedBoundaries, prior.TouchedBoundaries) {
		reasons = append(reasons, "touched boundaries changed")
	}
	if !equalStrings(current.DurableSearchSignals, prior.DurableSearchSignals) {
		reasons = append(reasons, "durable search context changed")
	}
	if !equalStrings(current.SourceSummaryHashes, prior.SourceSummaryHashes) {
		reasons = append(reasons, "source summary state changed")
	}
	if !equalStrings(current.VerificationRequirements, prior.VerificationRequirements) {
		reasons = append(reasons, "verification requirements changed")
	}
	return reasons
}

func (p PacketFingerprintInputs) normalized() PacketFingerprintInputs {
	clone := p
	clone.ChangedFiles = sortedStrings(clone.ChangedFiles)
	clone.TouchedBoundaries = sortedStrings(clone.TouchedBoundaries)
	clone.DurableSearchSignals = sortedStrings(clone.DurableSearchSignals)
	clone.SourceSummaryHashes = sortedStrings(clone.SourceSummaryHashes)
	clone.VerificationRequirements = sortedStrings(clone.VerificationRequirements)
	return clone
}

func normalizePacketCacheMetadata(meta PacketCacheMetadata) PacketCacheMetadata {
	meta.ChangedSections = sortedStrings(meta.ChangedSections)
	meta.ChangedItemIDs = sortedStrings(meta.ChangedItemIDs)
	meta.InvalidationReasons = sortedStrings(meta.InvalidationReasons)
	return meta
}

func sortedStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	out := append([]string(nil), values...)
	sort.Strings(out)
	return out
}

func equalStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

func cloneVerificationHints(items *[]VerificationHint) []VerificationHint {
	if items == nil {
		return nil
	}
	return append([]VerificationHint(nil), (*items)...)
}

func cloneStrings(items *[]string) []string {
	if items == nil {
		return nil
	}
	return append([]string(nil), (*items)...)
}

func cloneProvenance(items *[]PacketProvenance) []PacketProvenance {
	if items == nil {
		return nil
	}
	return append([]PacketProvenance(nil), (*items)...)
}
