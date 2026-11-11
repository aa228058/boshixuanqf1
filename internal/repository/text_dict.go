package repository

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"tg-sender/internal/domain"
)

// TextDictRepository 文案素材分组, 租户级。
type TextDictRepository struct {
	db *pgxpool.Pool
}

func NewTextDictRepository(db *pgxpool.Pool) *TextDictRepository {
	return &TextDictRepository{db: db}
}

// List returns the current tenant's named text collections and their saved items.
func (r *TextDictRepository) List(ctx context.Context, tenantID uuid.UUID) ([]*domain.TextDict, error) {
	rows, err := r.db.Query(ctx, `SELECT d.dict_id, d.tenant_id, d.name, d.created_at, e.word
		FROM text_dicts d LEFT JOIN text_dict_entries e ON e.dict_id = d.dict_id
		WHERE d.tenant_id = $1 ORDER BY d.created_at DESC, e.entry_id`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	byID := map[uuid.UUID]*domain.TextDict{}
	var list []*domain.TextDict
	for rows.Next() {
		var id, tid uuid.UUID
		var name string
		var created time.Time
		var word *string
		if err := rows.Scan(&id, &tid, &name, &created, &word); err != nil {
			return nil, err
		}
		item, ok := byID[id]
		if !ok {
			item = &domain.TextDict{
				DictID: id, TenantID: tid, Name: name,
				Items: []string{}, CreatedAt: created,
			}
			byID[id] = item
			list = append(list, item)
		}
		if word != nil {
			item.Items = append(item.Items, *word)
			item.WordCount++
		}
	}
	return list, rows.Err()
}

func (r *TextDictRepository) Create(ctx context.Context, tenantID uuid.UUID, name string, items []string) (*domain.TextDict, error) {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	d := &domain.TextDict{TenantID: tenantID, Name: name, Items: []string{}}
	if err = tx.QueryRow(ctx, `INSERT INTO text_dicts (tenant_id, name)
		VALUES ($1, $2) RETURNING dict_id, created_at`, tenantID, name).Scan(&d.DictID, &d.CreatedAt); err != nil {
		return nil, err
	}
	for _, item := range items {
		if _, err = tx.Exec(ctx, `INSERT INTO text_dict_entries (dict_id, word, weight)
			VALUES ($1, $2, 1)`, d.DictID, item); err != nil {
			return nil, err
		}
		d.Items = append(d.Items, item)
		d.WordCount++
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	return d, nil
}

func (r *TextDictRepository) Update(ctx context.Context, tenantID, dictID uuid.UUID, name string, items []string) error {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	res, err := tx.Exec(ctx, `UPDATE text_dicts SET name = $3
		WHERE dict_id = $1 AND tenant_id = $2`, dictID, tenantID, name)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	if _, err = tx.Exec(ctx, `DELETE FROM text_dict_entries WHERE dict_id = $1`, dictID); err != nil {
		return err
	}
	for _, item := range items {
		if _, err = tx.Exec(ctx, `INSERT INTO text_dict_entries (dict_id, word, weight)
			VALUES ($1, $2, 1)`, dictID, item); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (r *TextDictRepository) Delete(ctx context.Context, tenantID, dictID uuid.UUID) error {
	res, err := r.db.Exec(ctx, `DELETE FROM text_dicts WHERE dict_id = $1 AND tenant_id = $2`, dictID, tenantID)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// WordsByNames and PickWord remain for legacy internal consumers.
// The management API only stores and retrieves snippets.
func (r *TextDictRepository) WordsByNames(ctx context.Context, tenantID uuid.UUID, names []string) (map[string][]string, error) {
	if len(names) == 0 {
		return nil, nil
	}
	rows, err := r.db.Query(ctx, `SELECT d.name, e.word FROM text_dicts d
		JOIN text_dict_entries e ON e.dict_id = d.dict_id
		WHERE d.tenant_id = $1 AND d.name = ANY($2)`, tenantID, names)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := map[string][]string{}
	for rows.Next() {
		var name, word string
		if err := rows.Scan(&name, &word); err != nil {
			return nil, err
		}
		out[name] = append(out[name], strings.TrimSpace(word))
	}
	return out, rows.Err()
}

func (r *TextDictRepository) PickWord(ctx context.Context, tenantID uuid.UUID, name string) (string, error) {
	var word string
	err := r.db.QueryRow(ctx, `SELECT e.word FROM text_dict_entries e
		JOIN text_dicts d ON d.dict_id = e.dict_id
		WHERE d.tenant_id = $1 AND d.name = $2 AND e.weight > 0
		ORDER BY -ln(random()) / e.weight LIMIT 1`, tenantID, name).Scan(&word)
	return word, err
}
