# Session Changes

## Change Detection Notes

- Attempted `git status --short` and `git diff --name-only` from project root.
- The workspace is not a git repository (`.git` missing), so git-based staged/unstaged diff inspection is unavailable.
- As fallback, source code (`*.go`) and markdown (`*.md`) files were inspected directly in the workspace, and recently touched files were identified by file scan.

## Summary Of Work In This Session

- Default app entry was changed from direct `sync` start to an interactive launcher.
- The launcher now allows selecting one mode among `sync`, `restore`, and `view`.
- A new `view` command was added to list backups in read-only mode.
- Backup strategy was consolidated to one tool-level snapshot archive (`{tool}_snapshot_*.tar.gz`) instead of multiple per-asset backups.
- Interactive backup management was improved with timestamp display and delete workflow.
- Backup retention and dedup behavior were kept with hash-based skip and rolling retention.
- Documentation and tests were updated to match the new default mode and backup behavior.

## Changed Files And Brief Summary

- `cmd/root.go`: default execution path switched to interactive launcher; `view` command registered; usage text updated.
- `cmd/interactive.go`: added main mode selector (`sync/restore/view`) and tool selector; kept target/backup selection helpers; timestamps shown in backup selections.
- `cmd/view.go`: added new mode to inspect backup history for a selected tool.
- `cmd/backup.go`: backup create/list/delete flows maintained; create path now generates one snapshot archive per tool.
- `cmd/restore.go`: interactive backup selection path retained for restore.
- `cmd/sync.go`: sync command remains selectable from launcher and continues interactive/non-interactive operation.
- `cmd/interactive_test.go`: tests added/updated for mode selector, tool selector, backup selector output.
- `cmd/view_test.go`: added test coverage for view mode output and timestamp formatting.

- `internal/backup/backup.go`: added tool snapshot backup APIs, single-archive packaging, snapshot restore path handling (`main`, `commands`, `skills`, `agents`), and backup delete with metadata cleanup.

- `internal/backup/backup_test.go`: added/updated tests for snapshot create/skip/restore and delete behavior.
- `internal/sync/engine.go`: pre-sync backup switched to one snapshot backup per target tool.
- `internal/sync/engine_test.go`: expected backup count updated for snapshot model.

- `README.md`: usage and backup policy updated for interactive launcher default and snapshot archive model.

## Documentation Impact Analysis

- Existing root `README.md` was affected and updated to explain launcher behavior and snapshot backup semantics.
- No existing `./doc` documents were present, so new documentation files were required.
- No additional specialized docs were needed; session-specific updates were consolidated into this file to avoid unnecessary document sprawl.
