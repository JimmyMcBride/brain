package index

import (
	"context"
	"database/sql"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"brain/internal/embeddings"
	"brain/internal/notes"
	"brain/internal/vault"

	_ "modernc.org/sqlite"
)

type Store struct {
	DB   *sql.DB
	Path string
}

type ChunkRecord struct {
	ChunkID  int64   `json:"chunk_id"`
	NotePath string  `json:"note_path"`
	Heading  string  `json:"heading"`
	Content  string  `json:"content"`
	Snippet  string  `json:"snippet"`
	Score    float64 `json:"score"`
}

type Stats struct {
	Notes      int `json:"notes"`
	Chunks     int `json:"chunks"`
	Embeddings int `json:"embeddings"`
}

var ftsTokenPattern = regexp.MustCompile(`[[:alnum:]_]+`)

func New(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create db parent: %w", err)
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	store := &Store{DB: db, Path: path}
	if err := store.InitSchema(); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *Store) Close() error {
	if s == nil || s.DB == nil {
		return nil
	}
	return s.DB.Close()
}

func (s *Store) InitSchema() error {
	stmts := []string{
		`PRAGMA journal_mode=WAL;`,
		`PRAGMA foreign_keys=ON;`,
		`CREATE TABLE IF NOT EXISTS notes (
			path TEXT PRIMARY KEY,
			title TEXT NOT NULL,
			type TEXT NOT NULL,
			metadata_json TEXT NOT NULL,
			content TEXT NOT NULL,
			modified_at TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS chunks (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			note_path TEXT NOT NULL,
			heading TEXT NOT NULL,
			content TEXT NOT NULL,
			chunk_index INTEGER NOT NULL,
			FOREIGN KEY(note_path) REFERENCES notes(path) ON DELETE CASCADE
		);`,
		`CREATE VIRTUAL TABLE IF NOT EXISTS chunk_fts USING fts5(
			note_path UNINDEXED,
			heading,
			content,
			tokenize='porter unicode61'
		);`,
		`CREATE TABLE IF NOT EXISTS embeddings (
			chunk_id INTEGER PRIMARY KEY,
			provider TEXT NOT NULL,
			model TEXT NOT NULL,
			vector BLOB NOT NULL,
			dims INTEGER NOT NULL,
			updated_at TEXT NOT NULL,
			FOREIGN KEY(chunk_id) REFERENCES chunks(id) ON DELETE CASCADE
		);`,
	}
	for _, stmt := range stmts {
		if _, err := s.DB.Exec(stmt); err != nil {
			return fmt.Errorf("init schema: %w", err)
		}
	}
	return nil
}

func (s *Store) Reindex(ctx context.Context, vaultSvc *vault.Service, provider embeddings.Provider) (Stats, error) {
	files, err := vaultSvc.WalkMarkdownFiles()
	if err != nil {
		return Stats{}, err
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return Stats{}, err
	}
	defer tx.Rollback()

	for _, stmt := range []string{
		`DELETE FROM embeddings;`,
		`DELETE FROM chunk_fts;`,
		`DELETE FROM chunks;`,
		`DELETE FROM notes;`,
	} {
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			return Stats{}, fmt.Errorf("clear index: %w", err)
		}
	}

	insertNote, err := tx.PrepareContext(ctx, `INSERT INTO notes(path, title, type, metadata_json, content, modified_at) VALUES (?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return Stats{}, err
	}
	defer insertNote.Close()
	insertChunk, err := tx.PrepareContext(ctx, `INSERT INTO chunks(note_path, heading, content, chunk_index) VALUES (?, ?, ?, ?)`)
	if err != nil {
		return Stats{}, err
	}
	defer insertChunk.Close()
	insertFTS, err := tx.PrepareContext(ctx, `INSERT INTO chunk_fts(rowid, note_path, heading, content) VALUES (?, ?, ?, ?)`)
	if err != nil {
		return Stats{}, err
	}
	defer insertFTS.Close()
	insertEmbedding, err := tx.PrepareContext(ctx, `INSERT INTO embeddings(chunk_id, provider, model, vector, dims, updated_at) VALUES (?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return Stats{}, err
	}
	defer insertEmbedding.Close()

	stats := Stats{}
	batchTexts := make([]string, 0, 64)
	batchChunkIDs := make([]int64, 0, 64)
	flushEmbeddings := func() error {
		if len(batchTexts) == 0 || provider == nil || provider.Name() == "none" {
			batchTexts = batchTexts[:0]
			batchChunkIDs = batchChunkIDs[:0]
			return nil
		}
		vectors, err := provider.Embed(ctx, batchTexts)
		if err != nil {
			return err
		}
		now := time.Now().UTC().Format(time.RFC3339)
		for i, vec := range vectors {
			if len(vec) == 0 {
				continue
			}
			if _, err := insertEmbedding.ExecContext(ctx, batchChunkIDs[i], provider.Name(), provider.Model(), encodeVector(vec), len(vec), now); err != nil {
				return fmt.Errorf("insert embedding: %w", err)
			}
			stats.Embeddings++
		}
		batchTexts = batchTexts[:0]
		batchChunkIDs = batchChunkIDs[:0]
		return nil
	}

	for _, file := range files {
		raw, err := os.ReadFile(file)
		if err != nil {
			return Stats{}, fmt.Errorf("read note for indexing: %w", err)
		}
		meta, body, err := notes.ParseFrontmatter(string(raw))
		if err != nil {
			return Stats{}, err
		}
		rel, err := vaultSvc.Rel(file)
		if err != nil {
			return Stats{}, err
		}
		title := titleFromMeta(rel, meta)
		noteType := typeFromMeta(rel, meta)
		metaJSON, err := json.Marshal(meta)
		if err != nil {
			return Stats{}, fmt.Errorf("marshal note metadata: %w", err)
		}
		if _, err := insertNote.ExecContext(ctx, rel, title, noteType, string(metaJSON), body, time.Now().UTC().Format(time.RFC3339)); err != nil {
			return Stats{}, fmt.Errorf("insert note: %w", err)
		}
		stats.Notes++
		for _, chunk := range SplitMarkdownByHeadings(body) {
			res, err := insertChunk.ExecContext(ctx, rel, chunk.Heading, chunk.Content, chunk.Index)
			if err != nil {
				return Stats{}, fmt.Errorf("insert chunk: %w", err)
			}
			chunkID, err := res.LastInsertId()
			if err != nil {
				return Stats{}, err
			}
			if _, err := insertFTS.ExecContext(ctx, chunkID, rel, chunk.Heading, chunk.Content); err != nil {
				return Stats{}, fmt.Errorf("insert fts chunk: %w", err)
			}
			batchTexts = append(batchTexts, strings.TrimSpace(title+"\n"+chunk.Heading+"\n"+chunk.Content))
			batchChunkIDs = append(batchChunkIDs, chunkID)
			stats.Chunks++
			if len(batchTexts) >= 32 {
				if err := flushEmbeddings(); err != nil {
					return Stats{}, err
				}
			}
		}
	}
	if err := flushEmbeddings(); err != nil {
		return Stats{}, err
	}
	if err := tx.Commit(); err != nil {
		return Stats{}, fmt.Errorf("commit reindex: %w", err)
	}
	return stats, nil
}

func (s *Store) SearchFTS(ctx context.Context, query string, limit int) ([]ChunkRecord, error) {
	query = sanitizeFTSQuery(query)
	if query == "" {
		return nil, nil
	}
	rows, err := s.DB.QueryContext(ctx, `
		SELECT rowid, note_path, heading,
		       snippet(chunk_fts, 2, '[', ']', ' … ', 18) AS snippet,
		       bm25(chunk_fts) AS rank
		FROM chunk_fts
		WHERE chunk_fts MATCH ?
		ORDER BY rank
		LIMIT ?`, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []ChunkRecord
	for rows.Next() {
		var rec ChunkRecord
		var rank float64
		if err := rows.Scan(&rec.ChunkID, &rec.NotePath, &rec.Heading, &rec.Snippet, &rank); err != nil {
			return nil, err
		}
		rec.Score = rank
		out = append(out, rec)
	}
	return out, rows.Err()
}

func (s *Store) EmbeddingCandidates(ctx context.Context, provider, model string) ([]ChunkRecord, [][]float32, error) {
	rows, err := s.DB.QueryContext(ctx, `
		SELECT c.id, c.note_path, c.heading, c.content, e.vector
		FROM chunks c
		JOIN embeddings e ON e.chunk_id = c.id
		WHERE e.provider = ? AND e.model = ?`, provider, model)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	var records []ChunkRecord
	var vectors [][]float32
	for rows.Next() {
		var rec ChunkRecord
		var blob []byte
		if err := rows.Scan(&rec.ChunkID, &rec.NotePath, &rec.Heading, &rec.Content, &blob); err != nil {
			return nil, nil, err
		}
		rec.Snippet = snippet(rec.Content)
		records = append(records, rec)
		vectors = append(vectors, decodeVector(blob))
	}
	return records, vectors, rows.Err()
}

func (s *Store) NoteLikeSearch(ctx context.Context, query, noteType, pathFilter string, limit int) ([]map[string]any, error) {
	sqlText := `SELECT path, title, type, modified_at FROM notes WHERE 1=1`
	args := make([]any, 0, 4)
	if query != "" {
		sqlText += ` AND (path LIKE ? OR title LIKE ? OR content LIKE ?)`
		pattern := "%" + query + "%"
		args = append(args, pattern, pattern, pattern)
	}
	if noteType != "" {
		sqlText += ` AND type = ?`
		args = append(args, noteType)
	}
	if pathFilter != "" {
		sqlText += ` AND path LIKE ?`
		args = append(args, "%"+pathFilter+"%")
	}
	sqlText += ` ORDER BY modified_at DESC LIMIT ?`
	args = append(args, limit)

	rows, err := s.DB.QueryContext(ctx, sqlText, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []map[string]any
	for rows.Next() {
		var path, title, noteType, modified string
		if err := rows.Scan(&path, &title, &noteType, &modified); err != nil {
			return nil, err
		}
		results = append(results, map[string]any{
			"path":        path,
			"title":       title,
			"type":        noteType,
			"modified_at": modified,
		})
	}
	return results, rows.Err()
}

func (s *Store) Stats(ctx context.Context) (Stats, error) {
	var stats Stats
	if err := s.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM notes`).Scan(&stats.Notes); err != nil {
		return Stats{}, err
	}
	if err := s.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM chunks`).Scan(&stats.Chunks); err != nil {
		return Stats{}, err
	}
	if err := s.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM embeddings`).Scan(&stats.Embeddings); err != nil {
		return Stats{}, err
	}
	return stats, nil
}

func encodeVector(vec []float32) []byte {
	buf := make([]byte, len(vec)*4)
	for i, v := range vec {
		binary.LittleEndian.PutUint32(buf[i*4:], math.Float32bits(v))
	}
	return buf
}

func decodeVector(buf []byte) []float32 {
	if len(buf)%4 != 0 {
		return nil
	}
	vec := make([]float32, len(buf)/4)
	for i := range vec {
		vec[i] = math.Float32frombits(binary.LittleEndian.Uint32(buf[i*4:]))
	}
	return vec
}

func titleFromMeta(path string, meta map[string]any) string {
	if title, ok := meta["title"].(string); ok && title != "" {
		return title
	}
	return strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
}

func typeFromMeta(path string, meta map[string]any) string {
	if noteType, ok := meta["type"].(string); ok && noteType != "" {
		return noteType
	}
	first := strings.Split(filepath.ToSlash(path), "/")[0]
	switch first {
	case "Projects":
		return "project"
	case "Areas":
		return "area"
	case "Resources":
		return "resource"
	case "Archives":
		return "archive"
	default:
		return "note"
	}
}

func snippet(content string) string {
	content = strings.TrimSpace(strings.ReplaceAll(content, "\n", " "))
	if len(content) <= 180 {
		return content
	}
	return content[:177] + "..."
}

func sanitizeFTSQuery(input string) string {
	tokens := ftsTokenPattern.FindAllString(strings.ToLower(input), -1)
	if len(tokens) == 0 {
		return ""
	}
	return strings.Join(tokens, " ")
}
