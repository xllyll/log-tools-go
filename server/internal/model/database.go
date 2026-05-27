package model

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
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

CREATE TABLE IF NOT EXISTS scene_library (
    id TEXT PRIMARY KEY,
    device_id TEXT NOT NULL,
    title TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    config JSONB NOT NULL,
    module_count INTEGER NOT NULL DEFAULT 0,
    scene_count INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(device_id, title)
);
CREATE INDEX IF NOT EXISTS idx_scene_library_updated ON scene_library(updated_at DESC);

ALTER TABLE log_files ADD COLUMN IF NOT EXISTS progress INTEGER NOT NULL DEFAULT 0;
ALTER TABLE log_files ADD COLUMN IF NOT EXISTS parsed_lines INTEGER NOT NULL DEFAULT 0;
ALTER TABLE log_files ADD COLUMN IF NOT EXISTS source_path TEXT NOT NULL DEFAULT '';

ALTER TABLE log_files ADD COLUMN IF NOT EXISTS original_name TEXT NOT NULL DEFAULT '';
ALTER TABLE log_files ADD COLUMN IF NOT EXISTS file_format TEXT NOT NULL DEFAULT '';
ALTER TABLE log_files ADD COLUMN IF NOT EXISTS entry_type TEXT NOT NULL DEFAULT 'file';
ALTER TABLE log_files ADD COLUMN IF NOT EXISTS parent_id TEXT;
`
	_, err := d.pool.Exec(ctx, ddl)
	if err != nil {
		return err
	}
	_, _ = d.pool.Exec(ctx, `
UPDATE log_files SET
  original_name = CASE WHEN original_name = '' THEN name ELSE original_name END,
  file_format = CASE WHEN file_format = '' THEN lower(substring(name from '\.[^.]*$')) ELSE file_format END,
  entry_type = CASE WHEN entry_type = '' OR entry_type IS NULL THEN 'file' ELSE entry_type END
WHERE original_name = '' OR file_format = '' OR entry_type = '' OR entry_type IS NULL`)
	if err := d.migrateUnifiedTree(ctx); err != nil {
		return err
	}
	_, _ = d.pool.Exec(ctx, `
CREATE UNIQUE INDEX IF NOT EXISTS idx_log_files_folder_unique
ON log_files (device_id, name, parent_id) WHERE entry_type = 'folder'`)
	return nil
}

func (d *Database) migrateUnifiedTree(ctx context.Context) error {
	var hasFoldersTable bool
	_ = d.pool.QueryRow(ctx, `
SELECT EXISTS (
  SELECT 1 FROM information_schema.tables
  WHERE table_schema = 'public' AND table_name = 'log_folders'
)`).Scan(&hasFoldersTable)

	if hasFoldersTable {
		_, _ = d.pool.Exec(ctx, `
INSERT INTO log_files (id, device_id, name, original_name, entry_type, parent_id, size, upload_at, status, status_msg)
SELECT id, device_id, name, name, 'folder', parent_folder_id, 0, created_at, 'folder', ''
FROM log_folders
ON CONFLICT (id) DO NOTHING`)
		_, _ = d.pool.Exec(ctx, `
UPDATE log_files f SET parent_id = lf.parent_folder_id
FROM log_folders lf
WHERE f.id = lf.id AND f.entry_type = 'file' AND (f.parent_id IS NULL OR f.parent_id = '')`)
		_, _ = d.pool.Exec(ctx, `ALTER TABLE log_files DROP CONSTRAINT IF EXISTS log_files_parent_folder_id_fkey`)
		_, _ = d.pool.Exec(ctx, `DROP TABLE IF EXISTS log_folders CASCADE`)
	}

	var hasLegacyCol bool
	_ = d.pool.QueryRow(ctx, `
SELECT EXISTS (
  SELECT 1 FROM information_schema.columns
  WHERE table_schema = 'public' AND table_name = 'log_files' AND column_name = 'parent_folder_id'
)`).Scan(&hasLegacyCol)
	if hasLegacyCol {
		_, _ = d.pool.Exec(ctx, `
UPDATE log_files SET parent_id = parent_folder_id
WHERE (parent_id IS NULL OR parent_id = '') AND parent_folder_id IS NOT NULL AND parent_folder_id <> ''`)
		_, _ = d.pool.Exec(ctx, `ALTER TABLE log_files DROP CONSTRAINT IF EXISTS log_files_parent_folder_id_fkey`)
		_, _ = d.pool.Exec(ctx, `ALTER TABLE log_files DROP COLUMN IF EXISTS parent_folder_id`)
	}

	_, _ = d.pool.Exec(ctx, `ALTER TABLE log_files DROP CONSTRAINT IF EXISTS log_files_parent_id_fkey`)
	_, err := d.pool.Exec(ctx, `
ALTER TABLE log_files
ADD CONSTRAINT log_files_parent_id_fkey
FOREIGN KEY (parent_id) REFERENCES log_files(id) ON DELETE CASCADE`)
	if err != nil && !strings.Contains(err.Error(), "already exists") {
		return err
	}
	return nil
}

const logFileSelectCols = `id, device_id, COALESCE(entry_type,'file'), name, COALESCE(original_name,''), COALESCE(file_format,''),
COALESCE(parent_id,''), size, upload_at, total_entries, parsed_lines, progress, status, COALESCE(status_msg,''), COALESCE(source_path,'')`

func scanLogFile(scanner interface {
	Scan(dest ...any) error
}) (LogFile, error) {
	var f LogFile
	err := scanner.Scan(
		&f.ID, &f.DeviceID, &f.EntryType, &f.Name, &f.OriginalName, &f.FileFormat, &f.ParentID,
		&f.Size, &f.UploadAt, &f.Total, &f.ParsedLines, &f.Progress, &f.Status, &f.StatusMsg, &f.SourcePath,
	)
	if f.EntryType == "" {
		f.EntryType = EntryTypeFile
	}
	if f.OriginalName == "" {
		f.OriginalName = InferOriginalFromStorageName(f.Name)
	}
	if f.FileFormat == "" {
		f.FileFormat = FileFormatFromName(f.OriginalName)
	}
	return f, err
}

func (d *Database) EnsureFolderChain(ctx context.Context, deviceID string, dirParts []string) (string, error) {
	var parentID *string
	for _, part := range dirParts {
		part = strings.TrimSpace(part)
		if part == "" || part == "." {
			continue
		}
		// 防止 Windows 路径整段写入 name（如 foo\bar\baz）
		for _, seg := range splitPathSegmentsForDB(part) {
			id, err := d.ensureFolderEntry(ctx, deviceID, seg, parentID)
			if err != nil {
				return "", err
			}
			parentID = &id
		}
		continue
	}
	if parentID == nil {
		return "", nil
	}
	return *parentID, nil
}

func splitPathSegmentsForDB(part string) []string {
	part = strings.TrimSpace(filepath.ToSlash(part))
	if part == "" || part == "." {
		return nil
	}
	var segs []string
	for _, s := range strings.Split(part, "/") {
		s = strings.TrimSpace(s)
		if s != "" && s != "." {
			segs = append(segs, s)
		}
	}
	if len(segs) > 0 {
		return segs
	}
	return []string{part}
}

func (d *Database) ensureFolderEntry(ctx context.Context, deviceID, name string, parentID *string) (string, error) {
	var parentVal any
	if parentID != nil && *parentID != "" {
		parentVal = *parentID
	}
	var existing string
	err := d.pool.QueryRow(ctx, `
SELECT id FROM log_files
WHERE device_id=$1 AND name=$2 AND entry_type='folder' AND parent_id IS NOT DISTINCT FROM $3`,
		deviceID, name, parentVal).Scan(&existing)
	if err == nil {
		return existing, nil
	}
	id := uuid.NewString()
	_, err = d.pool.Exec(ctx, `
INSERT INTO log_files (id, device_id, name, original_name, entry_type, parent_id, size, upload_at, status, status_msg)
VALUES ($1,$2,$3,$3,'folder',$4,0,NOW(),'folder','')`,
		id, deviceID, name, parentVal)
	return id, err
}

func (d *Database) SaveLogFile(ctx context.Context, f *LogFile) error {
	if f.EntryType == "" {
		f.EntryType = EntryTypeFile
	}
	var parentID any
	if f.ParentID != "" {
		parentID = f.ParentID
	}
	if f.OriginalName == "" {
		f.OriginalName = InferOriginalFromStorageName(f.Name)
	}
	if f.FileFormat == "" {
		f.FileFormat = FileFormatFromName(f.OriginalName)
	}
	_, err := d.pool.Exec(ctx, `
INSERT INTO log_files (id, device_id, entry_type, name, original_name, file_format, parent_id, size, upload_at, total_entries, status, status_msg, source_path, parsed_lines, progress)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)
ON CONFLICT (id) DO UPDATE SET
  name=EXCLUDED.name,
  original_name=EXCLUDED.original_name,
  file_format=EXCLUDED.file_format,
  entry_type=EXCLUDED.entry_type,
  parent_id=EXCLUDED.parent_id,
  size=EXCLUDED.size,
  total_entries=EXCLUDED.total_entries,
  status=EXCLUDED.status,
  status_msg=EXCLUDED.status_msg,
  source_path=CASE WHEN EXCLUDED.source_path <> '' THEN EXCLUDED.source_path ELSE log_files.source_path END,
  parsed_lines=EXCLUDED.parsed_lines,
  progress=EXCLUDED.progress`,
		f.ID, f.DeviceID, f.EntryType, f.Name, f.OriginalName, f.FileFormat, parentID,
		f.Size, f.UploadAt, f.Total, f.Status, f.StatusMsg, f.SourcePath, f.ParsedLines, f.Progress)
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
	conn, err := d.pool.Acquire(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()

	_, err = conn.Conn().CopyFrom(
		ctx,
		pgx.Identifier{"log_entries"},
		[]string{"id", "file_id", "device_id", "log_time", "content", "line_number", "level", "module", "message"},
		pgx.CopyFromSlice(len(entries), func(i int) ([]any, error) {
			e := entries[i]
			content := xencoding.SanitizeForDB(e.Content)
			message := xencoding.SanitizeForDB(e.Message)
			module := xencoding.SanitizeForDB(e.Module)
			if message == "" {
				message = content
			}
			return []any{e.ID, fileID, deviceID, e.LogTime, content, e.Line, e.Level, module, message}, nil
		}),
	)
	return err
}

func (d *Database) DeleteEntriesByFile(ctx context.Context, fileID string) error {
	_, err := d.pool.Exec(ctx, `DELETE FROM log_entries WHERE file_id=$1`, fileID)
	return err
}

func (d *Database) ResolveFolderPath(ctx context.Context, folderID string) ([]string, error) {
	if folderID == "" {
		return nil, nil
	}
	var parts []string
	cur := folderID
	for cur != "" {
		var name, parent, entryType string
		err := d.pool.QueryRow(ctx, `
SELECT name, COALESCE(parent_id,''), COALESCE(entry_type,'') FROM log_files WHERE id=$1`, cur).
			Scan(&name, &parent, &entryType)
		if err != nil || entryType != EntryTypeFolder {
			return parts, nil
		}
		parts = append([]string{name}, parts...)
		cur = parent
	}
	return parts, nil
}

func (d *Database) GetLogFiles(ctx context.Context, deviceID string) ([]LogFile, error) {
	rows, err := d.pool.Query(ctx, `
SELECT `+logFileSelectCols+`
FROM log_files WHERE device_id=$1 AND entry_type='file' ORDER BY upload_at DESC`, deviceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var files []LogFile
	for rows.Next() {
		f, err := scanLogFile(rows)
		if err != nil {
			return nil, err
		}
		files = append(files, f)
	}
	return files, rows.Err()
}

func (d *Database) ListDeviceFiles(ctx context.Context, deviceID string) (*FileListData, error) {
	rows, err := d.pool.Query(ctx, `
SELECT `+logFileSelectCols+`
FROM log_files WHERE device_id=$1 ORDER BY entry_type DESC, name`, deviceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []LogFile
	for rows.Next() {
		item, err := scanLogFile(rows)
		if err != nil {
			return nil, err
		}
		if item.IsFile() && item.ParentID != "" {
			item.FolderPath, _ = d.ResolveFolderPath(ctx, item.ParentID)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return &FileListData{Items: items}, nil
}

func (d *Database) GetLogItem(ctx context.Context, deviceID, id string) (*LogFile, error) {
	row := d.pool.QueryRow(ctx, `
SELECT `+logFileSelectCols+`
FROM log_files WHERE id=$1 AND device_id=$2`, id, deviceID)
	f, err := scanLogFile(row)
	if err != nil {
		return nil, err
	}
	return &f, nil
}

func (d *Database) GetLogFile(ctx context.Context, deviceID, fileID string) (*LogFile, error) {
	row := d.pool.QueryRow(ctx, `
SELECT `+logFileSelectCols+`
FROM log_files WHERE id=$1 AND device_id=$2 AND entry_type='file'`, fileID, deviceID)
	f, err := scanLogFile(row)
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

func (d *Database) DeleteLogFileByID(ctx context.Context, fileID string) error {
	_, err := d.pool.Exec(ctx, `DELETE FROM log_files WHERE id=$1`, fileID)
	return err
}

func (d *Database) ListLogFilesBefore(ctx context.Context, before time.Time) ([]LogFile, error) {
	rows, err := d.pool.Query(ctx, `
SELECT `+logFileSelectCols+`
FROM log_files WHERE upload_at < $1 AND entry_type='file'`, before)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var files []LogFile
	for rows.Next() {
		f, err := scanLogFile(rows)
		if err != nil {
			return nil, err
		}
		files = append(files, f)
	}
	return files, rows.Err()
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

const SharedSceneDeviceID = "__shared__"
const SharedSceneConfigName = "default"

func (d *Database) SaveSharedSceneConfig(ctx context.Context, raw []byte) error {
	return d.SaveSceneConfig(ctx, SharedSceneDeviceID, SharedSceneConfigName, raw)
}

func (d *Database) GetSharedSceneConfig(ctx context.Context) ([]byte, time.Time, error) {
	var raw []byte
	var updated time.Time
	err := d.pool.QueryRow(ctx, `
SELECT config, updated_at FROM scene_configs WHERE device_id=$1 AND name=$2`,
		SharedSceneDeviceID, SharedSceneConfigName).Scan(&raw, &updated)
	if err != nil {
		return nil, time.Time{}, err
	}
	return raw, updated, nil
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

func countSceneConfigStats(cfg SceneConfig) (moduleCount, sceneCount int) {
	moduleCount = len(cfg.Modules)
	for _, m := range cfg.Modules {
		sceneCount += len(m.Scenes)
	}
	return moduleCount, sceneCount
}

func (d *Database) PublishSceneLibrary(ctx context.Context, deviceID, title, description string, cfg SceneConfig) (string, error) {
	raw, err := json.Marshal(cfg)
	if err != nil {
		return "", err
	}
	mc, sc := countSceneConfigStats(cfg)
	id := uuid.NewString()
	err = d.pool.QueryRow(ctx, `
INSERT INTO scene_library (id, device_id, title, description, config, module_count, scene_count, updated_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,NOW())
ON CONFLICT (device_id, title) DO UPDATE SET
  description=EXCLUDED.description,
  config=EXCLUDED.config,
  module_count=EXCLUDED.module_count,
  scene_count=EXCLUDED.scene_count,
  updated_at=NOW()
RETURNING id`,
		id, deviceID, title, description, raw, mc, sc).Scan(&id)
	return id, err
}

func (d *Database) ListSceneLibrary(ctx context.Context, viewerDeviceID string, limit int) ([]SceneLibraryItem, error) {
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	rows, err := d.pool.Query(ctx, `
SELECT id, device_id, title, COALESCE(description,''), module_count, scene_count, updated_at
FROM scene_library
ORDER BY updated_at DESC
LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []SceneLibraryItem
	for rows.Next() {
		var item SceneLibraryItem
		if err := rows.Scan(&item.ID, &item.DeviceID, &item.Title, &item.Description, &item.ModuleCount, &item.SceneCount, &item.UpdatedAt); err != nil {
			return nil, err
		}
		item.IsMine = item.DeviceID == viewerDeviceID
		list = append(list, item)
	}
	return list, rows.Err()
}

func (d *Database) GetSceneLibrary(ctx context.Context, id string) (*SceneLibraryDetail, error) {
	var item SceneLibraryDetail
	var raw []byte
	err := d.pool.QueryRow(ctx, `
SELECT id, device_id, title, COALESCE(description,''), module_count, scene_count, updated_at, config
FROM scene_library WHERE id=$1`, id).
		Scan(&item.ID, &item.DeviceID, &item.Title, &item.Description, &item.ModuleCount, &item.SceneCount, &item.UpdatedAt, &raw)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(raw, &item.Config); err != nil {
		return nil, err
	}
	return &item, nil
}

func (d *Database) DeleteSceneLibrary(ctx context.Context, deviceID, id string) error {
	tag, err := d.pool.Exec(ctx, `DELETE FROM scene_library WHERE id=$1 AND device_id=$2`, id, deviceID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("not found or forbidden")
	}
	return nil
}
