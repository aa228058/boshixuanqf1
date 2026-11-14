package handler

import (
	"context"
	"tg-sender/internal/domain"
	"tg-sender/internal/domain/account"
	"tg-sender/internal/domain/task"
	"tg-sender/internal/probe"
	"tg-sender/internal/service"
	"time"

	"github.com/google/uuid"
)

// 自动生成的零值假对象：让 handler 主干路径在无数据库时也能执行。
// 需要特定行为的测试自行覆写对应方法。

type zeroAccountSvc struct{ accountService }

func (f *zeroAccountSvc) BatchSetPool(p0 context.Context, p1 uuid.UUID, p2 []uuid.UUID, p3 string) (map[string]interface{}, error) {
	return *new(map[string]interface{}), nil
}
func (f *zeroAccountSvc) BatchUpdateProfile(p0 context.Context, p1 uuid.UUID, p2 []uuid.UUID, p3 string, p4 string, p5 string, p6 string, p7 string, p8 bool) (map[string]interface{}, error) {
	return *new(map[string]interface{}), nil
}
func (f *zeroAccountSvc) CloneProfile(p0 context.Context, p1 uuid.UUID, p2 string, p3 []uuid.UUID) (int, int, error) {
	return *new(int), *new(int), nil
}
func (f *zeroAccountSvc) Connect(p0 context.Context, p1 uuid.UUID, p2 uuid.UUID) error { return nil }
func (f *zeroAccountSvc) Create(p0 context.Context, p1 uuid.UUID, p2 string, p3 string, p4 domain.AccountLevel) (*domain.Account, error) {
	return *new(*domain.Account), nil
}
func (f *zeroAccountSvc) Delete(p0 context.Context, p1 uuid.UUID, p2 uuid.UUID) error     { return nil }
func (f *zeroAccountSvc) Disconnect(p0 context.Context, p1 uuid.UUID, p2 uuid.UUID) error { return nil }
func (f *zeroAccountSvc) ExportTransferableGroups(p0 context.Context, p1 uuid.UUID, p2 uuid.UUID) (*account.TransferGroupsExport, error) {
	return *new(*account.TransferGroupsExport), nil
}
func (f *zeroAccountSvc) GetAuthState(p0 uuid.UUID, p1 uuid.UUID) account.AuthState {
	return *new(account.AuthState)
}
func (f *zeroAccountSvc) GetAvatarPath(p0 context.Context, p1 uuid.UUID, p2 uuid.UUID) (string, error) {
	return *new(string), nil
}
func (f *zeroAccountSvc) GetByID(p0 context.Context, p1 uuid.UUID, p2 uuid.UUID) (*domain.Account, error) {
	return *new(*domain.Account), nil
}
func (f *zeroAccountSvc) GetProbeResult(p0 context.Context, p1 uuid.UUID, p2 uuid.UUID) (*probe.Result, error) {
	return *new(*probe.Result), nil
}
func (f *zeroAccountSvc) GetQuotas(p0 context.Context, p1 uuid.UUID, p2 uuid.UUID) ([]*domain.AccountQuota, error) {
	return *new([]*domain.AccountQuota), nil
}
func (f *zeroAccountSvc) GetTelegramContacts(p0 context.Context, p1 uuid.UUID, p2 uuid.UUID) ([]map[string]any, error) {
	return *new([]map[string]any), nil
}
func (f *zeroAccountSvc) GetTelegramDialogs(p0 context.Context, p1 uuid.UUID, p2 uuid.UUID) ([]map[string]any, error) {
	return *new([]map[string]any), nil
}
func (f *zeroAccountSvc) ImportSessionFiles(p0 context.Context, p1 uuid.UUID, p2 []string, p3 string) (*service.ImportResult, error) {
	return *new(*service.ImportResult), nil
}
func (f *zeroAccountSvc) ImportSessions(p0 context.Context, p1 uuid.UUID, p2 []string, p3 string) (*service.ImportResult, error) {
	return *new(*service.ImportResult), nil
}
func (f *zeroAccountSvc) List(p0 context.Context, p1 uuid.UUID, p2 string) ([]*domain.Account, error) {
	return *new([]*domain.Account), nil
}
func (f *zeroAccountSvc) MigrateAccountDC(p0 context.Context, p1 uuid.UUID, p2 uuid.UUID, p3 int32) error {
	return nil
}
func (f *zeroAccountSvc) Probe(p0 context.Context, p1 uuid.UUID, p2 uuid.UUID) (*probe.Result, error) {
	return *new(*probe.Result), nil
}
func (f *zeroAccountSvc) Reauthenticate(p0 context.Context, p1 uuid.UUID, p2 uuid.UUID) error {
	return nil
}
func (f *zeroAccountSvc) Reconnect(p0 context.Context, p1 uuid.UUID, p2 uuid.UUID) error { return nil }
func (f *zeroAccountSvc) RefreshAvatar(p0 context.Context, p1 uuid.UUID, p2 uuid.UUID) error {
	return nil
}
func (f *zeroAccountSvc) ResolvePeer(p0 context.Context, p1 uuid.UUID, p2 uuid.UUID, p3 string) (map[string]any, error) {
	return *new(map[string]any), nil
}
func (f *zeroAccountSvc) SetPoolStatus(p0 context.Context, p1 uuid.UUID, p2 uuid.UUID, p3 string) error {
	return nil
}
func (f *zeroAccountSvc) SetProxy(p0 context.Context, p1 uuid.UUID, p2 uuid.UUID, p3 string) error {
	return nil
}
func (f *zeroAccountSvc) SetRaffleEnabled(p0 context.Context, p1 uuid.UUID, p2 uuid.UUID, p3 bool) error {
	return nil
}
func (f *zeroAccountSvc) StatsByAccount(p0 context.Context, p1 uuid.UUID) (map[string]map[string]int, error) {
	return *new(map[string]map[string]int), nil
}
func (f *zeroAccountSvc) Submit2FA(p0 context.Context, p1 uuid.UUID, p2 uuid.UUID, p3 string) error {
	return nil
}
func (f *zeroAccountSvc) TestProxy(p0 context.Context, p1 uuid.UUID, p2 uuid.UUID) (map[string]interface{}, error) {
	return *new(map[string]interface{}), nil
}
func (f *zeroAccountSvc) Update(p0 context.Context, p1 uuid.UUID, p2 uuid.UUID, p3 *domain.AccountLevel, p4 *int, p5 *string) error {
	return nil
}
func (f *zeroAccountSvc) Update2FA(p0 context.Context, p1 uuid.UUID, p2 uuid.UUID, p3 string, p4 string) error {
	return nil
}
func (f *zeroAccountSvc) UpdateAccountProfile(p0 context.Context, p1 uuid.UUID, p2 uuid.UUID, p3 string, p4 string, p5 string, p6 string, p7 string, p8 string) error {
	return nil
}
func (f *zeroAccountSvc) UploadAccountAvatar(p0 context.Context, p1 uuid.UUID, p2 uuid.UUID, p3 []byte) error {
	return nil
}
func (f *zeroAccountSvc) VerifyCode(p0 context.Context, p1 uuid.UUID, p2 uuid.UUID, p3 string) error {
	return nil
}

type zeroTaskSvc struct{ taskService }

func (f *zeroTaskSvc) BatchCancel(p0 context.Context, p1 uuid.UUID, p2 []uuid.UUID) (int, int, error) {
	return *new(int), *new(int), nil
}
func (f *zeroTaskSvc) BatchDelete(p0 context.Context, p1 uuid.UUID, p2 []uuid.UUID) (int, int, error) {
	return *new(int), *new(int), nil
}
func (f *zeroTaskSvc) BioLinkScrapingEnabled(p0 uuid.UUID) bool                    { return *new(bool) }
func (f *zeroTaskSvc) Cancel(p0 context.Context, p1 uuid.UUID, p2 uuid.UUID) error { return nil }
func (f *zeroTaskSvc) Count(p0 context.Context, p1 uuid.UUID, p2 *domain.TaskStatus, p3 *uuid.UUID, p4 *domain.TaskType, p5 *time.Time) (int, error) {
	return *new(int), nil
}
func (f *zeroTaskSvc) CreateBroadcast(p0 context.Context, p1 uuid.UUID, p2 []uuid.UUID, p3 []string, p4 string, p5 *time.Time, p6 int, p7 int, p8 map[string]any) (*domain.Task, error) {
	return *new(*domain.Task), nil
}
func (f *zeroTaskSvc) CreateBroadcastWithMapping(p0 context.Context, p1 uuid.UUID, p2 []domain.AccountTargetMapping, p3 []uuid.UUID, p4 []string, p5 string, p6 *time.Time, p7 int, p8 int, p9 map[string]any) (*domain.Task, error) {
	return *new(*domain.Task), nil
}
func (f *zeroTaskSvc) CreateJoinGroups(p0 context.Context, p1 uuid.UUID, p2 uuid.UUID, p3 []string) (*domain.Task, error) {
	return *new(*domain.Task), nil
}
func (f *zeroTaskSvc) CreateMonitorScrape(p0 context.Context, p1 uuid.UUID, p2 uuid.UUID, p3 uuid.UUID, p4 string, p5 string, p6 int, p7 int) (*domain.Task, error) {
	return *new(*domain.Task), nil
}
func (f *zeroTaskSvc) CreateScheduledBroadcastWithSchedule(p0 context.Context, p1 uuid.UUID, p2 []domain.AccountTargetMapping, p3 []uuid.UUID, p4 []string, p5 string, p6 int, p7 int, p8 int, p9 bool, p10 int, p11 *time.Time, p12 map[string]any, p13 *time.Time) (*domain.Task, error) {
	return *new(*domain.Task), nil
}
func (f *zeroTaskSvc) CreateScrapeLinks(p0 context.Context, p1 uuid.UUID, p2 uuid.UUID, p3 []string, p4 int) (*domain.Task, error) {
	return *new(*domain.Task), nil
}
func (f *zeroTaskSvc) CreateScrapeMembers(p0 context.Context, p1 uuid.UUID, p2 uuid.UUID, p3 uuid.UUID, p4 map[string]any) (*domain.Task, error) {
	return *new(*domain.Task), nil
}
func (f *zeroTaskSvc) Delete(p0 context.Context, p1 uuid.UUID, p2 uuid.UUID) error { return nil }
func (f *zeroTaskSvc) DeleteTaskLogs(p0 context.Context, p1 uuid.UUID, p2 uuid.UUID) error {
	return nil
}
func (f *zeroTaskSvc) GetByID(p0 context.Context, p1 uuid.UUID, p2 uuid.UUID) (*domain.Task, error) {
	return *new(*domain.Task), nil
}
func (f *zeroTaskSvc) GetByNumber(p0 context.Context, p1 uuid.UUID, p2 int64) (*domain.Task, error) {
	return *new(*domain.Task), nil
}
func (f *zeroTaskSvc) GetLogs(p0 context.Context, p1 uuid.UUID, p2 uuid.UUID, p3 int, p4 int) ([]*domain.TaskLog, error) {
	return *new([]*domain.TaskLog), nil
}
func (f *zeroTaskSvc) GetLogsBefore(p0 context.Context, p1 uuid.UUID, p2 uuid.UUID, p3 uuid.UUID, p4 int) ([]*domain.TaskLog, error) {
	return *new([]*domain.TaskLog), nil
}
func (f *zeroTaskSvc) GetLogsBeforeInRange(p0 context.Context, p1 uuid.UUID, p2 uuid.UUID, p3 uuid.UUID, p4 *time.Time, p5 *time.Time, p6 int) ([]*domain.TaskLog, error) {
	return *new([]*domain.TaskLog), nil
}
func (f *zeroTaskSvc) GetLogsByAccount(p0 context.Context, p1 uuid.UUID, p2 uuid.UUID, p3 int, p4 int) ([]*domain.TaskLog, int, error) {
	return *new([]*domain.TaskLog), *new(int), nil
}
func (f *zeroTaskSvc) GetLogsInRange(p0 context.Context, p1 uuid.UUID, p2 uuid.UUID, p3 *time.Time, p4 *time.Time, p5 int, p6 int) ([]*domain.TaskLog, error) {
	return *new([]*domain.TaskLog), nil
}
func (f *zeroTaskSvc) LeaveAllGroups(p0 context.Context, p1 uuid.UUID, p2 uuid.UUID) (*task.LeaveAllResult, error) {
	return *new(*task.LeaveAllResult), nil
}
func (f *zeroTaskSvc) LinkScrapingEnabled(p0 uuid.UUID) bool { return *new(bool) }
func (f *zeroTaskSvc) List(p0 context.Context, p1 uuid.UUID, p2 *domain.TaskStatus, p3 *uuid.UUID, p4 *domain.TaskType, p5 bool, p6 *time.Time, p7 int, p8 int) ([]*domain.Task, error) {
	return *new([]*domain.Task), nil
}
func (f *zeroTaskSvc) ListScrapedLinks(p0 context.Context, p1 uuid.UUID, p2 uuid.UUID, p3 time.Time, p4 int, p5 int) ([]*domain.ScrapedLink, int, error) {
	return *new([]*domain.ScrapedLink), *new(int), nil
}
func (f *zeroTaskSvc) Pause(p0 context.Context, p1 uuid.UUID, p2 uuid.UUID) error { return nil }
func (f *zeroTaskSvc) RaffleResultClear(p0 context.Context, p1 uuid.UUID) error   { return nil }
func (f *zeroTaskSvc) RaffleResultList(p0 context.Context, p1 uuid.UUID) ([]map[string]interface{}, error) {
	return *new([]map[string]interface{}), nil
}
func (f *zeroTaskSvc) Resume(p0 context.Context, p1 uuid.UUID, p2 uuid.UUID) error { return nil }
func (f *zeroTaskSvc) Retry(p0 context.Context, p1 uuid.UUID, p2 uuid.UUID, p3 string) error {
	return nil
}
func (f *zeroTaskSvc) SetBioLinkScraping(p0 uuid.UUID, p1 bool)                                {}
func (f *zeroTaskSvc) SetLinkScraping(p0 context.Context, p1 uuid.UUID, p2 uuid.UUID, p3 bool) {}
func (f *zeroTaskSvc) TaskLogStatsByAccount(p0 context.Context, p1 uuid.UUID, p2 uuid.UUID) (map[string]map[string]int, error) {
	return *new(map[string]map[string]int), nil
}
func (f *zeroTaskSvc) Update(p0 context.Context, p1 uuid.UUID, p2 uuid.UUID, p3 []uuid.UUID, p4 []string, p5 map[string]any) error {
	return nil
}
func (f *zeroTaskSvc) ValidateTargets(p0 context.Context, p1 uuid.UUID, p2 []string) ([]string, []map[string]string) {
	return *new([]string), *new([]map[string]string)
}
