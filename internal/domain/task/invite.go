package task

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mtgo-labs/mtgo/tg"
	"github.com/mtgo-labs/mtgo/tgerr"
	"tg-sender/internal/domain"
	"tg-sender/internal/repository"
	securerand "tg-sender/internal/secure"
)

// executeInviteMembers invites collected source-group members into the target
// group. Every external side effect is guarded by a durable per-member claim.
func (e *Engine) executeInviteMembers(ctx context.Context, task *domain.Task) error {
	targetID, err := uuid.Parse(fmt.Sprint(task.Payload["target_group_id"]))
	if err != nil {
		return fmt.Errorf("目标群ID缺失: %w", err)
	}
	sourceID, err := uuid.Parse(fmt.Sprint(task.Payload["source_group_id"]))
	if err != nil {
		return fmt.Errorf("来源群ID缺失: %w", err)
	}
	target, err := e.groupRepo.GetByID(ctx, task.TenantID, targetID)
	if err != nil || target == nil {
		return fmt.Errorf("目标群不存在")
	}
	source, err := e.groupRepo.GetByID(ctx, task.TenantID, sourceID)
	if err != nil || source == nil {
		return fmt.Errorf("来源群不存在")
	}
	if len(task.AccountIDs) == 0 {
		return fmt.Errorf("无执行账号")
	}
	accID := task.AccountIDs[0]

	total, err := e.memberRepo.CountByGroup(ctx, task.TenantID, sourceID)
	if err != nil {
		return fmt.Errorf("统计成员失败: %w", err)
	}
	// A prior worker may have died after claiming a member. Resolve only this
	// task's old claims; never turn another tenant's active invite into unknown.
	staleCount, err := e.repo.ResolveStaleInviteAttempts(ctx, task.TenantID, task.TaskID, time.Now().Add(-5*time.Minute))
	if err != nil {
		return fmt.Errorf("恢复陈旧邀请检查点失败: %w", err)
	}
	if staleCount > 0 {
		taskLog.Printf("[invite] 任务 %s: %d 条陈旧邀请记录转为结果未知，禁止自动重试", task.TaskID, staleCount)
	}

	initProg := domain.TaskProgress{Total: total}
	if err := e.repo.UpdateProgressExact(ctx, task.TenantID, task.TaskID, initProg); err != nil {
		return fmt.Errorf("初始化邀请进度失败: %w", err)
	}
	e.publishProgress(task.TenantID, task.TaskID, initProg)
	taskLog.Printf("[invite] 任务 %s: 邀请 %s 的 %d 名成员进 %s", task.TaskID, source.Title, total, target.Title)

	client, err := e.accEngine.GetClient(ctx, task.TenantID, accID)
	if err != nil {
		return fmt.Errorf("取客户端失败: %w", err)
	}
	defer e.accEngine.ReleaseClient(task.TenantID, accID, client)

	channel := &tg.InputChannel{ChannelID: target.TGChatID, AccessHash: target.AccessHash}
	invited, failed := 0, 0
	const pageSize = 200
	finishAttempt := func(key repository.InviteAttemptKey, status repository.InviteAttemptStatus, lastError string) error {
		persistCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		return e.repo.FinishInviteAttempt(persistCtx, key, status, lastError)
	}

	saveProgress := func() error {
		prog := domain.TaskProgress{
			Total:         total,
			Done:          invited,
			Failed:        failed,
			CurrentTarget: fmt.Sprintf("已邀 %d / 失败 %d", invited, failed),
		}
		persistCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		if err := e.repo.UpdateProgressExact(persistCtx, task.TenantID, task.TaskID, prog); err != nil {
			return fmt.Errorf("保存邀请进度失败: %w", err)
		}
		e.publishProgress(task.TenantID, task.TaskID, prog)
		return nil
	}

	for offset := 0; ; offset += pageSize {
		select {
		case <-ctx.Done():
			if err := saveProgress(); err != nil {
				return err
			}
			return ctx.Err()
		default:
		}

		members, err := e.memberRepo.ListByGroup(ctx, task.TenantID, sourceID, pageSize, offset)
		if err != nil {
			return fmt.Errorf("读取来源群成员分页 offset=%d 失败: %w", offset, err)
		}
		if len(members) == 0 {
			break
		}

		for _, member := range members {
			select {
			case <-ctx.Done():
				if err := saveProgress(); err != nil {
					return err
				}
				return ctx.Err()
			default:
			}

			if member.TGUserID <= 0 || member.AccessHash == 0 {
				failed++
				continue
			}

			key := repository.InviteAttemptKey{
				TenantID:  task.TenantID,
				TaskID:    task.TaskID,
				AccountID: accID,
				TGUserID:  member.TGUserID,
			}
			claimed, status, err := e.repo.ClaimInviteAttempt(ctx, key)
			if err != nil {
				return fmt.Errorf("领取成员 %d 邀请检查点失败: %w", member.TGUserID, err)
			}
			if !claimed {
				switch status {
				case repository.InviteAttemptSucceeded:
					invited++
					continue
				case repository.InviteAttemptFailedDefinite:
					// ClaimInviteAttempt retries this state atomically. Seeing it
					// here means a concurrent transition raced with this worker.
					return fmt.Errorf("成员 %d 邀请状态发生并发变化: %s", member.TGUserID, status)
				case repository.InviteAttemptInProgress, repository.InviteAttemptOutcomeUnknown:
					failed++
					if err := saveProgress(); err != nil {
						return err
					}
					return fmt.Errorf("成员 %d 邀请结果未确认（%s），为防止重复邀请已停止任务", member.TGUserID, status)
				default:
					return fmt.Errorf("成员 %d 返回未知邀请状态: %q", member.TGUserID, status)
				}
			}

			// If cancellation is already visible before the RPC starts, the
			// side effect definitely did not happen and is safe to retry.
			select {
			case <-ctx.Done():
				if err := finishAttempt(key, repository.InviteAttemptFailedDefinite, ctx.Err().Error()); err != nil {
					return fmt.Errorf("取消前释放成员 %d 邀请检查点失败: %w", member.TGUserID, err)
				}
				if err := saveProgress(); err != nil {
					return err
				}
				return ctx.Err()
			default:
			}

			_, inviteErr := client.Raw().ChannelsInviteToChannel(ctx, &tg.ChannelsInviteToChannelRequest{
				Channel: channel,
				Users:   []tg.InputUserClass{&tg.InputUser{UserID: member.TGUserID, AccessHash: member.AccessHash}},
			})
			if inviteErr == nil {
				if err := finishAttempt(key, repository.InviteAttemptSucceeded, ""); err != nil {
					// Keep the in_progress tombstone. A later stale sweep will
					// mark it outcome_unknown, never automatically retry it.
					return fmt.Errorf("Telegram 已接受成员 %d 邀请，但保存成功状态失败: %w", member.TGUserID, err)
				}
				invited++
			} else {
				errStr := strings.ToLower(inviteErr.Error())
				if isDefiniteInviteRPCError(inviteErr) {
					// A typed Telegram RPC error is an explicit server response:
					// the invite was rejected, so it is a definite failure.
					if err := finishAttempt(key, repository.InviteAttemptFailedDefinite, inviteErr.Error()); err != nil {
						return fmt.Errorf("记录成员 %d 的 Telegram 明确拒绝失败: %w", member.TGUserID, err)
					}
					failed++
					if isGlobalFlood(inviteErr) || strings.Contains(errStr, "user_restricted") {
						waitSec := extractFloodSeconds(errStr)
						e.markAccountRestricted(ctx, task.TenantID, accID, "invite_flood", inviteErr.Error(), waitSec)
						if saveErr := saveProgress(); saveErr != nil {
							return fmt.Errorf("账号被限流且保存进度失败: %w (原始错误: %v)", saveErr, inviteErr)
						}
						return fmt.Errorf("账号被限流, 已进入冷却: %w", inviteErr)
					}
					if isAccountBanned(inviteErr) {
						e.markAccountBanned(ctx, task.TenantID, accID, "invite_account_banned", inviteErr.Error())
						if saveErr := saveProgress(); saveErr != nil {
							return fmt.Errorf("账号封禁且保存进度失败: %w (原始错误: %v)", saveErr, inviteErr)
						}
						return fmt.Errorf("账号已被封禁, 已停止邀请: %w", inviteErr)
					}
				} else {
					// Transport errors, timeouts and disconnects do not prove
					// whether Telegram applied the request. Never auto-retry.
					if err := finishAttempt(key, repository.InviteAttemptOutcomeUnknown, inviteErr.Error()); err != nil {
						return fmt.Errorf("成员 %d 邀请结果未知，且无法保存未知状态: %w (原始错误: %v)", member.TGUserID, err, inviteErr)
					}
					failed++
					if saveErr := saveProgress(); saveErr != nil {
						return saveErr
					}
					return fmt.Errorf("成员 %d 邀请结果未知，为防止重复邀请已停止任务: %w", member.TGUserID, inviteErr)
				}
			}

			if (invited+failed)%20 == 0 {
				if err := saveProgress(); err != nil {
					return err
				}
			}
			select {
			case <-ctx.Done():
				if err := saveProgress(); err != nil {
					return err
				}
				return ctx.Err()
			case <-time.After(time.Duration(1500+securerand.IntN(2000)) * time.Millisecond):
			}
		}
	}

	if err := saveProgress(); err != nil {
		return err
	}
	taskLog.Printf("[invite] 任务 %s 完成: 成功 %d / 失败 %d", task.TaskID, invited, failed)
	return nil
}

// isDefiniteInviteRPCError only classifies explicit, non-server-error Telegram
// RPC responses as definite rejection. Transport failures and server-side 5xx
// responses remain ambiguous because Telegram may have applied the side effect.
func isDefiniteInviteRPCError(err error) bool {
	rpcErr, ok := tgerr.As(err)
	return ok && rpcErr.Code > 0 && rpcErr.Code < 500
}
