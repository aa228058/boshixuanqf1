package grouplib

import (
	"context"
	"log"
	"math/rand"
	"strings"
	"time"

	"github.com/google/uuid"
	"tg-sender/internal/domain"
)

// resolveBatchSize 每轮(2分钟)最多解析的条数。1 万条私密链接按 30 条/轮 ≈ 11 小时跑完,
// 对 TG 来说是极低频, 不会触发 flood; 也可临时调大加速(撞限流会自动等)。
const resolveBatchSize = 30

// resolvePendingTitles 后台批量解析: 扫出私密且无群名的行, 用 checkChatInvite 补名。
// 挂进 Engine.runOnce, 导入 1 万条后无需任何手工操作, 群名自动逐批补齐。
func (e *Engine) resolvePendingTitles(ctx context.Context) {
	type row struct {
		sgID uuid.UUID
		link string
		dedupKey string
		username string
		gtype string
		title string
	}
	rows, err := e.pool.Query(ctx, "SELECT sg_id,tme_link,dedup_key,COALESCE(username,''),gtype,COALESCE(title,'') FROM system_groups WHERE gtype IN ('private','unknown') AND (COALESCE(title,'')='' OR (resolution_status='identity_pending' AND last_seen_at<NOW()-INTERVAL '30 minutes')) AND status='active' ORDER BY last_seen_at ASC LIMIT $1", resolveBatchSize)
	if err != nil {
		log.Printf("[grouplib] 扫描待解析群失败: %v", err)
		return
	}
	var pending []row
	for rows.Next() {
		var item row
		if err := rows.Scan(&item.sgID,&item.link,&item.dedupKey,&item.username,&item.gtype,&item.title); err == nil { pending = append(pending,item) }
	}
	rows.Close()
	if len(pending)==0 { return }

	type accRef struct { tenantID uuid.UUID; accountID uuid.UUID }
	accRows, err := e.pool.Query(ctx, "SELECT tenant_id,account_id FROM accounts WHERE status='online' AND (blocked_until IS NULL OR blocked_until<NOW()) ORDER BY RANDOM() LIMIT 5")
	if err != nil { log.Printf("[grouplib] 扫描在线账号失败: %v",err); return }
	var accs []accRef
	for accRows.Next() {
		var a accRef
		if err := accRows.Scan(&a.tenantID,&a.accountID); err == nil { accs=append(accs,a) }
	}
	accRows.Close()

	for i,p := range pending {
		if len(accs)==0 && p.title!="" { continue }
		var title string
		var count int
		var gtype string
		var chatID int64
		var resolveErr error
		if len(accs)>0 {
			a := accs[rand.Intn(len(accs))]
			client, clientErr := e.acc.GetClient(ctx,a.tenantID,a.accountID)
			if clientErr == nil {
				if p.gtype=="private" {
					hash := strings.TrimPrefix(strings.TrimPrefix(p.link,"https://"),"t.me/+")
					hash = strings.TrimPrefix(hash,"+")
					title,count,gtype,chatID,resolveErr = resolveInviteInfo(ctx,client,hash)
				} else {
					title,count,gtype,chatID,resolveErr = resolveGroupInfo(ctx,client,p.username)
				}
				e.acc.ReleaseClient(a.tenantID,a.accountID,client)
			} else { resolveErr=clientErr }
		}
		if resolveErr != nil || len(accs)==0 {
			var webTitle string
			var webCount int
			if p.gtype=="private" {
				hash := strings.TrimPrefix(strings.TrimPrefix(p.link,"https://"),"t.me/+")
				hash = strings.TrimPrefix(hash,"+")
				webTitle,webCount,resolveErr = resolveInviteInfoViaWeb(ctx,hash)
			} else {
				webTitle,webCount,resolveErr = resolvePublicGroupInfoViaWeb(ctx,p.username)
			}
			if resolveErr == nil && (webTitle!="" || webCount>0) {
				title,count,gtype,chatID = webTitle,webCount,p.gtype,0
			}
		}
		if resolveErr != nil {
			errText := strings.ToLower(resolveErr.Error())
			if p.gtype=="private" && (strings.Contains(errText,"过期") || strings.Contains(errText,"无效")) {
				_ = e.repo.MarkInvalidByDedupKey(ctx,p.dedupKey)
				_ = e.repo.UpdateLinkValidity(ctx,p.sgID,0,"invalid")
			}
		} else if title!="" || count>0 {
			if gtype=="" { gtype=p.gtype }
			if err := e.repo.UpdateResolvedInfo(ctx,p.sgID,title,count,gtype,chatID); err != nil {
				log.Printf("[grouplib] 回写群组资料失败 %s: %v",p.sgID,err)
			} else { _ = e.repo.UpdateLinkValidity(ctx,p.sgID,chatID,"valid") }
		}
		if i<len(pending)-1 {
			select { case <-ctx.Done(): return; case <-time.After(2*time.Second): }
		}
	}

	log.Printf("[grouplib] 本轮解析补名 %d 条(待解析队列滚动处理)", len(pending))
}

// 保留 domain 引用(resolver 返回的群信息最终也以 SystemGroup 形态被查询)
var _ = domain.SystemGroup{}
