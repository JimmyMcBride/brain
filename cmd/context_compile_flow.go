package cmd

import (
	"context"
	"errors"
	"strings"

	"brain/internal/app"
	"brain/internal/livecontext"
	"brain/internal/projectcontext"
	"brain/internal/search"
	"brain/internal/session"
	"brain/internal/taskcontext"
)

type compilePacketRequest struct {
	ProjectRoot   string
	Task          string
	TaskSource    string
	Budget        string
	Fresh         bool
	ActiveSession *session.ActiveSession
}

func runCompilePacketFlow(ctx context.Context, appCtx *app.App, req compilePacketRequest) (*projectcontext.CompileResponse, error) {
	if appCtx == nil {
		return nil, errors.New("app context is required")
	}
	projectRoot := strings.TrimSpace(req.ProjectRoot)
	resolvedTask := strings.TrimSpace(req.Task)
	taskSource := strings.TrimSpace(req.TaskSource)
	active := req.ActiveSession
	if projectRoot == "" {
		return nil, errors.New("project root is required")
	}
	if resolvedTask == "" {
		return nil, errors.New("task is required")
	}
	if taskSource == "" {
		taskSource = "flag"
	}

	activeTask := ""
	if active != nil {
		activeTask = strings.TrimSpace(active.Task)
	}
	if err := appCtx.SyncIndex(ctx); err != nil {
		return nil, err
	}
	searchResults, err := appCtx.Search.SearchWithOptions(ctx, resolvedTask, 12, search.Options{ActiveTask: activeTask})
	if err != nil {
		return nil, err
	}
	utilitySnapshot, err := appCtx.Session.BuildUtilitySnapshot(projectRoot)
	if err != nil {
		return nil, err
	}
	boundaryGraph, err := appCtx.Structure.BoundaryGraph(ctx)
	if err != nil {
		return nil, err
	}
	livePacket, err := appCtx.Live.Collect(ctx, livecontext.Request{
		ProjectDir:    projectRoot,
		Task:          resolvedTask,
		TaskSource:    taskSource,
		Session:       active,
		BoundaryGraph: boundaryGraph,
	})
	if err != nil {
		return nil, err
	}

	manager := taskcontext.New(appCtx.Context)
	compileRequest := taskcontext.Request{
		ProjectDir:     projectRoot,
		Task:           resolvedTask,
		TaskSource:     taskSource,
		Budget:         req.Budget,
		SearchResults:  searchResults,
		LivePacket:     livePacket,
		BoundaryGraph:  boundaryGraph,
		UtilitySignals: utilitySignalsFromSnapshot(utilitySnapshot),
	}
	fingerprintInputs, err := manager.BuildFingerprintInputs(compileRequest)
	if err != nil {
		return nil, err
	}
	fingerprint := fingerprintInputs.Hash()
	if active != nil && !req.Fresh {
		if reusable := latestMatchingPacketRecord(active.PacketRecords, fingerprint); reusable != nil && reusable.Packet != nil {
			meta := projectcontext.PacketCacheMetadata{
				CacheStatus:        projectcontext.PacketCacheStatusReused,
				Fingerprint:        fingerprint,
				ReusedFrom:         reusable.PacketHash,
				FullPacketIncluded: false,
			}
			if err := appCtx.Session.RecordCompiledPacket(projectRoot, active.ID, reusable.Packet, fingerprintInputs, meta); err != nil {
				return nil, err
			}
			return projectcontext.NewCompileResponse(reusable.Packet, meta), nil
		}
	}

	packet, err := manager.Compile(compileRequest)
	if err != nil {
		return nil, err
	}

	meta := projectcontext.PacketCacheMetadata{
		CacheStatus:        projectcontext.PacketCacheStatusFresh,
		Fingerprint:        fingerprint,
		FullPacketIncluded: true,
	}
	if req.Fresh {
		meta.FallbackReason = "fresh compile requested"
	} else if active == nil {
		meta.FallbackReason = "no active session; emitted a standalone full packet"
	}
	if active != nil && !req.Fresh {
		if previous := latestTaskPacketRecord(active.PacketRecords, packet.Task.Text); previous != nil {
			if previous.Fingerprint == "" {
				meta.FallbackReason = "prior packet lineage unavailable; emitted a standalone full packet"
			} else if previous.Packet != nil {
				meta.InvalidationReasons = fingerprintInputs.InvalidationReasons(previous.FingerprintInputs)
				changedSections, changedItemIDs := taskcontext.PacketDiff(previous.Packet, packet)
				if len(meta.InvalidationReasons) != 0 || len(changedSections) != 0 || len(changedItemIDs) != 0 {
					meta.CacheStatus = projectcontext.PacketCacheStatusDelta
					meta.DeltaFrom = previous.PacketHash
					meta.ChangedSections = changedSections
					meta.ChangedItemIDs = changedItemIDs
					meta.FullPacketIncluded = false
				}
			} else {
				meta.FallbackReason = "prior packet body unavailable; emitted a standalone full packet"
			}
		} else {
			meta.FallbackReason = "no prior session packet available"
		}
	}
	if active != nil {
		if err := appCtx.Session.RecordCompiledPacket(projectRoot, active.ID, packet, fingerprintInputs, meta); err != nil {
			return nil, err
		}
	}
	return projectcontext.NewCompileResponse(packet, meta), nil
}
