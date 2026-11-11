package task

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"tg-sender/internal/domain"
	securerand "tg-sender/internal/secure"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/mtgo-labs/mtgo/telegram"
	"github.com/mtgo-labs/mtgo/tg"
)

// classifyJoinError 把 Telegram 拒绝原因细化成可读错误码（原 join_failed 吞掉真因）
func classifyJoinError(err error) string {
	if err == nil {
		return ""
	}
	s := strings.ToLower(err.Error())
	switch {
	case strings.Contains(s, "invite_hash_expired"):
		return "invite_expired"
	case strings.Contains(s, "invite_hash_empty"), strings.Contains(s, "invite_hash_invalid"):
		return "invite_invalid"
	case strings.Contains(s, "username_invalid"), strings.Contains(s, "username_occupied"):
		return "username_invalid"
	case strings.Contains(s, "peer_flood"):
		return "peer_flood"
	case strings.Contains(s, "too much"), strings.Contains(s, "channels_too_much"):
		return "too_many_groups"
	case strings.Contains(s, "channel_private"), strings.Contains(s, "chat_private"):
		return "group_private"
	case strings.Contains(s, "user_not_participant"):
		return "not_invited"
	case strings.Contains(s, "banned"), strings.Contains(s, "kicked"):
		return "banned_from_group"
	case strings.Contains(s, "flood"):
		return "join_flood_wait"
	default:
		return "join_failed"
	}
}

func isJoinCheckpointStatus(status string) bool {
	return status == "success" || status == "pending_approval"
}

func (e *Engine) executeJoinGroups(ctx context.Context, task *domain.Task) {
	total := len(task.TargetIDs)
	done := 0
	failed := 0
	joinedTargetLinks := make([]string, 0, total)

	if len(task.AccountIDs) == 0 {
		if err := e.repo.UpdateStatusOwned(ctx, task.TenantID, task.TaskID, e.taskLeaseID(task), domain.TaskStatusFailed); err != nil {
			taskLog.Printf("[task] UpdateStatus failed: %v", err)
		}
		return
	}

	// 加群间隔: 租户可在创建任务时设置 最小/最大 间隔(分钟), 每次在区间内随机取值;
	// 未设置时退回全局配置 JoinIntervalMinutes(默认2分钟) + 0~50% 抖动
	intervalMin := time.Duration(0)
	intervalMax := time.Duration(0)
	if v, ok := task.Payload["interval_min_minutes"].(float64); ok && v > 0 {
		intervalMin = time.Duration(v * float64(time.Minute))
	}
	if v, ok := task.Payload["interval_max_minutes"].(float64); ok && v > 0 {
		intervalMax = time.Duration(v * float64(time.Minute))
	}
	if intervalMax > 0 && intervalMax < intervalMin {
		intervalMax = intervalMin
	}

	joinIntervalMinutes := e.cfg.JoinIntervalMinutes
	if joinIntervalMinutes <= 0 {
		joinIntervalMinutes = 2
	}
	joinInterval := time.Duration(joinIntervalMinutes) * time.Minute

	// R108 修复: accID 声明(此前被误删, join 全程单账号执行)
	accID := task.AccountIDs[0]

	// Check account status before using it
	acc, err := e.accEngine.GetByID(ctx, task.TenantID, accID)
	if err != nil || acc == nil || (acc.Status == domain.AccountStatusBanned || acc.Status == domain.AccountStatusRestricted || acc.Status == domain.AccountStatusPendingAuth) {
		status := "unknown"
		if acc != nil {
			status = string(acc.Status)
		}
		taskLog.Printf("[join] skip account %s: status=%s (not active)", accID, status)
		if err := e.repo.UpdateStatusOwned(ctx, task.TenantID, task.TaskID, e.taskLeaseID(task), domain.TaskStatusFailed); err != nil {
			taskLog.Printf("[task] UpdateStatus failed: %v", err)
		}
		return
	}
	if acc.BlockedUntil != nil && acc.BlockedUntil.After(time.Now().UTC()) {
		taskLog.Printf("[join] skip account %s: blocked_until=%s", accID, acc.BlockedUntil.Format(time.RFC3339))
		if err := e.repo.UpdateStatusOwned(ctx, task.TenantID, task.TaskID, e.taskLeaseID(task), domain.TaskStatusFailed); err != nil {
			taskLog.Printf("[task] UpdateStatus failed: %v", err)
		}
		return
	}

	client, err := e.accEngine.GetClient(ctx, task.TenantID, accID)
	if err != nil {
		if err := e.repo.UpdateStatusOwned(ctx, task.TenantID, task.TaskID, e.taskLeaseID(task), domain.TaskStatusFailed); err != nil {
			taskLog.Printf("[task] UpdateStatus failed: %v", err)
		}
		return
	}
	defer e.accEngine.ReleaseClient(task.TenantID, accID, client)

	completedTargets, checkpointErr := e.logRepo.ListCompletedJoinTargets(ctx, task.TenantID, task.TaskID, accID, task.TargetIDs)
	if checkpointErr != nil {
		taskLog.Printf("[join] 读取续传检查点失败，停止任务 task=%s: %v", task.TaskID, checkpointErr)
		if statusErr := e.repo.UpdateStatusOwned(ctx, task.TenantID, task.TaskID, e.taskLeaseID(task), domain.TaskStatusFailed); statusErr != nil {
			taskLog.Printf("[join] 标记检查点读取失败任务失败时出错 task=%s: %v", task.TaskID, statusErr)
		}
		return
	}
	for _, link := range task.TargetIDs {
		if completedTargets[link] {
			done++
		}
	}
	// Reset even when no target has succeeded yet; otherwise old failed counts
	// from a previous attempt remain visible during the resumed run.
	resumedProgress := domain.TaskProgress{Total: total, Done: done}
	if err := e.repo.UpdateProgressExact(ctx, task.TenantID, task.TaskID, resumedProgress); err != nil {
		taskLog.Printf("[join] 恢复检查点进度写入失败 task=%s: %v", task.TaskID, err)
		if statusErr := e.repo.UpdateStatusOwned(ctx, task.TenantID, task.TaskID, e.taskLeaseID(task), domain.TaskStatusFailed); statusErr != nil {
			taskLog.Printf("[join] 标记进度恢复失败任务失败时出错 task=%s: %v", task.TaskID, statusErr)
		}
		return
	}
	e.publishProgress(task.TenantID, task.TaskID, resumedProgress)

	for _, link := range task.TargetIDs {
		select {
		case <-ctx.Done():
			return
		case <-e.stopCh:
			if err := e.repo.UpdateStatusOwned(ctx, task.TenantID, task.TaskID, e.taskLeaseID(task), domain.TaskStatusPaused); err != nil {
				taskLog.Printf("[task] UpdateStatus failed: %v", err)
			}
			return
		default:
		}
		// Successful joins and persisted approval requests are durable checkpoints.
		// Failed targets remain eligible for retry.
		if completedTargets[link] {
			continue
		}

		status := "success"
		var errCode string
		var errMsg string // 2026-09-22: Telegram 拒绝原文, 落 task_logs.message, 前端详情可见
		floodHit := false
		floodWaitSec := 0 // 命中限流时记录 Telegram 要求的等待秒数

		// Distinguish private invite link vs public group link
		if strings.Contains(link, "+") || strings.Contains(link, "joinchat") {
			// Private group: use ImportChatInvite with hash
			// Fail closed when the approval store cannot be checked: otherwise a retry
			// could submit the same private join request again after an earlier request.
			if e.approvalRepo == nil {
				status = "failed"
				errCode = "approval_repository_unavailable"
				errMsg = "pending approval repository is unavailable"
				goto logAndContinue
			}
			existing, lookupErr := e.approvalRepo.GetByAccountAndLink(ctx, accID, link)
			if lookupErr == nil && existing != nil {
				status = "pending_approval"
				errCode = "already_pending"
				goto logAndContinue
			}
			if lookupErr != nil && !errors.Is(lookupErr, pgx.ErrNoRows) {
				status = "failed"
				errCode = "approval_lookup_failed"
				errMsg = lookupErr.Error()
				goto logAndContinue
			}
			hash := extractInviteHash(link)
			req := &tg.MessagesImportChatInviteRequest{Hash: hash}
			// 协议号执行任务可能在此永久阻塞（网络层卡死），加超时让错误可见
			invokeCtx, invokeCancel := context.WithTimeout(ctx, 90*time.Second)
			result, joinErr := client.Invoke(invokeCtx, req)
			invokeCancel()
			if joinErr != nil {
				if isInviteRequestSent(joinErr) {
					// Join request automatically sent by Telegram server
					status = "pending_approval"
					// The Telegram side effect may already have happened. Persist the
					// approval record with an independent bounded context so task cancellation
					// cannot erase the only durable evidence of the pending request.
					approvalCtx, approvalCancel := context.WithTimeout(context.Background(), 5*time.Second)
					approvalErr := e.approvalRepo.Create(approvalCtx, &domain.PendingApproval{
						TaskID:     task.TaskID,
						TenantID:   task.TenantID,
						AccountID:  accID,
						ChatID:     0, // private link, chat_id unknown until approved
						AccessHash: 0,
						TargetLink: link,
						Status:     "pending",
						MaxChecks:  12,
					})
					approvalCancel()
					if approvalErr != nil {
						status = "failed"
						errCode = "approval_record_create_failed"
						errMsg = approvalErr.Error()
						log.Printf("[join] 创建待审批记录失败: %v", approvalErr)
					}
					// A pending private invite has no confirmed Telegram chat identity yet.
					// Keep only the PendingApproval row; create the groups row after membership is confirmed.
				} else if isAlreadyMember(joinErr) {
					status = "success"
					errCode = "already_member"
				} else {
					status = "failed"
					errCode = "join_failed"
					if isGlobalFlood(joinErr) {
						floodHit = true
						floodWaitSec = extractFloodSeconds(strings.ToLower(joinErr.Error()))
						e.markAccountRestricted(ctx, task.TenantID, accID, "join_flood_wait", joinErr.Error(), floodWaitSec)
					} else {
						e.recordRiskEvent(ctx, task.TenantID, accID, "join_fail", "medium", joinErr.Error())
					}
				}
			} else {
				// Direct join success: extract group info from Updates
				if updates, ok := result.(*tg.Updates); ok && e.groupRepo != nil {
					e.extractAndSaveGroupFromUpdates(ctx, task, updates, link)
				} else if !ok {
					taskLog.Printf("[join] ImportChatInvite result is not *tg.Updates, type: %T", result)
				}
			}
		} else {
			// Public group: ResolvePeer → get chat ID → JoinChannel
			peer, resolveErr := client.ResolvePeer(ctx, link)
			if resolveErr != nil {
				status = "failed"
				errCode = "resolve_failed"
			} else {
				var chatID int64
				var accessHash int64
				var gtype string
				var username string
				switch p := peer.(type) {
				case *tg.InputPeerChat:
					chatID = p.ChatID
					gtype = "group"
				case *tg.InputPeerChannel:
					chatID = p.ChannelID
					accessHash = p.AccessHash
					gtype = "channel"
				default:
					status = "failed"
					errCode = "not a group"
				}
				if chatID != 0 {
					// Do not attempt a Telegram side effect when the local membership
					// lookup failed; only a confirmed not-found result is safe to proceed.
					if e.groupRepo == nil {
						status = "failed"
						errCode = "group_repository_unavailable"
						goto logAndContinue
					}
					existingGroup, groupLookupErr := e.groupRepo.GetByTGChatID(ctx, task.TenantID, chatID)
					if groupLookupErr == nil && existingGroup != nil && existingGroup.IsJoined {
						status = "success"
						errCode = "already_joined"
						goto logAndContinue
					}
					if groupLookupErr != nil && !errors.Is(groupLookupErr, pgx.ErrNoRows) {
						status = "failed"
						errCode = "group_lookup_failed"
						errMsg = groupLookupErr.Error()
						goto logAndContinue
					}
					// A Telegram join can succeed while local persistence fails. Reuse
					// this persistence path for both fresh joins and ALREADY_PARTICIPANT
					// retries so a retry can repair the missing local group row.
					persistJoinedGroup := func() error {
						if e.groupRepo == nil {
							return fmt.Errorf("group repository is unavailable")
						}
						if strings.HasPrefix(link, "@") {
							username = link[1:]
						} else if strings.HasPrefix(link, "https://t.me/") {
							username = strings.TrimPrefix(link, "https://t.me/")
						} else if strings.HasPrefix(link, "http://t.me/") {
							username = strings.TrimPrefix(link, "http://t.me/")
						} else if !strings.Contains(link, "/") {
							username = link
						}
						existing, lookupErr := e.groupRepo.GetByTGChatID(ctx, task.TenantID, chatID)
						var groupID uuid.UUID
						if lookupErr == nil && existing != nil {
							existing.AccessHash = accessHash
							existing.IsJoined = true
							existing.Type = gtype
							if username != "" {
								existing.Username = username
							}
							if existing.JoinedAt == nil {
								now := time.Now()
								existing.JoinedAt = &now
							}
							if err := e.groupRepo.Update(ctx, existing); err != nil {
								return fmt.Errorf("update joined group: %w", err)
							}
							groupID = existing.GroupID
						} else if lookupErr != nil && !errors.Is(lookupErr, pgx.ErrNoRows) {
							return fmt.Errorf("lookup joined group: %w", lookupErr)
						} else {
							now := time.Now()
							groupID = uuid.New()
							if err := e.groupRepo.Create(ctx, &domain.Group{
								GroupID: groupID, TenantID: task.TenantID, TGChatID: chatID,
								Title: link, Username: username, Type: gtype, AccessHash: accessHash,
								IsJoined: true, JoinedAt: &now,
							}); err != nil {
								return fmt.Errorf("create joined group: %w", err)
							}
							// Create uses ON CONFLICT; re-read to obtain the canonical ID
							// if another worker inserted the same Telegram chat concurrently.
							persisted, err := e.groupRepo.GetByTGChatID(ctx, task.TenantID, chatID)
							if err != nil {
								return fmt.Errorf("reload joined group: %w", err)
							}
							groupID = persisted.GroupID
							}
						if err := e.groupRepo.UpsertAccountGroup(ctx, accID, groupID, true); err != nil {
							return fmt.Errorf("link joined group to account: %w", err)
						}
						return nil
					}
					var joinErr error
					switch p := peer.(type) {
					case *tg.InputPeerChat:
						// Regular group: treat as channel (public groups are usually channels/supergroups)
						_, joinErr = client.Invoke(ctx, &tg.ChannelsJoinChannelRequest{
							Channel: &tg.InputChannel{
								ChannelID:  p.ChatID,
								AccessHash: 0,
							},
						})
					case *tg.InputPeerChannel:
						// Channel/supergroup: use channels.joinChannel
						_, joinErr = client.Invoke(ctx, &tg.ChannelsJoinChannelRequest{
							Channel: &tg.InputChannel{
								ChannelID:  p.ChannelID,
								AccessHash: p.AccessHash,
							},
						})
					}
					if joinErr != nil {
						if isInviteRequestSent(joinErr) {
							status = "pending_approval"
							if e.approvalRepo == nil {
								status = "failed"
								errCode = "approval_repository_unavailable"
								errMsg = "pending approval repository is unavailable"
							} else {
								approvalCtx, approvalCancel := context.WithTimeout(context.Background(), 5*time.Second)
								approvalErr := e.approvalRepo.Create(approvalCtx, &domain.PendingApproval{
									TaskID:     task.TaskID,
									TenantID:   task.TenantID,
									AccountID:  accID,
									ChatID:     chatID,
									TargetLink: link,
									Status:     "pending",
									MaxChecks:  12,
								})
								approvalCancel()
								if approvalErr != nil {
									status = "failed"
									errCode = "approval_record_create_failed"
									errMsg = approvalErr.Error()
									log.Printf("[join] 创建待审批记录失败: %v", approvalErr)
								}
							}
						} else if isAlreadyMember(joinErr) {
							status = "success"
							errCode = "already_member"
							if persistErr := persistJoinedGroup(); persistErr != nil {
								status = "failed"
								errCode = "group_persist_failed"
								errMsg = persistErr.Error()
								log.Printf("[join] 已在群组中，但修复本地群组记录失败: %v", persistErr)
							}
						} else {
							status = "failed"
							errMsg = joinErr.Error()
							errCode = classifyJoinError(joinErr)
							errStr := strings.ToLower(joinErr.Error())
							switch {
							case isGlobalFlood(joinErr):
								waitSec := extractFloodSeconds(errStr)
								if waitSec >= 60 {
									e.markAccountRestricted(ctx, task.TenantID, accID, "join_flood_wait", joinErr.Error(), extractFloodSeconds(strings.ToLower(joinErr.Error())))
								}
							case isAccountBanned(joinErr):
								e.markAccountBanned(ctx, task.TenantID, accID, "account_banned", joinErr.Error())
							default:
								e.recordRiskEvent(ctx, task.TenantID, accID, "join_fail", "medium", joinErr.Error())
							}
						}
					} else {
						if persistErr := persistJoinedGroup(); persistErr != nil {
							status = "failed"
							errCode = "group_persist_failed"
							errMsg = persistErr.Error()
							log.Printf("[join] 加群成功但保存本地群组记录失败: %v", persistErr)
						}
					}
					}
				}
			}

	logAndContinue:
		joinLog := &domain.TaskLog{
			LogID:      uuid.New(),
			TaskID:     task.TaskID,
			TenantID:   task.TenantID,
			AccountID:  accID,
			TargetID:   link,
			Action:     "join_group",
			Status:     status,
			ErrorCode:  errCode,
			Message:    errMsg,
			ExecutedAt: time.Now(),
		}
		// A Telegram join/request may already have succeeded. Persist its checkpoint
		// even if pause/cancel has cancelled the worker context, otherwise a restart
		// could repeat the external side effect.
		persistCtx, persistCancel := context.WithTimeout(context.Background(), 5*time.Second)
		if err := e.logRepo.Create(persistCtx, joinLog); err != nil {
			log.Printf("[join] 创建检查点日志失败，停止任务以避免重复加群: %v", err)
			if statusErr := e.repo.UpdateStatusOwned(persistCtx, task.TenantID, task.TaskID, e.taskLeaseID(task), domain.TaskStatusFailed); statusErr != nil {
				log.Printf("[join] 检查点写入失败后标记任务失败也失败: %v", statusErr)
			}
			persistCancel()
			return
		}
		persistCancel()
		e.publishLog(task.TenantID, task.TaskID, joinLog)
		if status == "success" {
			joinedTargetLinks = append(joinedTargetLinks, link)
		}

		// Only a confirmed invalid/expired Telegram URL changes that URL's validity.
		// Other aliases for the same group remain independently usable.
		if status == "failed" && e.systemGroupRepo != nil &&
			(errCode == "invite_expired" || errCode == "invite_invalid" || errCode == "username_invalid") {
			if merr := e.systemGroupRepo.UpdateLinkValidityByLink(ctx, link, "invalid"); merr != nil {
				log.Printf("[join] 标记群链接失效失败 %s: %v", link, merr)
			}
		}

		// Accumulate progress locally, flush at end of loop
		progressDelta := domain.TaskProgress{Done: 0, Failed: 0, CurrentTarget: link}
		if status == "success" {
			progressDelta.Done = 1
		} else if status == "pending_approval" {
			// pending_approval: join request sent successfully, awaiting admin approval
			// Count as done since the user's action (sending request) is complete
			progressDelta.Done = 1
			progressDelta.Failed = 0
		} else {
			progressDelta.Failed = 1
		}
		done += progressDelta.Done
		failed += progressDelta.Failed

		// 实时进度: 每个目标处理完立即上报(前端进度面板实时可见)
		liveProgress := domain.TaskProgress{Total: total, Done: done, Failed: failed, CurrentTarget: link}
		if err := e.repo.UpdateProgressExact(ctx, task.TenantID, task.TaskID, liveProgress); err != nil {
			log.Printf("[join] 更新进度失败: %v", err)
		}
		e.publishProgress(task.TenantID, task.TaskID, liveProgress)

		// 触发频繁限制(FloodWait): 不原地睡 1 小时（会占死 worker 槽位和 client
		// 连接，3 个 worker 撞 3 次 flood 队列即停摆）。改为等 Telegram 要求的时长
		// （上限 10 分钟）后继续；期间任务被取消则挂起由超时回收。
		if floodHit {
			waitDur := time.Duration(floodWaitSec) * time.Second
			if waitDur <= 0 || waitDur > 10*time.Minute {
				waitDur = 10 * time.Minute
			}
			waitLog := &domain.TaskLog{LogID: uuid.New(), TaskID: task.TaskID, TenantID: task.TenantID, TargetID: link, Action: "join_group", Status: "info", ErrorCode: "flood_wait", Message: fmt.Sprintf("触发频繁限制，等待 %d 秒后继续", int(waitDur.Seconds())), ExecutedAt: time.Now()}
			if err := e.logRepo.Create(ctx, waitLog); err != nil {
				log.Printf("[join] 创建日志失败: %v", err)
			}
			e.publishLog(task.TenantID, task.TaskID, waitLog)
			select {
			case <-ctx.Done():
				return
			case <-time.After(waitDur):
			}
		} else {
			waitDur := joinInterval
			if intervalMax > 0 {
				// 租户自定义区间: [intervalMin, intervalMax] 随机
				span := intervalMax - intervalMin
				if span < 0 {
					span = 0
				}
				waitDur = intervalMin + time.Duration(securerand.Int63n(int64(span+1)))
			} else if joinInterval > 0 {
				waitDur = joinInterval + time.Duration(securerand.Int63n(int64(joinInterval/2)))
			}
			select {
			case <-ctx.Done():
				return
			case <-time.After(waitDur):
			}
		}
	}

	// Set final exact progress (single DB write instead of per-target)
	// Use actual 'done' count (not total-failed) to correctly reflect pending_approval states
	finalProgress := domain.TaskProgress{Total: total, Done: done, Failed: failed}
	if err := e.repo.UpdateProgressExact(ctx, task.TenantID, task.TaskID, finalProgress); err != nil {
		log.Printf("[join] 更新进度失败: %v", err)
	}
	e.publishProgress(task.TenantID, task.TaskID, finalProgress)

	// Enrich only confirmed memberships. Resolving a failed target is not proof
	// that this account joined it; never mark failed or pending approvals as joined.
	if e.groupRepo != nil && len(joinedTargetLinks) > 0 {
		syncTask := *task
		syncTask.TargetIDs = joinedTargetLinks
		e.syncJoinedGroups(ctx, &syncTask, accID, client)
	}

	if failed == total && total > 0 {
		if err := e.repo.UpdateStatusOwned(ctx, task.TenantID, task.TaskID, e.taskLeaseID(task), domain.TaskStatusFailed); err != nil {
			taskLog.Printf("[task] UpdateStatus failed: %v", err)
		}
	} else {
		if err := e.repo.UpdateStatusOwned(ctx, task.TenantID, task.TaskID, e.taskLeaseID(task), domain.TaskStatusCompleted); err != nil {
			taskLog.Printf("[task] UpdateStatus failed: %v", err)
		}
	}
	// Approval checks may have resolved while this worker was still processing targets.
	// Finalize once the worker has written its terminal state and no approvals remain.
	if _, err := e.repo.FinalizeTaskAfterApprovals(ctx, task.TenantID, task.TaskID); err != nil {
		taskLog.Printf("[task] finalize pending approvals failed: %v", err)
	}
}

// syncJoinedGroups resolves joined group info and upserts to groups table.
// Saves access_hash for channels/supergroups to enable manual peer construction later.
func (e *Engine) syncJoinedGroups(ctx context.Context, task *domain.Task, accID uuid.UUID, client *telegram.Client) {
	for _, link := range task.TargetIDs {
		var chatID int64
		var title string
		var username string
		var gtype string
		var accessHash int64
		var memberCount int
		freeSpeechStatus := "unknown"

		if strings.Contains(link, "+") || strings.Contains(link, "joinchat") {
			// Private link: try to resolve by hash after joining
			// Retry up to 3 times with backoff because cache may not sync immediately
			hash := extractInviteHash(link)
			var peer any
			var resolveErr error
			for attempt := 0; attempt < 3; attempt++ {
				if attempt > 0 {
					select {
					case <-ctx.Done():
						return
					case <-time.After(time.Duration(attempt) * 2 * time.Second):
					}
				}
				peer, resolveErr = client.ResolvePeer(ctx, "t.me/+"+hash)
				if resolveErr == nil {
					break
				}
			}
			if resolveErr != nil {
				continue
			}
			switch p := peer.(type) {
			case *tg.InputPeerChat:
				chatID = p.ChatID
				gtype = "group"
			case *tg.InputPeerChannel:
				chatID = p.ChannelID
				accessHash = p.AccessHash
				gtype = "channel"
			}
		} else {
			// Public link: resolve to get chat ID and extract username
			peer, err := client.ResolvePeer(ctx, link)
			if err != nil {
				continue
			}
			switch p := peer.(type) {
			case *tg.InputPeerChat:
				chatID = p.ChatID
				gtype = "group"
			case *tg.InputPeerChannel:
				chatID = p.ChannelID
				accessHash = p.AccessHash
				gtype = "channel"
			}
			// Extract username from public link (t.me/username or @username)
			if strings.HasPrefix(link, "@") {
				username = link[1:]
			} else if strings.HasPrefix(link, "https://t.me/") {
				username = strings.TrimPrefix(link, "https://t.me/")
			} else if strings.HasPrefix(link, "http://t.me/") {
				username = strings.TrimPrefix(link, "http://t.me/")
			} else if !strings.Contains(link, "/") {
				username = link
			}
		}

		if chatID == 0 {
			continue
		}

		// Read shared group metadata from Telegram after the account has joined.
		// DefaultBannedRights describes the group-wide default, not this account's mute state.
		metadataResolved := false
		if gtype == "channel" && accessHash != 0 {
			fullReq := &tg.ChannelsGetFullChannelRequest{
				Channel: &tg.InputChannel{ChannelID: chatID, AccessHash: accessHash},
			}
			if fullResult, err := client.Invoke(ctx, fullReq); err == nil {
				if full, ok := fullResult.(*tg.MessagesChatFull); ok {
					for _, item := range full.Chats {
						if ch, ok := item.(*tg.Channel); ok && int64(ch.ID) == chatID {
							title = ch.Title
							memberCount = int(ch.ParticipantsCount)
							metadataResolved = strings.TrimSpace(title) != ""
							if ch.Megagroup {
								gtype = "supergroup"
								if ch.DefaultBannedRights == nil || !ch.DefaultBannedRights.SendMessages {
									freeSpeechStatus = "allowed"
								} else {
									freeSpeechStatus = "restricted"
								}
							} else {
								gtype = "channel"
							}
							break
						}
					}
				}
			}
		} else if gtype == "group" {
			fullReq := &tg.MessagesGetFullChatRequest{ChatID: chatID}
			if fullResult, err := client.Invoke(ctx, fullReq); err == nil {
				if full, ok := fullResult.(*tg.MessagesChatFull); ok {
					for _, item := range full.Chats {
						if ch, ok := item.(*tg.Chat); ok && ch.ID == chatID {
							title = ch.Title
							memberCount = int(ch.ParticipantsCount)
							metadataResolved = strings.TrimSpace(title) != ""
							if ch.DefaultBannedRights == nil || !ch.DefaultBannedRights.SendMessages {
								freeSpeechStatus = "allowed"
							} else {
								freeSpeechStatus = "restricted"
							}
							break
						}
					}
				}
			}
		}
		// Keep the type derived from ResolvePeer when metadata lookup fails.
		// "unknown" violates the groups.type constraint and loses useful peer data.
		// Fallback to link for the tenant-local record only; never publish this as a shared group title.
		if title == "" {
			title = link
		}

		// Any successful tenant-account membership enriches the system-wide shared group library.
		// Account-specific mute state is intentionally not read or persisted here.
		if e.systemGroupRepo != nil && metadataResolved && strings.TrimSpace(title) != "" {
			shared := &domain.SystemGroup{
				Source: "joined", Title: title, Username: username, TmeLink: link,
				GType: gtype, MemberCount: memberCount, TelegramChatID: chatID,
				Status: "active", ContributorTenantID: task.TenantID,
			}
			if err := e.systemGroupRepo.UpsertJoinedGroup(ctx, shared, freeSpeechStatus); err != nil {
				log.Printf("[join] 共享群组库补齐失败 chat=%d: %v", chatID, err)
			}
		}

		// Upsert group record with access_hash
		now := time.Now()
		g := &domain.Group{
			GroupID:    uuid.New(),
			TenantID:   task.TenantID,
			TGChatID:   chatID,
			Title:      title,
			Username:   username,
			Type:       gtype,
			AccessHash: accessHash,
			IsJoined:   true,
			JoinedAt:   &now,
		}
		// Check if group already exists by Telegram chat ID + tenant.
		existing, lookupErr := e.groupRepo.GetByTGChatID(ctx, task.TenantID, chatID)
		var groupID uuid.UUID
		if lookupErr == nil && existing != nil {
			existing.AccessHash = accessHash
			existing.IsJoined = true
			if username != "" {
				existing.Username = username
			}
			if err := e.groupRepo.Update(ctx, existing); err != nil {
				log.Printf("[join] 更新群组失败: %v", err)
				continue
			}
			groupID = existing.GroupID
		} else if errors.Is(lookupErr, pgx.ErrNoRows) {
			if err := e.groupRepo.Create(ctx, g); err != nil {
				log.Printf("[join] 创建群组失败: %v", err)
				continue
			}
			persisted, err := e.groupRepo.GetByTGChatID(ctx, task.TenantID, chatID)
			if err != nil {
				log.Printf("[join] 重载群组失败: %v", err)
				continue
			}
			groupID = persisted.GroupID
		} else {
			log.Printf("[join] 查询群组失败: %v", lookupErr)
			continue
		}
		if err := e.groupRepo.UpsertAccountGroup(ctx, accID, groupID, true); err != nil {
			log.Printf("[join] 关联群组到账号失败 account=%s group=%s: %v", accID, groupID, err)
		}
	}
}

// defaultFreeSpeechStatus reads only the group-wide default banned rights.
func defaultFreeSpeechStatus(rights *tg.ChatBannedRights) string {
	if rights == nil || !rights.SendMessages {
		return "allowed"
	}
	return "restricted"
}

// extractAndSaveGroupFromUpdates extracts group info from Updates (ImportChatInvite result)
// and saves/updates the groups table. Handles both *tg.Chat and *tg.Channel.
func (e *Engine) extractAndSaveGroupFromUpdates(ctx context.Context, task *domain.Task, updates *tg.Updates, sourceLink string) {
	for _, chat := range updates.Chats {
		var chatID int64
		var sharedChatID int64
		var title string
		var gtype string
		var accessHash int64
		var memberCount int
		var username string
		freeSpeechStatus := "unknown"

		switch c := chat.(type) {
		case *tg.Chat:
			chatID = c.ID
			sharedChatID = int64(c.ID)
			title = c.Title
			memberCount = int(c.ParticipantsCount)
			gtype = "group"
			freeSpeechStatus = defaultFreeSpeechStatus(c.DefaultBannedRights)
		case *tg.Channel:
			// Convert internal channel ID to external format (-100{channel_id})
			chatID = -(1000000000000 + c.ID)
			sharedChatID = int64(c.ID)
			title = c.Title
			username = c.Username
			memberCount = int(c.ParticipantsCount)
			accessHash = c.AccessHash
			if c.Megagroup {
				gtype = "supergroup"
				freeSpeechStatus = defaultFreeSpeechStatus(c.DefaultBannedRights)
			} else {
				gtype = "channel"
			}
		default:
			continue
		}

		if chatID == 0 {
			continue
		}

		sharedLink := sourceLink
		if username != "" {
			sharedLink = "https://t.me/" + username
		}
		if e.systemGroupRepo != nil && strings.TrimSpace(title) != "" {
			shared := &domain.SystemGroup{
				Source: "joined", Title: title, Username: username, TmeLink: sharedLink,
				GType: gtype, MemberCount: memberCount, TelegramChatID: sharedChatID,
				Status: "active", ContributorTenantID: task.TenantID,
			}
			if err := e.systemGroupRepo.UpsertJoinedGroup(ctx, shared, freeSpeechStatus); err != nil {
				log.Printf("[join] 邀请加入后补齐共享群库失败 chat=%d: %v", sharedChatID, err)
			}
		}

		now := time.Now()
		g := &domain.Group{
			GroupID:    uuid.New(),
			TenantID:   task.TenantID,
			TGChatID:   chatID,
			Title:      title,
			Type:       gtype,
			AccessHash: accessHash,
			IsJoined:   true,
			JoinedAt:   &now,
		}

		existing, lookupErr := e.groupRepo.GetByTGChatID(ctx, task.TenantID, chatID)
		var groupID uuid.UUID
		if lookupErr == nil && existing != nil {
			existing.AccessHash = accessHash
			existing.IsJoined = true
			existing.Title = title
			if err := e.groupRepo.Update(ctx, existing); err != nil {
				log.Printf("[join] 更新群组失败: %v", err)
				continue
			}
			groupID = existing.GroupID
		} else if errors.Is(lookupErr, pgx.ErrNoRows) {
			if err := e.groupRepo.Create(ctx, g); err != nil {
				log.Printf("[join] 创建群组失败: %v", err)
				continue
			}
			persisted, err := e.groupRepo.GetByTGChatID(ctx, task.TenantID, chatID)
			if err != nil {
				log.Printf("[join] 重载群组失败: %v", err)
				continue
			}
			groupID = persisted.GroupID
		} else {
			log.Printf("[join] 查询群组失败: %v", lookupErr)
			continue
		}
		if err := e.groupRepo.UpsertAccountGroup(ctx, task.AccountIDs[0], groupID, true); err != nil {
			log.Printf("[join] 邀请加入后关联群组到账号失败 account=%s group=%s: %v", task.AccountIDs[0], groupID, err)
		}
	}
}
