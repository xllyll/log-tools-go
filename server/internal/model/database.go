package model

import (
	"context"
	"fmt"
	"strings"
	"time"

	"log-tools/server/internal/config"

	"log-tools/server/pkg/xencoding"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Database struct {
	pool *pgxpool.Pool
}

func NewDatabase(cfg *config.Config) (*Database, error) {
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, cfg.Database.DSN())
	if err != nil {
		return nil, fmt.Errorf("connect postgres: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("ping postgres: %w", err)
	}
	db := &Database{pool: pool}
	if err := db.initTables(ctx); err != nil {
		return nil, err
	}
	return db, nil
}

func (d *Database) Close() {
	d.pool.Close()
}

func (d *Database) initTables(ctx context.Context) error {
	ddl := `
CREATE TABLE IF NOT EXISTS log_files (
    id TEXT PRIMARY KEY,
    device_id TEXT NOT NULL,
    name TEXT NOT NULL,
    size BIGINT NOT NULL DEFAULT 0,
    upload_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    total_entries INTEGER NOT NULL DEFAULT 0,
    status TEXT NOT NULL DEFAULT 'parsing',
    status_msg TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_log_files_device ON log_files(device_id);

CREATE TABLE IF NOT EXISTS log_entries (
    id TEXT PRIMARY KEY,
    file_id TEXT NOT NULL REFERENCES log_files(id) ON DELETE CASCADE,
    device_id TEXT NOT NULL,
    log_time TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    content TEXT NOT NULL,
    line_number INTEGER NOT NULL,
    level TEXT NOT NULL DEFAULT 'INFO',
    module TEXT,
    message TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_log_entries_file_line ON log_entries(file_id, line_number);
CREATE INDEX IF NOT EXISTS idx_log_entries_device ON log_entries(device_id);
CREATE INDEX IF NOT EXISTS idx_log_entries_content ON log_entries USING gin (to_tsvector('simple', content));

CREATE TABLE IF NOT EXISTS scene_configs (
    id TEXT PRIMARY KEY,
    device_id TEXT NOT NULL,
    name TEXT NOT NULL,
    config JSONB NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(device_id, name)
);

ALTER TABLE log_files ADD COLUMN IF NOT EXISTS progress INTEGER NOT NULL DEFAULT 0;
ALTER TABLE log_files ADD COLUMN IF NOT EXISTS parsed_lines INTEGER NOT NULL DEFAULT 0;
ALTER TABLE log_files ADD COLUMN IF NOT EXISTS source_path TEXT NOT NULL DEFAULT '';
`
	_, err := d.pool.Exec(ctx, ddl)
	return err
}

func (d *Database) SaveLogFile(ctx context.Context, f *LogFile) error {
	_, err := d.pool.Exec(ctx, `
INSERT INTO log_files (id, device_id, name, size, upload_at, total_entries, status, status_msg, source_path, parsed_lines, progress)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
ON CONFLICT (id) DO UPDATE SET
  name=EXCLUDED.name,
  size=EXCLUDED.size,
  total_entries=EXCLUDED.total_entries,
  status=EXCLUDED.status,
  status_msg=EXCLUDED.status_msg,
  source_path=CASE WHEN EXCLUDED.source_path <> '' THEN EXCLUDED.source_path ELSE log_files.source_path END,
  parsed_lines=EXCLUDED.parsed_lines,
  progress=EXCLUDED.progress`,
		f.ID, f.DeviceID, f.Name, f.Size, f.UploadAt, f.Total, f.Status, f.StatusMsg, f.SourcePath, f.ParsedLines, f.Progress)
	return err
}

func (d *Database) UpdateFileStatus(ctx context.Context, fileID, status, msg string, total int) error {
	return d.UpdateFileProgress(ctx, fileID, status, msg, total, total, progressForStatus(status, total))
}

func (d *Database) UpdateFileProgress(ctx context.Context, fileID, status, msg string, parsedLines, total, progress int) error {
	if progress < 0 {
		progress = 0
	}
	if progress > 100 {
		progress = 100
	}
	_, err := d.pool.Exec(ctx, `
UPDATE log_files SET status=$2, status_msg=$3, total_entries=$4, parsed_lines=$5, progress=$6 WHERE id=$1`,
		fileID, status, msg, total, parsedLines, progress)
	return err
}

func progressForStatus(status string, total int) int {
	switch status {
	case "ready":
		return 100
	case "failed":
		return 0
	default:
		if total > 0 {
			return 50
		}
		return 0
	}
}

func (d *Database) BatchInsertEntries(ctx context.Context, deviceID, fileID string, entries []LogEntry) error {
	if len(entries) == 0 {
		return nil
	}
	batch := &pgx.Batch{}
	for _, e := range entries {
		content := xencoding.SanitizeForDB(e.Content)
		message := xencoding.SanitizeForDB(e.Message)
		module := xencoding.SanitizeForDB(e.Module)
		if message == "" {
			message = content
		}
		batch.Queue(`
INSERT INTO log_entries (id, file_id, device_id, log_time, content, line_number, level, module, message)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
ON CONFLICT (id) DO NOTHING`,
			e.ID, fileID, deviceID, e.LogTime, content, e.Line, e.Level, module, message)
	}
	br := d.pool.SendBatch(ctx, batch)
	defer br.Close()
	for range entries {
		if _, err := br.Exec(); err != nil {
			return err
		}
	}
	return nil
}

func (d *Database) DeleteEntriesByFile(ctx context.Context, fileID string) error {
	_, err := d.pool.Exec(ctx, `DELETE FROM log_entries WHERE file_id=$1`, fileID)
	return err
}

func (d *Database) GetLogFiles(ctx context.Context, deviceID string) ([]LogFile, error) {
	rows, err := d.pool.Query(ctx, `
SELECT id, device_id, name, size, upload_at, total_entries, parsed_lines, progress, status, COALESCE(status_msg,''), COALESCE(source_path,'')
FROM log_files WHERE device_id=$1 ORDER BY upload_at DESC`, deviceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var files []LogFile
	for rows.Next() {
		var f LogFile
		if err := rows.Scan(&f.ID, &f.DeviceID, &f.Name, &f.Size, &f.UploadAt, &f.Total, &f.ParsedLines, &f.Progress, &f.Status, &f.StatusMsg, &f.SourcePath); err != nil {
			return nil, err
		}
		files = append(files, f)
	}
	return files, rows.Err()
}

func (d *Database) GetLogFile(ctx context.Context, deviceID, fileID string) (*LogFile, error) {
	var f LogFile
	err := d.pool.QueryRow(ctx, `
SELECT id, device_id, name, size, upload_at, total_entries, parsed_lines, progress, status, COALESCE(status_msg,''), COALESCE(source_path,'')
FROM log_files WHERE id=$1 AND device_id=$2`, fileID, deviceID).
		Scan(&f.ID, &f.DeviceID, &f.Name, &f.Size, &f.UploadAt, &f.Total, &f.ParsedLines, &f.Progress, &f.Status, &f.StatusMsg, &f.SourcePath)
	if err != nil {
		return nil, err
	}
	return &f, nil
}

func (d *Database) DeleteLogFile(ctx context.Context, deviceID, fileID string) error {
	tag, err := d.pool.Exec(ctx, `DELETE FROM log_files WHERE id=$1 AND device_id=$2`, fileID, deviceID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("file not found")
	}
	return nil
}

func (d *Database) GetLogEntries(ctx context.Context, filter LogFilter) ([]LogEntry, error) {
	var b strings.Builder
	args := make([]any, 0)
	argN := 1

	b.WriteString(`SELECT id, file_id, log_time, content, line_number, level, COALESCE(module,''), COALESCE(message,'')
FROM log_entries WHERE device_id=$`)
	args = append(args, filter.DeviceID)
	argN++
	b.WriteString(fmt.Sprintf("%d", 1))

	if filter.FileID != "" {
		fmt.Fprintf(&b, " AND file_id=$%d", argN)
		args = append(args, filter.FileID)
		argN++
	}
	if len(filter.FileIDs) > 0 {
		fmt.Fprintf(&b, " AND file_id = ANY($%d)", argN)
		args = append(args, filter.FileIDs)
		argN++
	}

	for _, kw := range filter.Keywords {
		if filter.UseRegex {
			fmt.Fprintf(&b, " AND content ~ $%d", argN)
		} else {
			fmt.Fprintf(&b, " AND content ILIKE $%d", argN)
		}
		if filter.UseRegex {
			args = append(args, kw)
		} else {
			args = append(args, "%"+kw+"%")
		}
		argN++
	}

	if len(filter.SceneKeywords) > 0 {
		b.WriteString(" AND (")
		for i, kw := range filter.SceneKeywords {
			if i > 0 {
				b.WriteString(" OR ")
			}
			fmt.Fprintf(&b, "content ILIKE $%d", argN)
			args = append(args, "%"+kw+"%")
			argN++
		}
		b.WriteString(")")
	}

	b.WriteString(" ORDER BY line_number ASC")

	if filter.Limit > 0 {
		fmt.Fprintf(&b, " LIMIT $%d", argN)
		args = append(args, filter.Limit)
		argN++
	}
	if filter.Offset > 0 {
		fmt.Fprintf(&b, " OFFSET $%d", argN)
		args = append(args, filter.Offset)
	}

	rows, err := d.pool.Query(ctx, b.String(), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []LogEntry
	for rows.Next() {
		var e LogEntry
		if err := rows.Scan(&e.ID, &e.FileID, &e.LogTime, &e.Content, &e.Line, &e.Level, &e.Module, &e.Message); err != nil {
			return nil, err
		}
		e.Message = e.Content
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

func (d *Database) GetContextEntries(ctx context.Context, deviceID, fileID string, line, before, after int) ([]LogEntry, error) {
	if before <= 0 {
		before = 10
	}
	if after <= 0 {
		after = 10
	}
	rows, err := d.pool.Query(ctx, `
SELECT id, file_id, log_time, content, line_number, level, COALESCE(module,''), COALESCE(message,'')
FROM log_entries
WHERE device_id=$1 AND file_id=$2 AND line_number BETWEEN $3 AND $4
ORDER BY line_number ASC`,
		deviceID, fileID, line-before, line+after)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var entries []LogEntry
	for rows.Next() {
		var e LogEntry
		if err := rows.Scan(&e.ID, &e.FileID, &e.LogTime, &e.Content, &e.Line, &e.Level, &e.Module, &e.Message); err != nil {
			return nil, err
		}
		e.Message = e.Content
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

func (d *Database) CountEntries(ctx context.Context, deviceID, fileID string, filter LogFilter) (int, error) {
	filter.Limit = 0
	filter.Offset = 0
	entries, err := d.GetLogEntries(ctx, LogFilter{
		DeviceID:      deviceID,
		FileID:        fileID,
		FileIDs:       filter.FileIDs,
		Keywords:      filter.Keywords,
		SceneKeywords: filter.SceneKeywords,
		UseRegex:      filter.UseRegex,
	})
	if err != nil {
		return 0, err
	}
	return len(entries), nil
}

func (d *Database) SaveSceneConfig(ctx context.Context, deviceID, name string, raw []byte) error {
	id := uuid.NewString()
	_, err := d.pool.Exec(ctx, `
INSERT INTO scene_configs (id, device_id, name, config, updated_at)
VALUES ($1,$2,$3,$4,NOW())
ON CONFLICT (device_id, name) DO UPDATE SET config=EXCLUDED.config, updated_at=NOW()`,
		id, deviceID, name, raw)
	return err
}

func (d *Database) ListSceneConfigs(ctx context.Context, deviceID string) ([]map[string]any, error) {
	rows, err := d.pool.Query(ctx, `
SELECT name, config, updated_at FROM scene_configs WHERE device_id=$1 ORDER BY updated_at DESC`, deviceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []map[string]any
	for rows.Next() {
		var name string
		var raw []byte
		var updated time.Time
		if err := rows.Scan(&name, &raw, &updated); err != nil {
			return nil, err
		}
		list = append(list, map[string]any{"name": name, "config": raw, "updated_at": updated})
	}
	return list, rows.Err()
}
