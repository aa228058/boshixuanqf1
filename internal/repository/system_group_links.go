package repository

import (
	"context"

	"github.com/google/uuid"
	"tg-sender/internal/domain"
)

func (r *SystemGroupRepository) attachLinks(ctx context.Context, groups []domain.SystemGroup) error {
	if len(groups) == 0 {
		return nil
	}
	ids := make([]uuid.UUID, 0, len(groups))
	for _, group := range groups {
		ids = append(ids, group.SGID)
	}
	rows, err := r.db.Query(ctx, "SELECT sg_id,link_id,original_link,source,validity_status,link_type FROM system_group_links WHERE sg_id=ANY($1) ORDER BY created_at ASC,link_id ASC", ids)
	if err != nil {
		return err
	}
	defer rows.Close()
	byGroup := make(map[uuid.UUID][]domain.SystemGroupLink, len(groups))
	for rows.Next() {
		var sgID uuid.UUID
		var link domain.SystemGroupLink
		if err := rows.Scan(&sgID, &link.LinkID, &link.TmeLink, &link.Source, &link.Status, &link.LinkType); err != nil {
			return err
		}
		byGroup[sgID] = append(byGroup[sgID], link)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for i := range groups {
		groups[i].Entries = byGroup[groups[i].SGID]
		groups[i].EntryCount = len(groups[i].Entries)
		if groups[i].TmeLink == "" && len(groups[i].Entries) > 0 {
			groups[i].TmeLink = groups[i].Entries[0].TmeLink
		}
		// The API status is link validity, never whether a tenant account joined.
		hasValid := false
		allInvalid := len(groups[i].Entries) > 0
		for _, entry := range groups[i].Entries {
			if entry.Status == "valid" {
				hasValid = true
			}
			if entry.Status != "invalid" {
				allInvalid = false
			}
		}
		switch {
		case hasValid:
			groups[i].Status = "valid"
		case allInvalid || groups[i].Status == "invalid":
			groups[i].Status = "invalid"
		default:
			groups[i].Status = "pending"
		}
	}
	return nil
}

func (r *SystemGroupRepository) UpdateLinkValidity(ctx context.Context, sgID uuid.UUID, chatID int64, status string) error {
	if status != "valid" && status != "invalid" && status != "pending" {
		return nil
	}
	if chatID != 0 {
		_, err := r.db.Exec(ctx, "UPDATE system_group_links SET validity_status=$3,last_verified_at=NOW() WHERE sg_id IN (SELECT sg_id FROM system_groups WHERE telegram_chat_id=$1 OR sg_id=$2)", chatID, sgID, status)
		return err
	}
	_, err := r.db.Exec(ctx, "UPDATE system_group_links SET validity_status=$2,last_verified_at=NOW() WHERE sg_id=$1", sgID, status)
	return err
}
