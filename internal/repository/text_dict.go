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

// TextDictRepository 文案变量(数据字典), 租户级。
type TextDictRepository struct {
	db *pgxpool.Pool
}

func NewTextDictRepository(db *pgxpool.Pool) *TextDictRepository {
	return &TextDictRepository{db: db}
}

// List returns the current tenant's named text collections and their saved items.
func (r *TextDictRepository) List(ctx context.Context, tenantID uuid.UUID) ([]*domain.TextDict, error) {
	rows, err := r.db.Query(ctx, `SELECT d.dict_id,d.tenant_id,d.name,d.created_at,e.word
		FROM text_dicts d LEFT JOIN text_dict_entries e ON e.dict_id=d.dict_id
		WHERE d.tenant_id=$1 ORDER BY d.created_at DESC,e.entry_id`, tenantID)
	if err != nil { return nil, err }; defer rows.Close()
	byID:=map[uuid.UUID]*domain.TextDict{}; var list []*domain.TextDict
	for rows.Next() {
		var id,tid uuid.UUID; var name string; var created time.Time; var word *string
		if err:=rows.Scan(&id,&tid,&name,&created,&word);err!=nil{return nil,err}
		item,ok:=byID[id]; if !ok { item=&domain.TextDict{DictID:id,TenantID:tid,Name:name,Items:[]string{},CreatedAt:created};byID[id]=item;list=append(list,item) }
		if word!=nil { item.Items=append(item.Items,*word) }
	}
	return list,rows.Err()
}
func (r *TextDictRepository) Create(ctx context.Context,tenantID uuid.UUID,name string,items []string)(*domain.TextDict,error) {
	tx,err:=r.db.BeginTx(ctx,pgx.TxOptions{});if err!=nil{return nil,err};defer tx.Rollback(ctx)
	d:=&domain.TextDict{TenantID:tenantID,Name:name,Items:[]string{}}
	if err=tx.QueryRow(ctx,`INSERT INTO text_dicts(tenant_id,name) VALUES($1,$2) RETURNING dict_id,created_at`,tenantID,name).Scan(&d.DictID,&d.CreatedAt);err!=nil{return nil,err}
	for _,item:=range items { if _,err=tx.Exec(ctx,`INSERT INTO text_dict_entries(dict_id,word,weight) VALUES($1,$2,1)`,d.DictID,item);err!=nil{return nil,err};d.Items=append(d.Items,item) }
	if err=tx.Commit(ctx);err!=nil{return nil,err};return d,nil
}
func (r *TextDictRepository) Update(ctx context.Context,tenantID,dictID uuid.UUID,name string,items []string)error {
	tx,err:=r.db.BeginTx(ctx,pgx.TxOptions{});if err!=nil{return err};defer tx.Rollback(ctx)
	res,err:=tx.Exec(ctx,`UPDATE text_dicts SET name=$3 WHERE dict_id=$1 AND tenant_id=$2`,dictID,tenantID,name);if err!=nil{return err};if res.RowsAffected()==0{return pgx.ErrNoRows}
	if _,err=tx.Exec(ctx,`DELETE FROM text_dict_entries WHERE dict_id=$1`,dictID);err!=nil{return err}
	for _,item:=range items { if _,err=tx.Exec(ctx,`INSERT INTO text_dict_entries(dict_id,word,weight) VALUES($1,$2,1)`,dictID,item);err!=nil{return err} }
	return tx.Commit(ctx)
}
func (r *TextDictRepository) Delete(ctx context.Context,tenantID,dictID uuid.UUID)error {
	res,err:=r.db.Exec(ctx,`DELETE FROM text_dicts WHERE dict_id=$1 AND tenant_id=$2`,dictID,tenantID);if err!=nil{return err};if res.RowsAffected()==0{return pgx.ErrNoRows};return nil
}

// WordsByNames 一次取出多个词典的全部词条(群发前构建替换池, 避免逐条查库)
func (r *TextDictRepository) WordsByNames(ctx context.Context, tenantID uuid.UUID, names []string) (map[string][]string, error) {
	if len(names) == 0 {
		return nil, nil
	}
	rows, err := r.db.Query(ctx, `
		SELECT d.name, e.word FROM text_dicts d
		JOIN text_dict_entries e ON e.dict_id = d.dict_id
		WHERE d.tenant_id = $1 AND d.name = ANY($2)`, tenantID, names)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string][]string{}
	for rows.Next() {
		var n, w string
		if err := rows.Scan(&n, &w); err != nil {
			return nil, err
		}
		out[n] = append(out[n], w)
	}
	return out, rows.Err()
}

// PickWord 随机取一个词(按权重, 发送时用)。无词条返回空串。
func (r *TextDictRepository) PickWord(ctx context.Context, tenantID uuid.UUID, name string) (string, error) {
	var w string
	err := r.db.QueryRow(ctx, `
		SELECT e.word FROM text_dict_entries e
		JOIN text_dicts d ON d.dict_id = e.dict_id
		-- V22 审计修复: weight<=0 时 -ln(random())/weight = Inf/NaN 排序未定义——排除
		WHERE d.tenant_id = $1 AND d.name = $2 AND e.weight > 0
		ORDER BY -ln(random()) / e.weight  -- 指数分布: weight 越大越容易被选中
		LIMIT 1`, tenantID, name).Scan(&w)
	return w, err
}
