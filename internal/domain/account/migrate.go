package account

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/mtgo-labs/mtgo/telegram"
	"github.com/mtgo-labs/mtgo/tg"
	"github.com/mtgo-labs/storage"
)

// MigrateDC 把账号会话从当前 DC 迁移到目标 DC（救活被机房路由封死的 DC，如 DC1→DC2）。
// 原理: 源DC导出授权(auth.exportAuthorization) → 目标DC匿名DH握手 →
//
//	导入授权(auth.importAuthorization) → 新auth_key覆盖库存会话。
//
// 前提: 必须已有一个活着的源DC连接（保活模式负责保持），导出令牌短时效须立即消费。
// 回滚: 迁移不注销源DC授权，原 .session 文件重新导入即可回到源DC。
func (e *Engine) MigrateDC(ctx context.Context, tenantID, accountID uuid.UUID, targetDC int32) error {
	if targetDC < 1 || targetDC > 5 {
		return fmt.Errorf("目标 DC 必须是 1-5，收到 %d", targetDC)
	}

	// 1. 拿活着的源DC客户端：优先保活的常驻客户端，其次池里现成连接
	key := accountKey(tenantID, accountID)
	e.mu.RLock()
	mc, managed := e.clients[key]
	e.mu.RUnlock()

	// 2. 读账号与旧会话元数据（UserID 迁移后不变）
	acc, err := e.repo.GetByID(ctx, tenantID, accountID)
	if err != nil {
		return err
	}
	var userID int64
	var oldDC int32
	if err := e.pool.QueryRow(ctx,
		`SELECT user_id, dc FROM sessions WHERE session_id = $1`, acc.SessionName,
	).Scan(&userID, &oldDC); err != nil {
		return fmt.Errorf("读取旧会话失败: %w", err)
	}
	if oldDC == targetDC {
		return fmt.Errorf("会话已在 DC%d，无需迁移", targetDC)
	}

	// K修复: 保活源连接必须持 mc.mu 使用(mtgo 非线程安全, 任务可能正占用)。
	// TryLock 失败说明任务执行中, 本轮迁移放弃稍后重试, 绝不阻塞等锁。
	if managed && mc.client != nil && mc.client.IsConnected() {
		if !mc.mu.TryLock() {
			return fmt.Errorf("账号 %s 正被任务占用, DC 迁移请稍后重试", accountID)
		}
		defer mc.mu.Unlock()
		log.Printf("[migrate] 账号 %s 使用保活连接作为源DC\n", accountID)
		return e.migrateDCWithSource(ctx, tenantID, accountID, acc.SessionName, mc.client, userID, oldDC, targetDC)
	}

	c, err := e.GetClient(ctx, tenantID, accountID)
	if err != nil {
		return fmt.Errorf("拿不到源DC连接（需先连上保活再迁移）: %w", err)
	}
	defer e.ReleaseClient(tenantID, accountID, c)
	return e.migrateDCWithSource(ctx, tenantID, accountID, acc.SessionName, c, userID, oldDC, targetDC)
}

// migrateDCWithSource 用已就位的源DC连接执行迁移主体。
// T修复: 目标DC客户端先建好连, 再导出授权令牌 —— 令牌窗口压到一次 RPC 内,
// 避免"先取令牌再 Connect(最长45s)"导致令牌过期、迁移失败进冷却。
func (e *Engine) migrateDCWithSource(ctx context.Context, tenantID, accountID uuid.UUID, sessionName string, src *telegram.Client, userID int64, oldDC, targetDC int32) error {
	// 3. 目标DC临时客户端：全新匿名DH握手，走HTTP传输（目标DC可达即可）
	cfg := &telegram.Config{
		// #nosec G115 -- TGAPIID is range-validated by Config.Validate before account migration.
		APIID:         int32(e.cfg.TGAPIID),
		APIHash:       e.cfg.TGAPIHash,
		SessionName:   fmt.Sprintf("migrate_tmp_%s", sessionName),
		DC:            int(targetDC),
		InMemory:      true,
		HTTPTransport: httpTransportCfg(dcHTTPURLs(targetDC, "")),
	}
	dst, err := telegram.NewClient(cfg.APIID, cfg.APIHash, cfg)
	if err != nil {
		return fmt.Errorf("创建目标DC客户端失败: %w", err)
	}
	if err := dst.Connect(45 * time.Second); err != nil {
		dst.Stop()
		return fmt.Errorf("连接目标DC失败（该DC的HTTP传输也不通?）: %w", err)
	}
	defer dst.Stop()

	// 4. 源DC导出授权（目标已就绪, 令牌拿到立刻导入）
	expCtx, expCancel := context.WithTimeout(ctx, 60*time.Second)
	exp, err := src.Raw().AuthExportAuthorization(expCtx, &tg.AuthExportAuthorizationRequest{DCID: targetDC})
	expCancel()
	if err != nil {
		return fmt.Errorf("源DC导出授权失败: %w", err)
	}

	// 5. 目标DC导入授权 → 该 auth_key 获得账号身份
	impCtx, impCancel := context.WithTimeout(ctx, 60*time.Second)
	if _, err := dst.Raw().AuthImportAuthorization(impCtx, &tg.AuthImportAuthorizationRequest{
		ID:    exp.ID,
		Bytes: exp.Bytes,
	}); err != nil {
		impCancel()
		return fmt.Errorf("目标DC导入授权失败: %w", err)
	}
	impCancel()

	// 6. 抠出目标DC的 auth_key，覆盖库存会话
	newKey := dst.Session().AuthKey()
	if len(newKey) != 256 {
		return fmt.Errorf("目标DC auth_key 长度异常: %d", len(newKey))
	}
	if err := e.StoreImportedSession(ctx, sessionName, "", &storage.Session{
		SessionID: sessionName,
		DC:        int(targetDC),
		// #nosec G115 -- TGAPIID is range-validated by Config.Validate before account migration.
		APIID:   int32(e.cfg.TGAPIID),
		AuthKey: newKey,
		UserID:  userID,
	}); err != nil {
		return fmt.Errorf("写入新会话失败（旧会话未动，可重试）: %w", err)
	}

	// 7. 断开旧DC连接（保活/池化），状态回待机，下次取客户端自动走新DC
	if err := e.DisconnectAccount(ctx, tenantID, accountID); err != nil {
		// 会话已写入目标 DC；断开旧客户端失败不回滚新会话，但必须留下可排查记录。
		log.Printf("[migrate] 账号 %s 已迁移但断开旧连接失败: %v", accountID, err)
	}
	log.Printf("[migrate] 账号 %s 已从 DC%d 迁移到 DC%d\n", accountID, oldDC, targetDC)
	return nil
}
