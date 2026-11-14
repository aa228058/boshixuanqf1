package handler

// 这一组窄接口只为"handler 可脱离数据库单测"而存在：
// handler 结构体字段用接口类型，生产装配仍传具体 service（自动满足），
// 测试则可以注入只实现用到方法的假对象。
// 每加一个接口，就相当于把那批 handler 从"必须有真实 PG 才能测"里解放出来。

import (
	"context"
	"encoding/json"
	"time"

	"tg-sender/internal/domain"
	"tg-sender/internal/domain/account"
	"tg-sender/internal/domain/task"
	"tg-sender/internal/probe"
	"tg-sender/internal/service"

	"github.com/google/uuid"
)

type dashboardService interface {
	Stats(ctx context.Context, tenantID uuid.UUID) (map[string]any, error)
	StatsForAccount(ctx context.Context, tenantID, accountID uuid.UUID) (map[string]any, error)
	SendCalendar(ctx context.Context, tenantID uuid.UUID, months int) (map[string]int, error)
	AuditLogs(ctx context.Context, tenantID uuid.UUID, limit, offset int, startTime, endTime string) ([]*domain.AuditLog, error)
	RiskEvents(ctx context.Context, tenantID uuid.UUID, limit, offset int, startTime, endTime string) ([]*domain.RiskEvent, error)
}

type groupService interface {
	List(ctx context.Context, tenantID uuid.UUID, page, pageSize int) ([]*domain.Group, error)
	ListByAccount(ctx context.Context, tenantID, accountID uuid.UUID, page, pageSize int) ([]*domain.Group, error)
	CountByAccount(ctx context.Context, tenantID, accountID uuid.UUID) (int, error)
	GetByID(ctx context.Context, tenantID, groupID uuid.UUID) (*domain.Group, error)
	Create(ctx context.Context, tenantID uuid.UUID, tgChatID int64, title, username, gtype string) (*domain.Group, error)
	Update(ctx context.Context, tenantID, groupID uuid.UUID, title, username *string) error
	GetMembers(ctx context.Context, tenantID, groupID uuid.UUID, page, pageSize int) ([]*domain.Member, int, error)
	ListGlobalMembers(ctx context.Context, tenantID, accountID uuid.UUID) ([]map[string]any, error)
}

// accountTaskCreator 只覆盖 AccountHandler 群转移时用到的那一个方法。
type accountTaskCreator interface {
	CreateJoinGroupsWithInterval(ctx context.Context, tenantID, accountID uuid.UUID, links []string, intervalMin, intervalMax int) (*domain.Task, error)
}

// groupTaskCreator 只覆盖 GroupHandler 用到的那一个任务创建方法。
type groupTaskCreator interface {
	CreateScrapeMembers(ctx context.Context, tenantID, accountID, groupID uuid.UUID, extraPayload map[string]any) (*domain.Task, error)
}

type monitorRuleService interface {
	List(ctx context.Context, tenantID uuid.UUID) ([]*domain.MonitorRule, error)
	ListActiveTenantIDs(ctx context.Context) ([]uuid.UUID, error)
	Create(ctx context.Context, tenantID uuid.UUID, mr *domain.MonitorRule) (*domain.MonitorRule, error)
	Update(ctx context.Context, mr *domain.MonitorRule) error
	Delete(ctx context.Context, tenantID, ruleID uuid.UUID) error
	Toggle(ctx context.Context, tenantID, ruleID uuid.UUID, isActive bool) error
	GetByID(ctx context.Context, tenantID, ruleID uuid.UUID) (*domain.MonitorRule, error)
}

type settingsService interface {
	ActivateCode(ctx context.Context, tenantID uuid.UUID, code string) (*domain.Tenant, error)
	CreateAccountGroup(ctx context.Context, tenantID uuid.UUID, name, color string, accountIDs []uuid.UUID) (*domain.AccountGroup, error)
	CreateTemplate(ctx context.Context, tenantID uuid.UUID, name, content string) (*domain.MessageTemplate, error)
	DeleteAccountGroup(ctx context.Context, tenantID, groupID uuid.UUID) error
	DeleteTemplate(ctx context.Context, tenantID, templateID uuid.UUID) error
	ExportMembersCSV(ctx context.Context, tenantID uuid.UUID, groupId *uuid.UUID) ([]byte, error)
	ExportMembersJSON(ctx context.Context, tenantID uuid.UUID, groupId *uuid.UUID) ([]byte, error)
	GetAnnouncements(ctx context.Context, tenantID uuid.UUID, planID string) ([]domain.Announcement, error)
	GetMessageDecorationEnabled(ctx context.Context, tenantID uuid.UUID) (bool, error)
	GetPlans(ctx context.Context) ([]domain.Plan, error)
	GetProfile(ctx context.Context, tenantID uuid.UUID) (*domain.Tenant, error)
	GetSupport(ctx context.Context) (*domain.SupportConfig, error)
	ListAccountGroups(ctx context.Context, tenantID uuid.UUID) ([]*domain.AccountGroup, error)
	ListTemplates(ctx context.Context, tenantID uuid.UUID) ([]*domain.MessageTemplate, error)
	MarkAnnouncementRead(ctx context.Context, tenantID, announcementID uuid.UUID) error
	UpdateAccountGroup(ctx context.Context, tenantID, groupID uuid.UUID, name, color string, accountIDs []uuid.UUID) error
	UpdateMessageDecorationEnabled(ctx context.Context, tenantID uuid.UUID, enabled bool) error
	UpdateNotify(ctx context.Context, tenantID uuid.UUID, email string, settings, smtp json.RawMessage) error
	UpdateTemplate(ctx context.Context, tenantID, templateID uuid.UUID, name, content string) error
}

type adminPlaneService interface {
	AdminLogin(ctx context.Context, username, password string) (string, error)
	BatchDeleteCodes(ctx context.Context, ids []uuid.UUID) error
	BatchRevokeCodes(ctx context.Context, ids []uuid.UUID) error
	CreateAnnouncement(ctx context.Context, a *domain.Announcement) error
	CreateTenant(ctx context.Context, name, username, password, planID string) (*domain.Tenant, string, error)
	DeleteAnnouncement(ctx context.Context, id uuid.UUID) error
	DeleteTenant(ctx context.Context, tenantID uuid.UUID) error
	ExportCodes(ctx context.Context) ([]domain.ActivationCode, error)
	GenerateCodes(ctx context.Context, adminID uuid.UUID, planID string, count int, timingType string, expiresDays *int, notes string) ([]domain.ActivationCode, error)
	GetAuditLogs(ctx context.Context, page, pageSize int, startTime, endTime string) ([]*domain.AuditLog, int, error)
	GetCode(ctx context.Context, codeID uuid.UUID) (*domain.ActivationCode, error)
	GetProbe(ctx context.Context) (*domain.ProbeConfig, error)
	GetStats(ctx context.Context) (map[string]any, error)
	GetSupport(ctx context.Context) (*domain.SupportConfig, error)
	GetSystemMetrics(ctx context.Context) (map[string]any, error)
	GetTenant(ctx context.Context, tenantID uuid.UUID) (*domain.Tenant, error)
	ListAnnouncements(ctx context.Context) ([]domain.Announcement, error)
	ListCodes(ctx context.Context, status, planID string, page, pageSize int) ([]domain.ActivationCode, int, error)
	ListPlans(ctx context.Context) ([]domain.Plan, error)
	ListTenants(ctx context.Context, page, pageSize int, status, planID, search string) ([]*domain.Tenant, int, error)
	PurgeExpiredTenants(ctx context.Context, retentionDays int) (int, error)
	ResetAPIKey(ctx context.Context, tenantID uuid.UUID) (string, error)
	RevokeCode(ctx context.Context, codeID uuid.UUID) error
	UpdateAnnouncement(ctx context.Context, req *domain.Announcement) error
	UpdatePlan(ctx context.Context, planID string, updates map[string]any) error
	UpdateProbe(ctx context.Context, cfg *domain.ProbeConfig) error
	UpdateSupport(ctx context.Context, cfg *domain.SupportConfig) error
	UpdateTenant(ctx context.Context, tenantID uuid.UUID, planID, status string, maxAccounts, maxConcurrent *int, expiresAtStr *string) error
}

type accountService interface {
	BatchSetPool(ctx context.Context, tenantID uuid.UUID, accountIDs []uuid.UUID, target string) (map[string]interface{}, error)
	BatchUpdateProfile(ctx context.Context, tenantID uuid.UUID, accountIDs []uuid.UUID, firstName, lastName, about, username, remark string, randomUsername bool) (map[string]interface{}, error)
	CloneProfile(ctx context.Context, tenantID uuid.UUID, sourceUsername string, targetIDs []uuid.UUID) (int, int, error)
	Connect(ctx context.Context, tenantID, accountID uuid.UUID) error
	Create(ctx context.Context, tenantID uuid.UUID, phone string, remark string, level domain.AccountLevel) (*domain.Account, error)
	Delete(ctx context.Context, tenantID, accountID uuid.UUID) error
	Disconnect(ctx context.Context, tenantID, accountID uuid.UUID) error
	ExportTransferableGroups(ctx context.Context, tenantID, accountID uuid.UUID) (*account.TransferGroupsExport, error)
	GetAuthState(tenantID, accountID uuid.UUID) account.AuthState
	GetAvatarPath(ctx context.Context, tenantID, accountID uuid.UUID) (string, error)
	GetByID(ctx context.Context, tenantID, accountID uuid.UUID) (*domain.Account, error)
	GetProbeResult(ctx context.Context, tenantID, accountID uuid.UUID) (*probe.Result, error)
	GetQuotas(ctx context.Context, tenantID, accountID uuid.UUID) ([]*domain.AccountQuota, error)
	GetTelegramContacts(ctx context.Context, tenantID, accountID uuid.UUID) ([]map[string]any, error)
	GetTelegramDialogs(ctx context.Context, tenantID, accountID uuid.UUID) ([]map[string]any, error)
	ImportSessionFiles(ctx context.Context, tenantID uuid.UUID, paths []string, remark string) (*service.ImportResult, error)
	ImportSessions(ctx context.Context, tenantID uuid.UUID, sessionStrings []string, remark string) (*service.ImportResult, error)
	List(ctx context.Context, tenantID uuid.UUID, source string) ([]*domain.Account, error)
	MigrateAccountDC(ctx context.Context, tenantID, accountID uuid.UUID, targetDC int32) error
	Probe(ctx context.Context, tenantID, accountID uuid.UUID) (*probe.Result, error)
	Reauthenticate(ctx context.Context, tenantID, accountID uuid.UUID) error
	Reconnect(ctx context.Context, tenantID, accountID uuid.UUID) error
	RefreshAvatar(ctx context.Context, tenantID, accountID uuid.UUID) error
	ResolvePeer(ctx context.Context, tenantID, accountID uuid.UUID, identifier string) (map[string]any, error)
	SetPoolStatus(ctx context.Context, tenantID, accountID uuid.UUID, target string) error
	SetProxy(ctx context.Context, tenantID, accountID uuid.UUID, proxyURL string) error
	SetRaffleEnabled(ctx context.Context, tenantID, accountID uuid.UUID, enabled bool) error
	StatsByAccount(ctx context.Context, tenantID uuid.UUID) (map[string]map[string]int, error)
	Submit2FA(ctx context.Context, tenantID, accountID uuid.UUID, password string) error
	TestProxy(ctx context.Context, tenantID, accountID uuid.UUID) (map[string]interface{}, error)
	Update(ctx context.Context, tenantID, accountID uuid.UUID, level *domain.AccountLevel, healthScore *int, remark *string) error
	Update2FA(ctx context.Context, tenantID, accountID uuid.UUID, current, next string) error
	UpdateAccountProfile(ctx context.Context, tenantID, accountID uuid.UUID, firstName, lastName, about, username, remark, twoFA string) error
	UploadAccountAvatar(ctx context.Context, tenantID, accountID uuid.UUID, photo []byte) error
	VerifyCode(ctx context.Context, tenantID, accountID uuid.UUID, code string) error
}

type taskService interface {
	BatchCancel(ctx context.Context, tenantID uuid.UUID, taskIDs []uuid.UUID) (cancelled, failed int, err error)
	BatchDelete(ctx context.Context, tenantID uuid.UUID, taskIDs []uuid.UUID) (deleted, failed int, err error)
	BioLinkScrapingEnabled(accountID uuid.UUID) bool
	Cancel(ctx context.Context, tenantID, taskID uuid.UUID) error
	Count(ctx context.Context, tenantID uuid.UUID, status *domain.TaskStatus, accountID *uuid.UUID, taskType *domain.TaskType, since *time.Time) (int, error)
	CreateBroadcast(ctx context.Context, tenantID uuid.UUID, accountIDs []uuid.UUID, targetIDs []string, message string, scheduledAt *time.Time, intervalMin, intervalMax int, extraPayload map[string]any) (*domain.Task, error)
	CreateBroadcastWithMapping(ctx context.Context, tenantID uuid.UUID, mappings []domain.AccountTargetMapping, accountIDs []uuid.UUID, targetIDs []string, message string, scheduledAt *time.Time, intervalMin, intervalMax int, extraPayload map[string]any) (*domain.Task, error)
	CreateJoinGroups(ctx context.Context, tenantID uuid.UUID, accountID uuid.UUID, links []string) (*domain.Task, error)
	CreateMonitorScrape(ctx context.Context, tenantID, accountID, groupID uuid.UUID, matchType, matchContent string, durationMinutes, maxResults int) (*domain.Task, error)
	CreateScheduledBroadcastWithSchedule(ctx context.Context, tenantID uuid.UUID, mappings []domain.AccountTargetMapping, accountIDs []uuid.UUID, targetIDs []string, message string, loopIntervalMinutes, intervalMin, intervalMax int, isLoop bool, maxRounds int, endAt *time.Time, schedule map[string]any, firstRunAt *time.Time) (*domain.Task, error)
	CreateScrapeLinks(ctx context.Context, tenantID uuid.UUID, accountID uuid.UUID, _ []string, durationMinutes int) (*domain.Task, error)
	CreateScrapeMembers(ctx context.Context, tenantID, accountID, groupID uuid.UUID, extraPayload map[string]any) (*domain.Task, error)
	Delete(ctx context.Context, tenantID, taskID uuid.UUID) error
	DeleteTaskLogs(ctx context.Context, tenantID, taskID uuid.UUID) error
	GetByID(ctx context.Context, tenantID, taskID uuid.UUID) (*domain.Task, error)
	GetByNumber(ctx context.Context, tenantID uuid.UUID, taskNumber int64) (*domain.Task, error)
	GetLogs(ctx context.Context, tenantID, taskID uuid.UUID, page, pageSize int) ([]*domain.TaskLog, error)
	GetLogsBefore(ctx context.Context, tenantID, taskID, beforeLogID uuid.UUID, pageSize int) ([]*domain.TaskLog, error)
	GetLogsBeforeInRange(ctx context.Context, tenantID, taskID, beforeLogID uuid.UUID, from, to *time.Time, pageSize int) ([]*domain.TaskLog, error)
	GetLogsByAccount(ctx context.Context, tenantID, accountID uuid.UUID, page, pageSize int) ([]*domain.TaskLog, int, error)
	GetLogsInRange(ctx context.Context, tenantID, taskID uuid.UUID, from, to *time.Time, page, pageSize int) ([]*domain.TaskLog, error)
	LeaveAllGroups(ctx context.Context, tenantID, accountID uuid.UUID) (*task.LeaveAllResult, error)
	LinkScrapingEnabled(accountID uuid.UUID) bool
	List(ctx context.Context, tenantID uuid.UUID, status *domain.TaskStatus, accountID *uuid.UUID, taskType *domain.TaskType, hideLoops bool, since *time.Time, page, pageSize int) ([]*domain.Task, error)
	ListScrapedLinks(ctx context.Context, tenantID uuid.UUID, accountID uuid.UUID, since time.Time, page, pageSize int) ([]*domain.ScrapedLink, int, error)
	Pause(ctx context.Context, tenantID, taskID uuid.UUID) error
	RaffleResultClear(ctx context.Context, tenantID uuid.UUID) error
	RaffleResultList(ctx context.Context, tenantID uuid.UUID) ([]map[string]interface{}, error)
	Resume(ctx context.Context, tenantID, taskID uuid.UUID) error
	Retry(ctx context.Context, tenantID, taskID uuid.UUID, scope string) error
	SetBioLinkScraping(accountID uuid.UUID, enabled bool)
	SetLinkScraping(ctx context.Context, tenantID, accountID uuid.UUID, enabled bool)
	TaskLogStatsByAccount(ctx context.Context, tenantID, taskID uuid.UUID) (map[string]map[string]int, error)
	Update(ctx context.Context, tenantID, taskID uuid.UUID, accountIDs []uuid.UUID, targetIDs []string, payload map[string]any) error
	ValidateTargets(ctx context.Context, tenantID uuid.UUID, targets []string) ([]string, []map[string]string)
}
