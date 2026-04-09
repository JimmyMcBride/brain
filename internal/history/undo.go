package history

import (
	"errors"
	"fmt"
	"os"

	"brain/internal/backup"
	"brain/internal/vault"
)

type Undoer struct {
	Logger  *Logger
	Backups *backup.Manager
	Vault   *vault.Service
}

func NewUndoer(logger *Logger, backups *backup.Manager, vaultSvc *vault.Service) *Undoer {
	return &Undoer{
		Logger:  logger,
		Backups: backups,
		Vault:   vaultSvc,
	}
}

func (u *Undoer) Undo() (*Entry, error) {
	entries, err := u.Logger.All()
	if err != nil {
		return nil, err
	}
	undone := map[string]struct{}{}
	for _, entry := range entries {
		if entry.Operation == "undo" && entry.UndoOf != "" {
			undone[entry.UndoOf] = struct{}{}
		}
	}

	var target *Entry
	for i := len(entries) - 1; i >= 0; i-- {
		entry := entries[i]
		if entry.Operation == "undo" {
			continue
		}
		if _, ok := undone[entry.ID]; ok {
			continue
		}
		target = &entry
		break
	}
	if target == nil {
		return nil, errors.New("nothing to undo")
	}

	switch target.Operation {
	case "create":
		if err := os.Remove(u.Vault.Abs(target.File)); err != nil && !errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("remove created file: %w", err)
		}
	case "update", "publish", "seed":
		if target.BackupPath == "" {
			return nil, fmt.Errorf("history entry %s has no backup", target.ID)
		}
		if err := u.Backups.Restore(target.BackupPath, u.Vault.Abs(target.File)); err != nil {
			return nil, err
		}
	case "move", "rename", "archive":
		if target.BackupPath == "" {
			return nil, fmt.Errorf("history entry %s has no backup", target.ID)
		}
		if err := u.Backups.Restore(target.BackupPath, u.Vault.Abs(target.File)); err != nil {
			return nil, err
		}
		if target.Target != "" {
			if err := os.Remove(u.Vault.Abs(target.Target)); err != nil && !errors.Is(err, os.ErrNotExist) {
				return nil, fmt.Errorf("remove moved file: %w", err)
			}
		}
	default:
		if target.BackupPath == "" {
			return nil, fmt.Errorf("cannot undo operation %s", target.Operation)
		}
		if err := u.Backups.Restore(target.BackupPath, u.Vault.Abs(target.File)); err != nil {
			return nil, err
		}
	}

	if err := u.Logger.Append(Entry{
		Operation: "undo",
		File:      target.File,
		Target:    target.Target,
		Summary:   "reverted " + target.Operation,
		UndoOf:    target.ID,
	}); err != nil {
		return nil, err
	}
	return target, nil
}
