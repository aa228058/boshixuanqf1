package backup

import (
	"compress/gzip"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"sync"
	"tg-sender/internal/safego"
	"time"

	"tg-sender/internal/config"
)

const (
	// dbBackupRetention 保留最近 N 份数据库备份
	dbBackupRetention = 7
	// dbBackupInterval 备份间隔(每小时)
	dbBackupInterval = time.Hour
)

var dbBackupOnce sync.Once

// StartDBBackup 启动数据库定时备份(pg_dump + gzip, 保留最近 7 份).
// 服务器需安装 pg_dump; 未安装时每分钟重试探测一次, 不阻塞启动.
func StartDBBackup(cfg *config.Config) {
	dbBackupOnce.Do(func() {
		dir := os.Getenv("DB_BACKUP_DIR")
		if dir == "" {
			dir = "backups/db"
		}
		// #nosec G703 -- DB_BACKUP_DIR is trusted deployment configuration, never request input; subsequent access is scoped by os.Root.
		if err := os.MkdirAll(dir, 0o700); err != nil {
			log.Printf("[db-backup] 创建备份目录失败: %v", err)
			return
		}
		safego.Go(func() { dbBackupLoop(cfg, dir, stopCh) })
	})
}

var stopCh = make(chan struct{})

// Stop 协调终止备份循环(服务关闭时调用, 审计P2-04)
// P2-69 审计修复: 原来裸 close(stopCh)——双调(优雅停机+信号处理各一次)必 panic。
var stopOnce sync.Once

func Stop() {
	stopOnce.Do(func() { close(stopCh) })
}

func dbBackupLoop(cfg *config.Config, dir string, stop <-chan struct{}) {
	for {
		if _, err := exec.LookPath("pg_dump"); err != nil {
			log.Printf("[db-backup] 未找到 pg_dump, 60s 后重试(安装: apt install postgresql-client)")
			select {
			case <-stop:
				return
			case <-time.After(time.Minute):
			}
			continue
		}
		if err := runDBDump(cfg, dir); err != nil {
			log.Printf("[db-backup] 备份失败: %v", err)
		}
		pruneOldDBBackups(dir, dbBackupRetention)
		// 顺带清理过期媒体文件(群发上传的附件, 默认保留 30 天, 防止 media 目录无限膨胀)
		pruneOldMediaFiles(30 * 24 * time.Hour)
		select {
		case <-stop:
			return
		case <-time.After(dbBackupInterval):
		}
	}
}

func runDBDump(cfg *config.Config, dir string) error {
	name := fmt.Sprintf("db_%s.sql.gz", time.Now().Format("20060102_150405"))
	path := filepath.Join(dir, name)

	root, err := os.OpenRoot(dir)
	if err != nil {
		return fmt.Errorf("open backup directory: %w", err)
	}
	defer func() {
		if err := root.Close(); err != nil {
			log.Printf("[db-backup] 关闭备份目录失败: %v", err)
		}
	}()

	f, err := root.OpenFile(name, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("create backup file: %w", err)
	}
	fileClosed := false
	defer func() {
		if !fileClosed {
			if closeErr := f.Close(); closeErr != nil {
				log.Printf("[db-backup] 关闭备份文件失败: %v", closeErr)
			}
		}
	}()

	gz := gzip.NewWriter(f)
	gzipClosed := false
	defer func() {
		if !gzipClosed {
			if closeErr := gz.Close(); closeErr != nil {
				log.Printf("[db-backup] 关闭 gzip writer 失败: %v", closeErr)
			}
		}
	}()

	// 密码经环境变量传递, 不出现在命令行参数/日志里
	env := append(os.Environ(), "PGPASSWORD="+cfg.DBPassword)
	// #nosec G204 -- fixed pg_dump executable and argument vector; no shell is invoked, and DB values are passed as arguments/env.
	cmd := exec.Command("pg_dump",
		"-h", cfg.DBHost,
		"-p", fmt.Sprintf("%d", cfg.DBPort),
		"-U", cfg.DBUser,
		"-d", cfg.DBName,
		"--no-owner", "--no-privileges",
		"-Fc", // 自定义格式, 支持 pg_restore 并行恢复
	)
	cmd.Env = env
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("stdout pipe: %w", err)
	}
	stderr, err := root.OpenFile(name+".err", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("create backup error log: %w", err)
	}
	stderrClosed := false
	defer func() {
		if !stderrClosed {
			if closeErr := stderr.Close(); closeErr != nil {
				log.Printf("[db-backup] 关闭备份错误日志失败: %v", closeErr)
			}
		}
	}()
	cmd.Stderr = stderr

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("pg_dump start: %w", err)
	}
	if _, err := io.Copy(gz, stdout); err != nil {
		waitErr := cmd.Wait()
		if waitErr != nil {
			return fmt.Errorf("copy dump: %w (pg_dump exit: %v)", err, waitErr)
		}
		return fmt.Errorf("copy dump: %w", err)
	}
	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("pg_dump exit: %w (详见 %s)", err, path+".err")
	}
	if err := gz.Close(); err != nil {
		return fmt.Errorf("finalize gzip backup: %w", err)
	}
	gzipClosed = true
	if err := f.Sync(); err != nil {
		return fmt.Errorf("sync backup file: %w", err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("close backup file: %w", err)
	}
	fileClosed = true
	if err := stderr.Close(); err != nil {
		return fmt.Errorf("close backup error log: %w", err)
	}
	stderrClosed = true
	// #nosec G706 -- path is composed from a trusted configured directory and a fixed timestamp filename.
	log.Printf("[db-backup] 备份完成: %q", path)
	return nil
}

// pruneOldMediaFiles 删除 media 目录中超过 maxAge 的文件(上传的群发附件).
// 附件发送完成后即完成任务, 文件无需长期保留; 默认 30 天兜底防误删近期文件.
func pruneOldMediaFiles(maxAge time.Duration) {
	dir := os.Getenv("MEDIA_DIR")
	if dir == "" {
		dir = "media"
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("[db-backup] 打开媒体目录失败")
		}
		return
	}
	defer func() {
		if err := root.Close(); err != nil {
			log.Printf("[db-backup] 关闭媒体目录失败: %v", err)
		}
	}()
	dirFile, err := root.Open(".")
	if err != nil {
		log.Printf("[db-backup] 读取媒体目录失败: %v", err)
		return
	}
	entries, readErr := dirFile.ReadDir(-1)
	if closeErr := dirFile.Close(); closeErr != nil {
		log.Printf("[db-backup] 关闭媒体目录句柄失败: %v", closeErr)
	}
	if readErr != nil {
		log.Printf("[db-backup] 枚举媒体目录失败: %v", readErr)
		return
	}
	cutoff := time.Now().Add(-maxAge)
	removed := 0
	for _, e := range entries {
		info, err := e.Info()
		if err != nil {
			log.Printf("[db-backup] 读取媒体文件信息失败")
			continue
		}
		if info.IsDir() || !info.ModTime().Before(cutoff) {
			continue
		}
		if err := root.Remove(e.Name()); err != nil {
			if !os.IsNotExist(err) {
				log.Printf("[db-backup] 删除过期媒体文件失败")
			}
			continue
		}
		removed++
	}
	if removed > 0 {
		log.Printf("[db-backup] 清理过期媒体文件: %d 个(>%v)", removed, maxAge)
	}
}

// pruneOldDBBackups 只保留最近 N 份, 按文件名排序(时间戳在文件名里).
func pruneOldDBBackups(dir string, keep int) {
	if keep < 0 {
		return
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		log.Printf("[db-backup] 打开备份目录失败")
		return
	}
	defer func() {
		if err := root.Close(); err != nil {
			log.Printf("[db-backup] 关闭备份目录失败: %v", err)
		}
	}()

	dirFile, err := root.Open(".")
	if err != nil {
		log.Printf("[db-backup] 读取备份目录失败: %v", err)
		return
	}
	entries, readErr := dirFile.ReadDir(-1)
	if closeErr := dirFile.Close(); closeErr != nil {
		log.Printf("[db-backup] 关闭备份目录句柄失败: %v", closeErr)
	}
	if readErr != nil {
		log.Printf("[db-backup] 枚举备份目录失败: %v", readErr)
		return
	}
	matches := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if matched, _ := filepath.Match("db_*.sql.gz", entry.Name()); matched {
			matches = append(matches, entry.Name())
		}
	}
	if len(matches) <= keep {
		return
	}
	sort.Strings(matches)
	for _, old := range matches[:len(matches)-keep] {
		if err := root.Remove(old); err != nil {
			if !os.IsNotExist(err) {
				log.Printf("[db-backup] 清理旧备份失败: %v", err)
			}
		} else {
			log.Printf("[db-backup] 清理旧备份完成")
		}
		if err := root.Remove(old + ".err"); err != nil && !os.IsNotExist(err) {
			log.Printf("[db-backup] 清理旧备份错误日志失败")
		}
	}
}
