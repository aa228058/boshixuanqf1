package task

import (
	"context"
	"log"
	"strings"
	"tg-sender/internal/safego"
	"time"

	"github.com/mtgo-labs/mtgo/telegram"
	"github.com/mtgo-labs/mtgo/tg"
)

func (e *Engine) StartApprovalChecker(ctx context.Context) {
	safego.Go(func() { e.runApprovalChecker(ctx) })
}

func (e *Engine) runApprovalChecker(ctx context.Context) {
	ticker := time.NewTicker(e.cfg.ApprovalCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			e.checkPendingApprovals(ctx)
		case <-e.stopCh:
			return
		case <-ctx.Done():
			return
		}
	}
}

func (e *Engine) checkPendingApprovals(ctx context.Context) {
	if e.approvalRepo == nil {
		return
	}

	pendings, err := e.approvalRepo.GetPendingAll(ctx, e.cfg.ApprovalCheckLimit)
	if err != nil {
		return
	}

	for _, p := range pendings {
		if _, loaded := e.approvalChecks.LoadOrStore(p.ID, true); loaded {
			continue
		}
		if p.CheckCount >= p.MaxChecks {
			resolved, err := e.approvalRepo.ResolvePending(ctx, p.TenantID, p.ID, "timeout")
			if err != nil {
				log.Printf("[approval] 更新状态失败: %v", err)
			} else if resolved {
				if _, err := e.repo.FinalizeTaskAfterApprovals(ctx, p.TenantID, p.TaskID); err != nil {
					taskLog.Printf("[task] UpdateStatus failed: %v", err)
				}
			}
			e.approvalChecks.Delete(p.ID)
			continue
		}

		safego.Go(func() {
			defer e.approvalChecks.Delete(p.ID)
			defer func() {
				if r := recover(); r != nil {
					taskLog.Printf("[task] approval check panic recovered: %v", r)
				}
			}()
			client, err := e.accEngine.GetClient(ctx, p.TenantID, p.AccountID)
			if err != nil {
				taskLog.Printf("[task] GetClient failed for approval check: %v", err)
				// Count unavailable-account checks too; otherwise an offline account
				// can leave its approval pending forever without reaching MaxChecks.
				if incErr := e.approvalRepo.IncrementCheck(ctx, p.TenantID, p.ID); incErr != nil {
					log.Printf("[approval] 增加检查次数失败: %v", incErr)
				}
				return
			}
			defer e.accEngine.ReleaseClient(p.TenantID, p.AccountID, client)

			inGroup := e.isAccountInGroup(ctx, client, p.ChatID, p.AccessHash, p.TargetLink)
			if inGroup {
				resolved, err := e.approvalRepo.ResolvePending(ctx, p.TenantID, p.ID, "approved")
				if err != nil {
					log.Printf("[approval] 更新状态失败: %v", err)
				} else if resolved {
					if _, err := e.repo.FinalizeTaskAfterApprovals(ctx, p.TenantID, p.TaskID); err != nil {
						taskLog.Printf("[task] UpdateStatus failed: %v", err)
					}
				}
			} else {
				if err := e.approvalRepo.IncrementCheck(ctx, p.TenantID, p.ID); err != nil {
					log.Printf("[approval] 增加检查次数失败: %v", err)
				}
			}
		})
	}
}

func (e *Engine) isAccountInGroup(ctx context.Context, client *telegram.Client, chatID int64, accessHash int64, targetLink string) bool {
	// Method 1: Check via GetDialogs (safest, no side effects)
	// GetDialogs returns []*types.Chat, not []*types.Dialog
	dialogs, err := client.GetDialogs(ctx, e.cfg.DialogsLimit, 0)
	if err == nil {
		for _, chat := range dialogs {
			if chat == nil {
				continue
			}
			// Match by chatID (normalize both to positive for comparison)
			if chatID != 0 {
				dialogID := chat.ID
				if dialogID < 0 {
					dialogID = -dialogID
				}
				checkID := chatID
				if checkID < 0 {
					checkID = -checkID
				}
				if dialogID == checkID {
					return true
				}
			}
			// Match by invite link in title/username (fallback for private groups)
			if targetLink != "" && (strings.Contains(chat.Title, targetLink) || strings.Contains(chat.Username, targetLink)) {
				return true
			}
		}
	}

	// For private invite links the chat ID is unknown until approval. Use
	// messages.checkChatInvite, which is read-only, to distinguish an accepted
	// invite (ChatInviteAlready) from a still-pending/rejected invite.
	if targetLink != "" && (strings.Contains(targetLink, "+") || strings.Contains(targetLink, "joinchat")) {
		hash := extractInviteHash(targetLink)
		if hash != "" {
			result, checkErr := client.Invoke(ctx, &tg.MessagesCheckChatInviteRequest{Hash: hash})
			if checkErr == nil {
				if _, ok := result.(*tg.ChatInviteAlready); ok {
					return true
				}
			}
		}
	}

	// Never re-import an invite link from the checker: ImportChatInvite is
	// mutating and can submit another join request while merely polling status.
	return false
}

// StartResourceMonitor starts a background goroutine that monitors system resources
// and pushes alerts via SSE when thresholds are exceeded.
