package session

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"brain/internal/projectcontext"
)

func RenderPacketExplanationHuman(w io.Writer, explanation *PacketExplanation) error {
	if explanation == nil {
		return errors.New("packet explanation is required")
	}
	if _, err := fmt.Fprintf(
		w,
		"## Packet\n\n- Hash: `%s`\n- Compiled: %s\n- Task: `%s`\n- Summary: %s\n- Source: `%s`\n- Session: `%s` (%s)\n- Cache status: `%s`\n- Fingerprint: `%s`\n- Full packet included: %t\n\n",
		explanation.Packet.PacketHash,
		explanation.Packet.CompiledAt.Format(timeLayout),
		explanation.Packet.TaskText,
		explanation.Packet.TaskSummary,
		explanation.Packet.TaskSource,
		explanation.Packet.SessionID,
		explanation.Packet.SessionStatus,
		explanation.Packet.CacheStatus,
		explanation.Packet.Fingerprint,
		explanation.Packet.FullPacketIncluded,
	); err != nil {
		return err
	}
	if err := renderPacketBudget(w, explanation.Packet.Budget); err != nil {
		return err
	}
	if _, err := io.WriteString(w, "## Lineage\n\n"); err != nil {
		return err
	}
	if explanation.Packet.ReusedFrom != "" {
		if _, err := fmt.Fprintf(w, "- Reused from: `%s`\n", explanation.Packet.ReusedFrom); err != nil {
			return err
		}
	}
	if explanation.Packet.DeltaFrom != "" {
		if _, err := fmt.Fprintf(w, "- Delta from: `%s`\n", explanation.Packet.DeltaFrom); err != nil {
			return err
		}
	}
	if explanation.Packet.FallbackReason != "" {
		if _, err := fmt.Fprintf(w, "- Full-packet fallback: %s\n", explanation.Packet.FallbackReason); err != nil {
			return err
		}
	}
	for _, reason := range explanation.Packet.InvalidationReasons {
		if _, err := fmt.Fprintf(w, "- Invalidation: %s\n", reason); err != nil {
			return err
		}
	}
	if len(explanation.Packet.ChangedSections) != 0 {
		if _, err := fmt.Fprintf(w, "- Changed sections: %s\n", strings.Join(explanation.Packet.ChangedSections, ", ")); err != nil {
			return err
		}
	}
	if len(explanation.Packet.ChangedItemIDs) != 0 {
		if _, err := fmt.Fprintf(w, "- Changed item ids: %s\n", strings.Join(explanation.Packet.ChangedItemIDs, ", ")); err != nil {
			return err
		}
	}
	if _, err := io.WriteString(w, "\n"); err != nil {
		return err
	}
	if _, err := io.WriteString(w, "## Included Items\n\n"); err != nil {
		return err
	}
	if len(explanation.IncludedItems) == 0 {
		if _, err := io.WriteString(w, "- None recorded.\n"); err != nil {
			return err
		}
	} else {
		for _, item := range explanation.IncludedItems {
			if _, err := fmt.Fprintf(w, "- `%s` [%s] (`%s`): %s\n", item.ItemID, item.Section, anchorLabel(item.Anchor), item.Reason); err != nil {
				return err
			}
			if item.Expanded > 0 {
				if _, err := fmt.Fprintf(w, "  Expanded later: %d time(s)\n", item.Expanded); err != nil {
					return err
				}
			}
			if _, err := fmt.Fprintf(
				w,
				"  Diagnostics: include=%d expand=%d verify_success=%d durable_updates=%d likely_utility=%s\n",
				item.Diagnostics.IncludeCount,
				item.Diagnostics.ExpandCount,
				item.Diagnostics.SuccessfulVerificationCount,
				item.Diagnostics.DurableUpdateCount,
				item.Diagnostics.LikelyUtility,
			); err != nil {
				return err
			}
			for _, reason := range item.Diagnostics.Reasons {
				if _, err := fmt.Fprintf(w, "  Utility signal: %s\n", reason); err != nil {
					return err
				}
			}
		}
	}
	if _, err := io.WriteString(w, "\n## Expanded Later\n\n"); err != nil {
		return err
	}
	if len(explanation.ExpandedLater) == 0 {
		if _, err := io.WriteString(w, "- No later expansions were recorded for this packet.\n"); err != nil {
			return err
		}
	} else {
		for _, item := range explanation.ExpandedLater {
			if _, err := fmt.Fprintf(w, "- `%s` (`%s`): %d time(s)\n", item.ItemID, anchorLabel(item.Anchor), item.Count); err != nil {
				return err
			}
		}
	}
	if _, err := io.WriteString(w, "\n## Downstream Outcomes\n\n"); err != nil {
		return err
	}
	if len(explanation.Downstream.VerificationRuns) == 0 {
		if _, err := io.WriteString(w, "- Verification: none recorded.\n"); err != nil {
			return err
		}
	} else {
		for _, run := range explanation.Downstream.VerificationRuns {
			status := "failed"
			if run.Success {
				status = "succeeded"
			}
			if _, err := fmt.Fprintf(w, "- Verification %s: `%s`\n", status, run.Command); err != nil {
				return err
			}
		}
	}
	if len(explanation.Downstream.DurableUpdates) == 0 {
		if _, err := io.WriteString(w, "- Durable updates: none recorded.\n"); err != nil {
			return err
		}
	} else {
		for _, update := range explanation.Downstream.DurableUpdates {
			if _, err := fmt.Fprintf(w, "- Durable update [%s]: `%s`\n", update.Operation, update.File); err != nil {
				return err
			}
		}
	}
	if explanation.Downstream.SessionClose == nil {
		if _, err := io.WriteString(w, "- Session closeout: session is still active or no closeout event was recorded.\n"); err != nil {
			return err
		}
		return nil
	}
	closeStatus := "not_ok"
	if explanation.Downstream.SessionClose.Success {
		closeStatus = "ok"
	}
	if _, err := fmt.Fprintf(w, "- Session closeout [%s]: %s\n", explanation.Downstream.SessionClose.Status, closeStatus); err != nil {
		return err
	}
	if explanation.Downstream.SessionClose.MemorySatisfiedBy != "" {
		if _, err := fmt.Fprintf(w, "  Memory satisfied by: `%s`\n", explanation.Downstream.SessionClose.MemorySatisfiedBy); err != nil {
			return err
		}
	}
	if len(explanation.Downstream.SessionClose.MissingCommands) != 0 {
		if _, err := fmt.Fprintf(w, "  Missing commands: %s\n", strings.Join(explanation.Downstream.SessionClose.MissingCommands, ", ")); err != nil {
			return err
		}
	}
	return nil
}

func renderPacketBudget(w io.Writer, budget projectcontext.PacketBudget) error {
	if _, err := io.WriteString(w, "## Budget\n\n"); err != nil {
		return err
	}
	if budget.Preset != "" {
		if _, err := fmt.Fprintf(w, "- Preset: `%s`\n", budget.Preset); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintf(
		w,
		"- Target: %d tokens\n- Used: %d tokens\n- Remaining: %d tokens\n- Reserve base contract: %d tokens\n- Reserve verification: %d tokens\n- Reserve diagnostics: %d tokens\n- Omitted due to budget: %d\n",
		budget.Target,
		budget.Used,
		budget.Remaining,
		budget.ReserveBaseContract,
		budget.ReserveVerification,
		budget.ReserveDiagnostics,
		budget.OmittedDueToBudget,
	); err != nil {
		return err
	}
	if budget.MandatoryOverTarget {
		if _, err := io.WriteString(w, "- Mandatory sections exceeded the target budget before optional working-set selection.\n"); err != nil {
			return err
		}
	}
	for _, item := range budget.OmittedCandidates {
		if _, err := fmt.Fprintf(w, "- Omitted `%s` [%s] (`%s`, %d tokens): %s\n", item.Title, item.Section, anchorLabel(item.Anchor), item.EstimatedTokens, item.Reason); err != nil {
			return err
		}
	}
	_, err := io.WriteString(w, "\n")
	return err
}

func RenderContextStatsHuman(w io.Writer, stats *ContextStats) error {
	if stats == nil {
		return errors.New("context stats are required")
	}
	if _, err := fmt.Fprintf(
		w,
		"## Context Stats\n\n- Sessions analyzed: %d\n- Packets analyzed: %d\n- Items analyzed: %d\n\n",
		stats.SessionsAnalyzed,
		stats.PacketsAnalyzed,
		stats.ItemsAnalyzed,
	); err != nil {
		return err
	}
	if err := renderUtilityGroup(w, "Top Signal", stats.TopSignal); err != nil {
		return err
	}
	if err := renderUtilityGroup(w, "Top Noise", stats.TopNoise); err != nil {
		return err
	}
	if err := renderUtilityGroup(w, "Frequently Expanded", stats.FrequentlyExpanded); err != nil {
		return err
	}
	if err := renderFreshPacketPressure(w, stats.FreshPacketPressure); err != nil {
		return err
	}
	if err := renderOmittedDocs(w, "Frequently Omitted Docs", stats.FrequentlyOmittedDocs); err != nil {
		return err
	}
	if _, err := io.WriteString(w, "## Common Verification Links\n\n"); err != nil {
		return err
	}
	if len(stats.CommonVerificationLinks) == 0 {
		if _, err := io.WriteString(w, "- None recorded yet.\n"); err != nil {
			return err
		}
		return nil
	}
	for _, link := range stats.CommonVerificationLinks {
		if _, err := fmt.Fprintf(w, "- `%s`: packets=%d success=%d\n", link.Command, link.PacketCount, link.SuccessCount); err != nil {
			return err
		}
	}
	return nil
}

func RenderContextEffectivenessHuman(w io.Writer, effectiveness *ContextEffectiveness) error {
	if effectiveness == nil {
		return errors.New("context effectiveness report is required")
	}
	if _, err := fmt.Fprintf(
		w,
		"## Context Effectiveness\n\n- Sessions analyzed: %d\n- Packets analyzed: %d\n- Items analyzed: %d\n\n",
		effectiveness.SessionsAnalyzed,
		effectiveness.PacketsAnalyzed,
		effectiveness.ItemsAnalyzed,
	); err != nil {
		return err
	}
	if err := renderPacketUse(w, effectiveness.PacketUse); err != nil {
		return err
	}
	if err := renderCacheAndBudget(w, effectiveness.Cache, effectiveness.Budget); err != nil {
		return err
	}
	if err := renderOutcomeSummary(w, effectiveness.Outcomes); err != nil {
		return err
	}
	if err := renderUtilityGroup(w, "Likely Signal", effectiveness.TopSignal); err != nil {
		return err
	}
	if err := renderUtilityGroup(w, "Likely Noise", effectiveness.TopNoise); err != nil {
		return err
	}
	if err := renderOmittedDocs(w, "Likely Misses", effectiveness.FrequentlyOmittedDocs); err != nil {
		return err
	}
	if err := renderStringList(w, "Telemetry Gaps", effectiveness.TelemetryGaps); err != nil {
		return err
	}
	return renderStringList(w, "Recommendations", effectiveness.Recommendations)
}

func renderPacketUse(w io.Writer, summary PacketUseSummary) error {
	if _, err := io.WriteString(w, "## Packet Use\n\n"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(
		w,
		"- Sessions with packets: %d\n- Single-packet sessions: %d\n- Multi-packet sessions: %d\n- Average packets per session: %.2f\n- Max packets in one session: %d\n",
		summary.SessionsWithPackets,
		summary.SinglePacketSessions,
		summary.MultiPacketSessions,
		summary.AveragePacketsPerSession,
		summary.MaxPacketsInSession,
	); err != nil {
		return err
	}
	if !summary.FirstPacketAt.IsZero() && !summary.LastPacketAt.IsZero() {
		if _, err := fmt.Fprintf(w, "- Window: %s to %s\n", summary.FirstPacketAt.Format(timeLayout), summary.LastPacketAt.Format(timeLayout)); err != nil {
			return err
		}
	}
	_, err := io.WriteString(w, "\n")
	return err
}

func renderCacheAndBudget(w io.Writer, cache PacketCacheSummary, budget PacketBudgetSummary) error {
	if _, err := io.WriteString(w, "## Cache And Budget\n\n"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(
		w,
		"- Fresh packets: %d\n- Reused packets: %d\n- Delta packets: %d\n- Unknown cache status: %d\n- Full packets: %d\n- Compact packets: %d\n",
		cache.FreshPackets,
		cache.ReusedPackets,
		cache.DeltaPackets,
		cache.UnknownPackets,
		cache.FullPackets,
		cache.CompactPackets,
	); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(
		w,
		"- Full packet average target: %d tokens\n- Full packet average used: %d tokens\n- Full packet average remaining: %d tokens\n- Full packet average omitted candidates: %.2f\n- Fresh packets under pressure: %d\n- Mandatory over target: %d\n- Pressure rate: %.1f%%\n\n",
		budget.AverageFullTarget,
		budget.AverageFullUsed,
		budget.AverageFullRemaining,
		budget.AverageFullOmitted,
		budget.FreshPacketsUnderPressure,
		budget.MandatoryOverTarget,
		budget.PressureRate*100,
	); err != nil {
		return err
	}
	return nil
}

func renderOutcomeSummary(w io.Writer, outcomes PacketOutcomeSummary) error {
	if _, err := io.WriteString(w, "## Outcomes\n\n"); err != nil {
		return err
	}
	_, err := fmt.Fprintf(
		w,
		"- Packets with expansions: %d\n- Expansion events: %d\n- Packets with successful verification: %d\n- Successful verification events: %d\n- Failed verification events: %d\n- Packets with durable updates: %d\n- Durable update events: %d\n- Successful session closes: %d\n\n",
		outcomes.PacketsWithExpansions,
		outcomes.ExpansionEvents,
		outcomes.PacketsWithSuccessfulVerification,
		outcomes.SuccessfulVerificationEvents,
		outcomes.FailedVerificationEvents,
		outcomes.PacketsWithDurableUpdates,
		outcomes.DurableUpdateEvents,
		outcomes.SuccessfulSessionCloses,
	)
	return err
}

func renderStringList(w io.Writer, title string, items []string) error {
	if _, err := fmt.Fprintf(w, "## %s\n\n", title); err != nil {
		return err
	}
	if len(items) == 0 {
		_, err := io.WriteString(w, "- None.\n\n")
		return err
	}
	for _, item := range items {
		if _, err := fmt.Fprintf(w, "- %s\n", item); err != nil {
			return err
		}
	}
	_, err := io.WriteString(w, "\n")
	return err
}

func renderUtilityGroup(w io.Writer, title string, items []UtilityItemStat) error {
	if _, err := fmt.Fprintf(w, "## %s\n\n", title); err != nil {
		return err
	}
	if len(items) == 0 {
		_, err := io.WriteString(w, "- None recorded yet.\n\n")
		return err
	}
	for _, item := range items {
		if _, err := fmt.Fprintf(
			w,
			"- `%s` [%s] (`%s`): likely_utility=%s include=%d expand=%d verify_success=%d durable_updates=%d\n",
			item.ItemID,
			item.Section,
			anchorLabel(item.Anchor),
			item.LikelyUtility,
			item.IncludeCount,
			item.ExpandCount,
			item.SuccessfulVerificationCount,
			item.DurableUpdateCount,
		); err != nil {
			return err
		}
		for _, reason := range item.Reasons {
			if _, err := fmt.Fprintf(w, "  Evidence: %s\n", reason); err != nil {
				return err
			}
		}
	}
	_, err := io.WriteString(w, "\n")
	return err
}

func renderFreshPacketPressure(w io.Writer, pressure FreshPacketPressure) error {
	if _, err := io.WriteString(w, "## Fresh Packet Pressure\n\n"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(
		w,
		"- Fresh packets analyzed: %d\n- Fresh packets under budget pressure: %d\n- Mandatory over target: %d\n- Pressure rate: %.1f%%\n\n",
		pressure.FreshPacketsAnalyzed,
		pressure.FreshPacketsUnderPressure,
		pressure.MandatoryOverTarget,
		pressure.PressureRate*100,
	); err != nil {
		return err
	}
	return nil
}

func renderOmittedDocs(w io.Writer, title string, items []OmittedDocStat) error {
	if _, err := fmt.Fprintf(w, "## %s\n\n", title); err != nil {
		return err
	}
	if len(items) == 0 {
		_, err := io.WriteString(w, "- None recorded yet.\n\n")
		return err
	}
	for _, item := range items {
		if _, err := fmt.Fprintf(w, "- `%s`: omitted in %d fresh packet(s)\n", item.Path, item.OmittedPackets); err != nil {
			return err
		}
	}
	_, err := io.WriteString(w, "\n")
	return err
}

func anchorLabel(anchor projectcontext.ContextAnchor) string {
	if anchor.Section == "" {
		return anchor.Path
	}
	return anchor.Path + "#" + anchor.Section
}

const timeLayout = "2006-01-02 15:04:05Z07:00"
