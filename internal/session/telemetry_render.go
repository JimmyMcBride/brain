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
		"## Packet\n\n- Hash: `%s`\n- Compiled: %s\n- Task: `%s`\n- Summary: %s\n- Source: `%s`\n- Session: `%s` (%s)\n\n",
		explanation.Packet.PacketHash,
		explanation.Packet.CompiledAt.Format(timeLayout),
		explanation.Packet.TaskText,
		explanation.Packet.TaskSummary,
		explanation.Packet.TaskSource,
		explanation.Packet.SessionID,
		explanation.Packet.SessionStatus,
	); err != nil {
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

func anchorLabel(anchor projectcontext.ContextAnchor) string {
	if anchor.Section == "" {
		return anchor.Path
	}
	return anchor.Path + "#" + anchor.Section
}

const timeLayout = "2006-01-02 15:04:05Z07:00"
