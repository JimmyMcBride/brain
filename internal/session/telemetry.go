package session

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"brain/internal/projectcontext"
)

const (
	maxTelemetryAnalysisSessions = 64
	maxTelemetryAnalysisPackets  = 256
)

type PacketExplainRequest struct {
	ProjectDir string
	PacketHash string
	Last       bool
}

type PacketReference struct {
	PacketHash  string                      `json:"packet_hash"`
	CompiledAt  time.Time                   `json:"compiled_at"`
	TaskText    string                      `json:"task_text"`
	TaskSummary string                      `json:"task_summary"`
	TaskSource  string                      `json:"task_source"`
	Budget      projectcontext.PacketBudget `json:"budget"`
	projectcontext.PacketCacheMetadata
	SessionID     string `json:"session_id"`
	SessionStatus string `json:"session_status"`
}

type ItemUtilityDiagnostic struct {
	IncludeCount                int      `json:"include_count"`
	ExpandCount                 int      `json:"expand_count"`
	SuccessfulVerificationCount int      `json:"successful_verification_count"`
	FailedVerificationCount     int      `json:"failed_verification_count"`
	DurableUpdateCount          int      `json:"durable_update_count"`
	SuccessfulSessionCloseCount int      `json:"successful_session_close_count"`
	UnusedIncludeCount          int      `json:"unused_include_count"`
	UtilityScore                int      `json:"utility_score"`
	NoiseScore                  int      `json:"noise_score"`
	LikelyUtility               string   `json:"likely_utility"`
	Reasons                     []string `json:"reasons,omitempty"`
}

type ExplainedPacketItem struct {
	ItemID      string                       `json:"item_id"`
	Section     string                       `json:"section"`
	Anchor      projectcontext.ContextAnchor `json:"anchor"`
	Reason      string                       `json:"reason"`
	Expanded    int                          `json:"expanded"`
	Diagnostics ItemUtilityDiagnostic        `json:"diagnostics"`
}

type PacketExpandedItem struct {
	ItemID string                       `json:"item_id"`
	Anchor projectcontext.ContextAnchor `json:"anchor"`
	Count  int                          `json:"count"`
}

type PacketVerificationOutcome struct {
	Command   string    `json:"command"`
	Success   bool      `json:"success"`
	Timestamp time.Time `json:"timestamp"`
}

type PacketDurableUpdate struct {
	File      string    `json:"file"`
	Operation string    `json:"operation"`
	Timestamp time.Time `json:"timestamp"`
}

type PacketSessionClose struct {
	Status            string    `json:"status"`
	Success           bool      `json:"success"`
	Timestamp         time.Time `json:"timestamp"`
	MemorySatisfiedBy string    `json:"memory_satisfied_by,omitempty"`
	MissingCommands   []string  `json:"missing_commands,omitempty"`
}

type PacketDownstreamOutcomes struct {
	VerificationRuns []PacketVerificationOutcome `json:"verification_runs,omitempty"`
	DurableUpdates   []PacketDurableUpdate       `json:"durable_updates,omitempty"`
	SessionClose     *PacketSessionClose         `json:"session_close,omitempty"`
}

type PacketExplanation struct {
	Packet        PacketReference          `json:"packet"`
	IncludedItems []ExplainedPacketItem    `json:"included_items"`
	ExpandedLater []PacketExpandedItem     `json:"expanded_later,omitempty"`
	Downstream    PacketDownstreamOutcomes `json:"downstream"`
}

type UtilityItemStat struct {
	ItemID  string                       `json:"item_id"`
	Section string                       `json:"section"`
	Anchor  projectcontext.ContextAnchor `json:"anchor"`
	ItemUtilityDiagnostic
}

type VerificationLinkStat struct {
	Command      string `json:"command"`
	PacketCount  int    `json:"packet_count"`
	SuccessCount int    `json:"success_count"`
}

type FreshPacketPressure struct {
	FreshPacketsAnalyzed      int     `json:"fresh_packets_analyzed"`
	FreshPacketsUnderPressure int     `json:"fresh_packets_under_pressure"`
	MandatoryOverTarget       int     `json:"mandatory_over_target"`
	PressureRate              float64 `json:"pressure_rate"`
}

type OmittedDocStat struct {
	Path           string `json:"path"`
	OmittedPackets int    `json:"omitted_packets"`
}

type UtilitySnapshot struct {
	SessionsAnalyzed      int                    `json:"sessions_analyzed"`
	PacketsAnalyzed       int                    `json:"packets_analyzed"`
	ItemsAnalyzed         int                    `json:"items_analyzed"`
	Items                 []UtilityItemStat      `json:"items"`
	VerificationLinks     []VerificationLinkStat `json:"verification_links,omitempty"`
	FreshPacketPressure   FreshPacketPressure    `json:"fresh_packet_pressure"`
	FrequentlyOmittedDocs []OmittedDocStat       `json:"frequently_omitted_docs"`
}

type ContextStatsRequest struct {
	ProjectDir string
	Limit      int
}

type ContextStats struct {
	SessionsAnalyzed        int                    `json:"sessions_analyzed"`
	PacketsAnalyzed         int                    `json:"packets_analyzed"`
	ItemsAnalyzed           int                    `json:"items_analyzed"`
	TopSignal               []UtilityItemStat      `json:"top_signal,omitempty"`
	TopNoise                []UtilityItemStat      `json:"top_noise,omitempty"`
	FrequentlyExpanded      []UtilityItemStat      `json:"frequently_expanded,omitempty"`
	CommonVerificationLinks []VerificationLinkStat `json:"common_verification_links,omitempty"`
	FreshPacketPressure     FreshPacketPressure    `json:"fresh_packet_pressure"`
	FrequentlyOmittedDocs   []OmittedDocStat       `json:"frequently_omitted_docs"`
}

type ContextEffectivenessRequest struct {
	ProjectDir string
	Limit      int
}

type PacketUseSummary struct {
	SessionsWithPackets      int       `json:"sessions_with_packets"`
	PacketsAnalyzed          int       `json:"packets_analyzed"`
	SinglePacketSessions     int       `json:"single_packet_sessions"`
	MultiPacketSessions      int       `json:"multi_packet_sessions"`
	AveragePacketsPerSession float64   `json:"average_packets_per_session"`
	MaxPacketsInSession      int       `json:"max_packets_in_session"`
	FirstPacketAt            time.Time `json:"first_packet_at"`
	LastPacketAt             time.Time `json:"last_packet_at"`
}

type PacketCacheSummary struct {
	FreshPackets   int `json:"fresh_packets"`
	ReusedPackets  int `json:"reused_packets"`
	DeltaPackets   int `json:"delta_packets"`
	UnknownPackets int `json:"unknown_packets"`
	FullPackets    int `json:"full_packets"`
	CompactPackets int `json:"compact_packets"`
}

type PacketBudgetSummary struct {
	FullPacketsAnalyzed       int     `json:"full_packets_analyzed"`
	AverageFullTarget         int     `json:"average_full_target"`
	AverageFullUsed           int     `json:"average_full_used"`
	AverageFullRemaining      int     `json:"average_full_remaining"`
	AverageFullOmitted        float64 `json:"average_full_omitted"`
	FreshPacketsUnderPressure int     `json:"fresh_packets_under_pressure"`
	MandatoryOverTarget       int     `json:"mandatory_over_target"`
	PressureRate              float64 `json:"pressure_rate"`
}

type PacketOutcomeSummary struct {
	PacketsWithExpansions             int `json:"packets_with_expansions"`
	ExpansionEvents                   int `json:"expansion_events"`
	PacketsWithSuccessfulVerification int `json:"packets_with_successful_verification"`
	SuccessfulVerificationEvents      int `json:"successful_verification_events"`
	FailedVerificationEvents          int `json:"failed_verification_events"`
	PacketsWithDurableUpdates         int `json:"packets_with_durable_updates"`
	DurableUpdateEvents               int `json:"durable_update_events"`
	SuccessfulSessionCloses           int `json:"successful_session_closes"`
}

type ContextEffectiveness struct {
	SessionsAnalyzed      int                  `json:"sessions_analyzed"`
	PacketsAnalyzed       int                  `json:"packets_analyzed"`
	ItemsAnalyzed         int                  `json:"items_analyzed"`
	PacketUse             PacketUseSummary     `json:"packet_use"`
	Cache                 PacketCacheSummary   `json:"cache"`
	Budget                PacketBudgetSummary  `json:"budget"`
	Outcomes              PacketOutcomeSummary `json:"outcomes"`
	TopSignal             []UtilityItemStat    `json:"top_signal,omitempty"`
	TopNoise              []UtilityItemStat    `json:"top_noise,omitempty"`
	FrequentlyOmittedDocs []OmittedDocStat     `json:"frequently_omitted_docs,omitempty"`
	TelemetryGaps         []string             `json:"telemetry_gaps,omitempty"`
	Recommendations       []string             `json:"recommendations,omitempty"`
}

type packetIncludedItem struct {
	ItemID  string
	Section string
	Anchor  projectcontext.ContextAnchor
	Reason  string
}

type packetObservation struct {
	PacketReference
	IncludedItems []packetIncludedItem
	Expanded      map[string]int
	Verification  []PacketVerificationOutcome
	Durable       []PacketDurableUpdate
	SessionClose  *PacketSessionClose
}

type utilityAggregate struct {
	ItemID                      string
	Section                     string
	Anchor                      projectcontext.ContextAnchor
	IncludeCount                int
	ExpandCount                 int
	SuccessfulVerificationCount int
	FailedVerificationCount     int
	DurableUpdateCount          int
	SuccessfulSessionCloseCount int
	UnusedIncludeCount          int
	LastIncludedAt              time.Time
}

type verificationAggregate struct {
	Command      string
	PacketCount  int
	SuccessCount int
}

type telemetryAnalysis struct {
	sessionsAnalyzed int
	packets          []packetObservation
	items            []UtilityItemStat
	itemsByID        map[string]UtilityItemStat
	verification     []VerificationLinkStat
	freshPressure    FreshPacketPressure
	omittedDocs      []OmittedDocStat
}

func (m *Manager) ExplainPacket(req PacketExplainRequest) (*PacketExplanation, error) {
	projectDir, err := filepath.Abs(defaultProjectDir(req.ProjectDir))
	if err != nil {
		return nil, err
	}
	analysis, err := m.buildTelemetryAnalysis(projectDir)
	if err != nil {
		return nil, err
	}
	if len(analysis.packets) == 0 {
		return nil, errors.New("no compiled context packets recorded yet")
	}
	target := selectExplainedPacket(analysis.packets, strings.TrimSpace(req.PacketHash))
	if target == nil {
		return nil, fmt.Errorf("packet %q was not found in local telemetry", strings.TrimSpace(req.PacketHash))
	}
	explanation := &PacketExplanation{
		Packet:        target.PacketReference,
		IncludedItems: make([]ExplainedPacketItem, 0, len(target.IncludedItems)),
		ExpandedLater: expandedItemsForPacket(*target),
		Downstream: PacketDownstreamOutcomes{
			VerificationRuns: append([]PacketVerificationOutcome(nil), target.Verification...),
			DurableUpdates:   append([]PacketDurableUpdate(nil), target.Durable...),
			SessionClose:     target.SessionClose,
		},
	}
	for _, item := range target.IncludedItems {
		diagnostics := analysis.itemsByID[item.ItemID]
		explanation.IncludedItems = append(explanation.IncludedItems, ExplainedPacketItem{
			ItemID:      item.ItemID,
			Section:     item.Section,
			Anchor:      item.Anchor,
			Reason:      item.Reason,
			Expanded:    target.Expanded[item.ItemID],
			Diagnostics: diagnostics.ItemUtilityDiagnostic,
		})
	}
	return explanation, nil
}

func (m *Manager) BuildUtilitySnapshot(projectDir string) (*UtilitySnapshot, error) {
	projectDir, err := filepath.Abs(defaultProjectDir(projectDir))
	if err != nil {
		return nil, err
	}
	analysis, err := m.buildTelemetryAnalysis(projectDir)
	if err != nil {
		return nil, err
	}
	return &UtilitySnapshot{
		SessionsAnalyzed:      analysis.sessionsAnalyzed,
		PacketsAnalyzed:       len(analysis.packets),
		ItemsAnalyzed:         len(analysis.items),
		Items:                 append([]UtilityItemStat(nil), analysis.items...),
		VerificationLinks:     append([]VerificationLinkStat(nil), analysis.verification...),
		FreshPacketPressure:   analysis.freshPressure,
		FrequentlyOmittedDocs: trimOmittedDocs(analysis.omittedDocs, 0),
	}, nil
}

func (m *Manager) ContextStats(req ContextStatsRequest) (*ContextStats, error) {
	snapshot, err := m.BuildUtilitySnapshot(req.ProjectDir)
	if err != nil {
		return nil, err
	}
	if snapshot.PacketsAnalyzed == 0 {
		return nil, errors.New("no compiled context packets recorded yet")
	}
	limit := req.Limit
	if limit <= 0 {
		limit = 5
	}
	if limit > 10 {
		limit = 10
	}
	topSignal := filterUtilityItems(snapshot.Items, func(item UtilityItemStat) bool {
		return item.LikelyUtility == "likely_signal"
	})
	topNoise := filterUtilityItems(snapshot.Items, func(item UtilityItemStat) bool {
		return item.LikelyUtility == "likely_noise"
	})
	frequentlyExpanded := filterUtilityItems(snapshot.Items, func(item UtilityItemStat) bool {
		return item.ExpandCount > 0
	})
	sort.Slice(topSignal, func(i, j int) bool { return utilitySignalLess(topSignal[i], topSignal[j]) })
	sort.Slice(topNoise, func(i, j int) bool { return utilityNoiseLess(topNoise[i], topNoise[j]) })
	sort.Slice(frequentlyExpanded, func(i, j int) bool { return utilityExpandedLess(frequentlyExpanded[i], frequentlyExpanded[j]) })
	return &ContextStats{
		SessionsAnalyzed:        snapshot.SessionsAnalyzed,
		PacketsAnalyzed:         snapshot.PacketsAnalyzed,
		ItemsAnalyzed:           snapshot.ItemsAnalyzed,
		TopSignal:               trimUtilityItems(topSignal, limit),
		TopNoise:                trimUtilityItems(topNoise, limit),
		FrequentlyExpanded:      trimUtilityItems(frequentlyExpanded, limit),
		CommonVerificationLinks: trimVerificationLinks(snapshot.VerificationLinks, limit),
		FreshPacketPressure:     snapshot.FreshPacketPressure,
		FrequentlyOmittedDocs:   trimOmittedDocs(snapshot.FrequentlyOmittedDocs, limit),
	}, nil
}

func (m *Manager) ContextEffectiveness(req ContextEffectivenessRequest) (*ContextEffectiveness, error) {
	projectDir, err := filepath.Abs(defaultProjectDir(req.ProjectDir))
	if err != nil {
		return nil, err
	}
	analysis, err := m.buildTelemetryAnalysis(projectDir)
	if err != nil {
		return nil, err
	}
	if len(analysis.packets) == 0 {
		return nil, errors.New("no compiled context packets recorded yet")
	}
	limit := normalizedTelemetryLimit(req.Limit)
	topSignal := filterUtilityItems(analysis.items, func(item UtilityItemStat) bool {
		return item.LikelyUtility == "likely_signal"
	})
	topNoise := filterUtilityItems(analysis.items, func(item UtilityItemStat) bool {
		return item.LikelyUtility == "likely_noise"
	})
	sort.Slice(topSignal, func(i, j int) bool { return utilitySignalLess(topSignal[i], topSignal[j]) })
	sort.Slice(topNoise, func(i, j int) bool { return utilityNoiseLess(topNoise[i], topNoise[j]) })

	effectiveness := &ContextEffectiveness{
		SessionsAnalyzed:      analysis.sessionsAnalyzed,
		PacketsAnalyzed:       len(analysis.packets),
		ItemsAnalyzed:         len(analysis.items),
		PacketUse:             summarizePacketUse(analysis.packets, analysis.sessionsAnalyzed),
		Cache:                 summarizePacketCache(analysis.packets),
		Budget:                summarizePacketBudget(analysis.packets, analysis.freshPressure),
		Outcomes:              summarizePacketOutcomes(analysis.packets),
		TopSignal:             trimUtilityItems(topSignal, limit),
		TopNoise:              trimUtilityItems(topNoise, limit),
		FrequentlyOmittedDocs: trimOmittedDocs(analysis.omittedDocs, limit),
	}
	effectiveness.TelemetryGaps = contextTelemetryGaps(effectiveness)
	effectiveness.Recommendations = contextEffectivenessRecommendations(effectiveness)
	return effectiveness, nil
}

func normalizedTelemetryLimit(limit int) int {
	if limit <= 0 {
		return 5
	}
	if limit > 10 {
		return 10
	}
	return limit
}

func (m *Manager) buildTelemetryAnalysis(projectDir string) (*telemetryAnalysis, error) {
	sessions, err := loadTelemetrySessions(projectDir)
	if err != nil {
		return nil, err
	}
	packets := []packetObservation{}
	for _, current := range sessions {
		packets = append(packets, packetObservationsFromSession(current)...)
	}
	sort.Slice(packets, func(i, j int) bool {
		if packets[i].CompiledAt.Equal(packets[j].CompiledAt) {
			if packets[i].SessionID == packets[j].SessionID {
				return packets[i].PacketHash > packets[j].PacketHash
			}
			return packets[i].SessionID > packets[j].SessionID
		}
		return packets[i].CompiledAt.After(packets[j].CompiledAt)
	})
	if len(packets) > maxTelemetryAnalysisPackets {
		packets = append([]packetObservation(nil), packets[:maxTelemetryAnalysisPackets]...)
	}

	itemAgg := map[string]*utilityAggregate{}
	verifAgg := map[string]*verificationAggregate{}
	omittedAgg := map[string]*OmittedDocStat{}
	freshPressure := FreshPacketPressure{}
	for _, packet := range packets {
		successfulVerifications := countVerificationOutcomes(packet.Verification, true)
		failedVerifications := countVerificationOutcomes(packet.Verification, false)
		successfulClose := packet.SessionClose != nil && packet.SessionClose.Success
		if packet.CacheStatus == projectcontext.PacketCacheStatusFresh {
			freshPressure.FreshPacketsAnalyzed++
			if packet.Budget.MandatoryOverTarget {
				freshPressure.MandatoryOverTarget++
			}
			if packet.Budget.MandatoryOverTarget || packet.Budget.OmittedDueToBudget > 0 {
				freshPressure.FreshPacketsUnderPressure++
			}
			seenPaths := map[string]struct{}{}
			for _, item := range packet.Budget.OmittedCandidates {
				path := strings.TrimSpace(item.Anchor.Path)
				if !isMarkdownDocPath(path) {
					continue
				}
				if _, exists := seenPaths[path]; exists {
					continue
				}
				seenPaths[path] = struct{}{}
				entry := omittedAgg[path]
				if entry == nil {
					entry = &OmittedDocStat{Path: path}
					omittedAgg[path] = entry
				}
				entry.OmittedPackets++
			}
		}
		for _, run := range packet.Verification {
			entry := verifAgg[run.Command]
			if entry == nil {
				entry = &verificationAggregate{Command: run.Command}
				verifAgg[run.Command] = entry
			}
			entry.PacketCount++
			if run.Success {
				entry.SuccessCount++
			}
		}
		for _, item := range packet.IncludedItems {
			entry := itemAgg[item.ItemID]
			if entry == nil {
				entry = &utilityAggregate{
					ItemID:  item.ItemID,
					Section: item.Section,
					Anchor:  item.Anchor,
				}
				itemAgg[item.ItemID] = entry
			}
			entry.IncludeCount++
			entry.LastIncludedAt = maxTime(entry.LastIncludedAt, packet.CompiledAt)
			if packet.Expanded[item.ItemID] > 0 {
				entry.ExpandCount += packet.Expanded[item.ItemID]
			}
			entry.SuccessfulVerificationCount += successfulVerifications
			entry.FailedVerificationCount += failedVerifications
			entry.DurableUpdateCount += len(packet.Durable)
			if successfulClose {
				entry.SuccessfulSessionCloseCount++
			}
			if packet.Expanded[item.ItemID] == 0 && successfulVerifications == 0 && len(packet.Durable) == 0 && !successfulClose {
				entry.UnusedIncludeCount++
			}
		}
	}

	items := make([]UtilityItemStat, 0, len(itemAgg))
	itemsByID := map[string]UtilityItemStat{}
	for _, agg := range itemAgg {
		stat := utilityStatFromAggregate(*agg)
		items = append(items, stat)
		itemsByID[stat.ItemID] = stat
	}
	sort.Slice(items, func(i, j int) bool { return utilityStatLess(items[i], items[j]) })

	verification := make([]VerificationLinkStat, 0, len(verifAgg))
	for _, agg := range verifAgg {
		verification = append(verification, VerificationLinkStat{
			Command:      agg.Command,
			PacketCount:  agg.PacketCount,
			SuccessCount: agg.SuccessCount,
		})
	}
	sort.Slice(verification, func(i, j int) bool {
		if verification[i].SuccessCount == verification[j].SuccessCount {
			if verification[i].PacketCount == verification[j].PacketCount {
				return verification[i].Command < verification[j].Command
			}
			return verification[i].PacketCount > verification[j].PacketCount
		}
		return verification[i].SuccessCount > verification[j].SuccessCount
	})

	omittedDocs := make([]OmittedDocStat, 0, len(omittedAgg))
	for _, agg := range omittedAgg {
		omittedDocs = append(omittedDocs, *agg)
	}
	sort.Slice(omittedDocs, func(i, j int) bool {
		if omittedDocs[i].OmittedPackets == omittedDocs[j].OmittedPackets {
			return omittedDocs[i].Path < omittedDocs[j].Path
		}
		return omittedDocs[i].OmittedPackets > omittedDocs[j].OmittedPackets
	})
	if freshPressure.FreshPacketsAnalyzed > 0 {
		freshPressure.PressureRate = float64(freshPressure.FreshPacketsUnderPressure) / float64(freshPressure.FreshPacketsAnalyzed)
	}

	return &telemetryAnalysis{
		sessionsAnalyzed: len(sessions),
		packets:          packets,
		items:            items,
		itemsByID:        itemsByID,
		verification:     verification,
		freshPressure:    freshPressure,
		omittedDocs:      omittedDocs,
	}, nil
}

func summarizePacketUse(packets []packetObservation, sessionsAnalyzed int) PacketUseSummary {
	summary := PacketUseSummary{
		SessionsWithPackets: sessionsAnalyzed,
		PacketsAnalyzed:     len(packets),
	}
	if len(packets) == 0 {
		return summary
	}
	bySession := map[string]int{}
	for _, packet := range packets {
		bySession[packet.SessionID]++
		if summary.FirstPacketAt.IsZero() || packet.CompiledAt.Before(summary.FirstPacketAt) {
			summary.FirstPacketAt = packet.CompiledAt
		}
		if summary.LastPacketAt.IsZero() || packet.CompiledAt.After(summary.LastPacketAt) {
			summary.LastPacketAt = packet.CompiledAt
		}
	}
	for _, count := range bySession {
		if count == 1 {
			summary.SinglePacketSessions++
		}
		if count > 1 {
			summary.MultiPacketSessions++
		}
		if count > summary.MaxPacketsInSession {
			summary.MaxPacketsInSession = count
		}
	}
	if len(bySession) > 0 {
		summary.AveragePacketsPerSession = float64(len(packets)) / float64(len(bySession))
	}
	return summary
}

func summarizePacketCache(packets []packetObservation) PacketCacheSummary {
	summary := PacketCacheSummary{}
	for _, packet := range packets {
		switch packet.CacheStatus {
		case projectcontext.PacketCacheStatusFresh:
			summary.FreshPackets++
		case projectcontext.PacketCacheStatusReused:
			summary.ReusedPackets++
		case projectcontext.PacketCacheStatusDelta:
			summary.DeltaPackets++
		default:
			summary.UnknownPackets++
		}
		if packet.FullPacketIncluded {
			summary.FullPackets++
		} else {
			summary.CompactPackets++
		}
	}
	return summary
}

func summarizePacketBudget(packets []packetObservation, pressure FreshPacketPressure) PacketBudgetSummary {
	summary := PacketBudgetSummary{
		FreshPacketsUnderPressure: pressure.FreshPacketsUnderPressure,
		MandatoryOverTarget:       pressure.MandatoryOverTarget,
		PressureRate:              pressure.PressureRate,
	}
	fullPackets := 0
	targetTotal := 0
	usedTotal := 0
	remainingTotal := 0
	omittedTotal := 0
	for _, packet := range packets {
		if !packet.FullPacketIncluded {
			continue
		}
		fullPackets++
		targetTotal += packet.Budget.Target
		usedTotal += packet.Budget.Used
		remainingTotal += packet.Budget.Remaining
		omittedTotal += packet.Budget.OmittedDueToBudget
	}
	summary.FullPacketsAnalyzed = fullPackets
	if fullPackets == 0 {
		return summary
	}
	summary.AverageFullTarget = targetTotal / fullPackets
	summary.AverageFullUsed = usedTotal / fullPackets
	summary.AverageFullRemaining = remainingTotal / fullPackets
	summary.AverageFullOmitted = float64(omittedTotal) / float64(fullPackets)
	return summary
}

func summarizePacketOutcomes(packets []packetObservation) PacketOutcomeSummary {
	summary := PacketOutcomeSummary{}
	for _, packet := range packets {
		expansions := 0
		for _, count := range packet.Expanded {
			expansions += count
		}
		if expansions > 0 {
			summary.PacketsWithExpansions++
			summary.ExpansionEvents += expansions
		}
		successes := countVerificationOutcomes(packet.Verification, true)
		failures := countVerificationOutcomes(packet.Verification, false)
		if successes > 0 {
			summary.PacketsWithSuccessfulVerification++
			summary.SuccessfulVerificationEvents += successes
		}
		summary.FailedVerificationEvents += failures
		if len(packet.Durable) > 0 {
			summary.PacketsWithDurableUpdates++
			summary.DurableUpdateEvents += len(packet.Durable)
		}
		if packet.SessionClose != nil && packet.SessionClose.Success {
			summary.SuccessfulSessionCloses++
		}
	}
	return summary
}

func contextEffectivenessRecommendations(effectiveness *ContextEffectiveness) []string {
	if effectiveness == nil {
		return nil
	}
	recommendations := []string{}
	if effectiveness.Budget.PressureRate >= 0.5 {
		recommendations = append(recommendations, "Fresh packets are frequently under budget pressure; inspect repeated omissions and consider tighter summaries or capsule-style source summaries before raising defaults.")
	}
	if len(effectiveness.FrequentlyOmittedDocs) > 0 {
		recommendations = append(recommendations, fmt.Sprintf("Start packet-shaping review with `%s`, the most frequently omitted markdown source.", effectiveness.FrequentlyOmittedDocs[0].Path))
	}
	if len(effectiveness.TopNoise) > 0 {
		recommendations = append(recommendations, "Review likely-noise includes and suppress or narrow selection rules that repeatedly add context with no recorded downstream signal.")
	}
	if effectiveness.Outcomes.ExpansionEvents == 0 || effectiveness.Outcomes.ExpansionEvents*10 < effectiveness.PacketsAnalyzed {
		recommendations = append(recommendations, "Do not treat low expansion counts as proof packets are sufficient yet; add telemetry for post-packet search and non-`brain read` context access.")
	}
	if effectiveness.Outcomes.SuccessfulVerificationEvents == 0 {
		recommendations = append(recommendations, "Run verification through `brain session run -- <command>` so packet usefulness can be linked to execution outcomes.")
	}
	if effectiveness.Outcomes.DurableUpdateEvents == 0 {
		recommendations = append(recommendations, "Record durable note updates during packet-backed work so Brain can link context packets to memory quality.")
	}
	if len(recommendations) == 0 {
		recommendations = append(recommendations, "Current packet telemetry has no obvious pressure or noise signal; next improvement should add stronger quality feedback instead of changing selection heuristics.")
	}
	return recommendations
}

func contextTelemetryGaps(effectiveness *ContextEffectiveness) []string {
	gaps := []string{
		"Expansion telemetry currently records matching `brain read` calls; raw shell, editor, and agent file reads are invisible.",
		"Post-packet search, user correction, and human quality rating signals are not recorded yet.",
		"Verification and durable-update links are useful correlation signals, not proof that packet contents caused better output.",
	}
	if effectiveness != nil && effectiveness.Cache.UnknownPackets > 0 {
		gaps = append(gaps, "Some older packet records lack cache status; compact legacy packets may predate reuse and delta metadata.")
	}
	return gaps
}

func trimOmittedDocs(items []OmittedDocStat, limit int) []OmittedDocStat {
	if len(items) == 0 {
		return []OmittedDocStat{}
	}
	if limit <= 0 || len(items) <= limit {
		return append([]OmittedDocStat(nil), items...)
	}
	return append([]OmittedDocStat(nil), items[:limit]...)
}

func isMarkdownDocPath(path string) bool {
	return strings.HasSuffix(strings.ToLower(strings.TrimSpace(path)), ".md")
}

func loadTelemetrySessions(projectDir string) ([]*ActiveSession, error) {
	projectDir, err := filepath.Abs(defaultProjectDir(projectDir))
	if err != nil {
		return nil, err
	}
	policy, _, _, err := projectcontext.LoadPolicy(projectDir)
	if err != nil {
		return nil, err
	}
	activePath := filepath.Join(projectDir, filepath.FromSlash(policy.Session.ActiveFile))
	sessions := []*ActiveSession{}
	active, err := loadActiveSessionIfExists(activePath)
	if err != nil {
		return nil, err
	}
	if active != nil && len(active.PacketRecords) != 0 {
		sessions = append(sessions, active)
	}

	ledgerDir := filepath.Join(projectDir, filepath.FromSlash(policy.Session.LedgerDir))
	entries, err := os.ReadDir(ledgerDir)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	paths := []string{}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		paths = append(paths, filepath.Join(ledgerDir, entry.Name()))
	}
	sort.Slice(paths, func(i, j int) bool {
		return filepath.Base(paths[i]) > filepath.Base(paths[j])
	})
	for _, path := range paths {
		if len(sessions) >= maxTelemetryAnalysisSessions {
			break
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read session ledger %s: %w", filepath.ToSlash(path), err)
		}
		var current ActiveSession
		if err := jsonUnmarshal(raw, &current); err != nil {
			return nil, fmt.Errorf("parse session ledger %s: %w", filepath.ToSlash(path), err)
		}
		if len(current.PacketRecords) == 0 {
			continue
		}
		sessions = append(sessions, &current)
	}
	sort.Slice(sessions, func(i, j int) bool {
		return sessionTelemetryTime(sessions[i]).After(sessionTelemetryTime(sessions[j]))
	})
	if len(sessions) > maxTelemetryAnalysisSessions {
		sessions = append([]*ActiveSession(nil), sessions[:maxTelemetryAnalysisSessions]...)
	}
	return sessions, nil
}

func sessionTelemetryTime(active *ActiveSession) time.Time {
	if active == nil {
		return time.Time{}
	}
	if len(active.PacketRecords) != 0 {
		return active.PacketRecords[len(active.PacketRecords)-1].CompiledAt
	}
	if active.EndedAt != nil {
		return active.EndedAt.UTC()
	}
	return active.StartedAt.UTC()
}

func packetObservationsFromSession(active *ActiveSession) []packetObservation {
	if active == nil || len(active.PacketRecords) == 0 {
		return nil
	}
	packets := make([]packetObservation, 0, len(active.PacketRecords))
	for _, record := range active.PacketRecords {
		observation := packetObservation{
			PacketReference: PacketReference{
				PacketHash:  record.PacketHash,
				CompiledAt:  record.CompiledAt.UTC(),
				TaskText:    record.TaskText,
				TaskSummary: record.TaskSummary,
				TaskSource:  record.TaskSource,
				Budget:      record.Budget,
				PacketCacheMetadata: projectcontext.PacketCacheMetadata{
					CacheStatus:         record.CacheStatus,
					Fingerprint:         record.Fingerprint,
					ReusedFrom:          record.ReusedFrom,
					DeltaFrom:           record.DeltaFrom,
					ChangedSections:     append([]string(nil), record.ChangedSections...),
					ChangedItemIDs:      append([]string(nil), record.ChangedItemIDs...),
					InvalidationReasons: append([]string(nil), record.InvalidationReasons...),
					FullPacketIncluded:  record.FullPacketIncluded,
					FallbackReason:      record.FallbackReason,
				},
				SessionID:     active.ID,
				SessionStatus: active.Status,
			},
			IncludedItems: make([]packetIncludedItem, 0, len(record.IncludedItemIDs)),
			Expanded:      map[string]int{},
		}
		for i, itemID := range record.IncludedItemIDs {
			item := packetIncludedItem{ItemID: itemID}
			if i < len(record.InclusionReasons) {
				item.Section = record.InclusionReasons[i].Section
				item.Reason = record.InclusionReasons[i].Reason
			}
			if i < len(record.IncludedAnchors) {
				item.Anchor = record.IncludedAnchors[i]
			}
			observation.IncludedItems = append(observation.IncludedItems, item)
		}
		packets = append(packets, observation)
	}
	for _, event := range active.TelemetryEvents {
		index := packetObservationIndexForEvent(packets, event)
		if index < 0 {
			continue
		}
		switch event.Type {
		case PacketTelemetryEventExpanded:
			if strings.TrimSpace(event.ItemID) == "" {
				continue
			}
			packets[index].Expanded[event.ItemID]++
		case PacketTelemetryEventVerification:
			packets[index].Verification = append(packets[index].Verification, PacketVerificationOutcome{
				Command:   event.Command,
				Success:   event.Success == nil || *event.Success,
				Timestamp: event.Timestamp.UTC(),
			})
		case PacketTelemetryEventDurableUpdate:
			packets[index].Durable = append(packets[index].Durable, PacketDurableUpdate{
				File:      event.File,
				Operation: event.Operation,
				Timestamp: event.Timestamp.UTC(),
			})
		case PacketTelemetryEventSessionClosed:
			closeEvent := &PacketSessionClose{
				Status:    event.CloseStatus,
				Success:   event.Success == nil || *event.Success,
				Timestamp: event.Timestamp.UTC(),
			}
			if memorySatisfiedBy, ok := stringMetadata(event.Metadata, "memory_satisfied_by"); ok {
				closeEvent.MemorySatisfiedBy = memorySatisfiedBy
			}
			if missingCommands := stringSliceMetadata(event.Metadata, "missing_commands"); len(missingCommands) != 0 {
				closeEvent.MissingCommands = missingCommands
			}
			packets[index].SessionClose = closeEvent
		}
	}
	return packets
}

func packetObservationIndexForEvent(packets []packetObservation, event PacketTelemetryEvent) int {
	if len(packets) == 0 || strings.TrimSpace(event.PacketHash) == "" {
		return -1
	}
	index := -1
	for i := range packets {
		if packets[i].PacketHash != event.PacketHash {
			continue
		}
		if packets[i].CompiledAt.After(event.Timestamp) {
			continue
		}
		index = i
	}
	if index >= 0 {
		return index
	}
	for i := len(packets) - 1; i >= 0; i-- {
		if packets[i].PacketHash == event.PacketHash {
			return i
		}
	}
	return -1
}

func selectExplainedPacket(packets []packetObservation, packetHash string) *packetObservation {
	if len(packets) == 0 {
		return nil
	}
	if packetHash == "" {
		return &packets[0]
	}
	for i := range packets {
		if packets[i].PacketHash == packetHash {
			return &packets[i]
		}
	}
	return nil
}

func expandedItemsForPacket(packet packetObservation) []PacketExpandedItem {
	items := make([]PacketExpandedItem, 0, len(packet.Expanded))
	for _, included := range packet.IncludedItems {
		count := packet.Expanded[included.ItemID]
		if count == 0 {
			continue
		}
		items = append(items, PacketExpandedItem{
			ItemID: included.ItemID,
			Anchor: included.Anchor,
			Count:  count,
		})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Count == items[j].Count {
			return items[i].ItemID < items[j].ItemID
		}
		return items[i].Count > items[j].Count
	})
	return items
}

func countVerificationOutcomes(runs []PacketVerificationOutcome, success bool) int {
	total := 0
	for _, run := range runs {
		if run.Success == success {
			total++
		}
	}
	return total
}

func utilityStatFromAggregate(agg utilityAggregate) UtilityItemStat {
	diagnostics := ItemUtilityDiagnostic{
		IncludeCount:                agg.IncludeCount,
		ExpandCount:                 agg.ExpandCount,
		SuccessfulVerificationCount: agg.SuccessfulVerificationCount,
		FailedVerificationCount:     agg.FailedVerificationCount,
		DurableUpdateCount:          agg.DurableUpdateCount,
		SuccessfulSessionCloseCount: agg.SuccessfulSessionCloseCount,
		UnusedIncludeCount:          agg.UnusedIncludeCount,
	}
	diagnostics.UtilityScore = agg.ExpandCount*5 + agg.DurableUpdateCount*4 + agg.SuccessfulVerificationCount*2 + agg.SuccessfulSessionCloseCount
	diagnostics.NoiseScore = agg.UnusedIncludeCount*3 + agg.FailedVerificationCount
	diagnostics.LikelyUtility, diagnostics.Reasons = classifyUtility(agg.Section, agg, diagnostics)
	return UtilityItemStat{
		ItemID:                agg.ItemID,
		Section:               agg.Section,
		Anchor:                agg.Anchor,
		ItemUtilityDiagnostic: diagnostics,
	}
}

func classifyUtility(section string, agg utilityAggregate, diagnostics ItemUtilityDiagnostic) (string, []string) {
	reasons := []string{}
	if agg.ExpandCount > 0 {
		reasons = append(reasons, fmt.Sprintf("expanded after inclusion %d time(s)", agg.ExpandCount))
	}
	if agg.SuccessfulVerificationCount > 0 {
		reasons = append(reasons, fmt.Sprintf("included in %d packet(s) with later successful verification", agg.SuccessfulVerificationCount))
	}
	if agg.DurableUpdateCount > 0 {
		reasons = append(reasons, fmt.Sprintf("included in %d packet(s) that later recorded durable updates", agg.DurableUpdateCount))
	}
	if agg.UnusedIncludeCount > 0 {
		reasons = append(reasons, fmt.Sprintf("included %d time(s) with no recorded expansion or downstream success", agg.UnusedIncludeCount))
	}
	switch {
	case agg.IncludeCount >= 2 && (agg.ExpandCount >= 2 || agg.DurableUpdateCount >= 1 || (agg.ExpandCount >= 1 && agg.SuccessfulVerificationCount >= 1)):
		return "likely_signal", reasons
	case section != "base_contract" && agg.IncludeCount >= 3 && agg.ExpandCount == 0 && agg.DurableUpdateCount == 0 && agg.SuccessfulVerificationCount == 0 && agg.SuccessfulSessionCloseCount == 0:
		return "likely_noise", reasons
	case agg.IncludeCount < 2:
		return "insufficient_evidence", reasons
	default:
		return "mixed", reasons
	}
}

func utilityStatLess(a, b UtilityItemStat) bool {
	if a.UtilityScore == b.UtilityScore {
		if a.NoiseScore == b.NoiseScore {
			if a.IncludeCount == b.IncludeCount {
				return a.ItemID < b.ItemID
			}
			return a.IncludeCount > b.IncludeCount
		}
		return a.NoiseScore < b.NoiseScore
	}
	return a.UtilityScore > b.UtilityScore
}

func utilitySignalLess(a, b UtilityItemStat) bool {
	if a.UtilityScore == b.UtilityScore {
		if a.ExpandCount == b.ExpandCount {
			return a.ItemID < b.ItemID
		}
		return a.ExpandCount > b.ExpandCount
	}
	return a.UtilityScore > b.UtilityScore
}

func utilityNoiseLess(a, b UtilityItemStat) bool {
	if a.NoiseScore == b.NoiseScore {
		if a.IncludeCount == b.IncludeCount {
			return a.ItemID < b.ItemID
		}
		return a.IncludeCount > b.IncludeCount
	}
	return a.NoiseScore > b.NoiseScore
}

func utilityExpandedLess(a, b UtilityItemStat) bool {
	if a.ExpandCount == b.ExpandCount {
		if a.IncludeCount == b.IncludeCount {
			return a.ItemID < b.ItemID
		}
		return a.IncludeCount > b.IncludeCount
	}
	return a.ExpandCount > b.ExpandCount
}

func filterUtilityItems(items []UtilityItemStat, keep func(UtilityItemStat) bool) []UtilityItemStat {
	out := []UtilityItemStat{}
	for _, item := range items {
		if keep(item) {
			out = append(out, item)
		}
	}
	return out
}

func trimUtilityItems(items []UtilityItemStat, limit int) []UtilityItemStat {
	if len(items) <= limit {
		return items
	}
	return append([]UtilityItemStat(nil), items[:limit]...)
}

func trimVerificationLinks(items []VerificationLinkStat, limit int) []VerificationLinkStat {
	if len(items) <= limit {
		return items
	}
	return append([]VerificationLinkStat(nil), items[:limit]...)
}

func stringMetadata(meta map[string]any, key string) (string, bool) {
	if len(meta) == 0 {
		return "", false
	}
	value, ok := meta[key]
	if !ok {
		return "", false
	}
	text, ok := value.(string)
	return text, ok
}

func stringSliceMetadata(meta map[string]any, key string) []string {
	if len(meta) == 0 {
		return nil
	}
	value, ok := meta[key]
	if !ok {
		return nil
	}
	raw, ok := value.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		text, ok := item.(string)
		if !ok || strings.TrimSpace(text) == "" {
			continue
		}
		out = append(out, text)
	}
	return out
}

func maxTime(a, b time.Time) time.Time {
	if a.After(b) {
		return a
	}
	return b
}
