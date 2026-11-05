package repository

import (
	"context"

	"github.com/google/uuid"
	"tg-sender/internal/domain"
)

func (r *SystemGroupRepository) attachLinks(ctx context.Context, groups []domain.SystemGroup) error {
	if len(groups)==0 { return nil }
	ids:=make([]uuid.UUID,0,len(groups))
	for _, group := range groups { ids=append(ids,group.SGID) }
	rows,err:=r.db.Query(ctx,"SELECT sg_id,link_id,original_link,source,validity_status,link_type FROM system_group_links WHERE sg_id=ANY($1) ORDER BY created_at ASC,link_id ASC",ids)
	if err!=nil { return err }
	defer rows.Close()
	byGroup:=make(map[uuid.UUID][]domain.SystemGroupLink,len(groups))
	for rows.Next() {
		var sgID uuid.UUID
		var link domain.SystemGroupLink
		if err:=rows.Scan(&sgID,&link.LinkID,&link.TmeLink,&link.Source,&link.Status,&link.LinkType);err!=nil{return err}
		byGroup[sgID]=append(byGroup[sgID],link)
	}
	if err:=rows.Err();err!=nil{return err}
	for i:=range groups {
		groups[i].Entries=byGroup[groups[i].SGID]
		groups[i].EntryCount=len(groups[i].Entries)
		if groups[i].TmeLink=="" && len(groups[i].Entries)>0 {
			groups[i].TmeLink=groups[i].Entries[0].TmeLink
		}
	}
	return nil
}
