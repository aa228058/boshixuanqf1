package account

import (
	"io"
	"log"
	"os"
	"strings"
	"tg-sender/internal/safego"
	"time"
)

func (e *Engine) StartSessionBackup() {
	e.wg.Add(1)
	safego.Go(func() {
		defer e.wg.Done()
		ticker := time.NewTicker(e.cfg.SessionBackupInterval)
		defer ticker.Stop()
		for {
			select {
			case <-e.stopCh:
				return
			case <-ticker.C:
				e.backupSessionFiles()
			}
		}
	})
}

func (e *Engine) backupSessionFiles() {
	if err := os.MkdirAll(e.backupDir, 0o700); err != nil {
		log.Printf("[BACKUP] 创建备份目录失败: %v", err)
		return
	}
	backupRoot, err := os.OpenRoot(e.backupDir)
	if err != nil {
		log.Printf("[BACKUP] 打开备份目录失败: %v", err)
		return
	}
	defer backupRoot.Close()
	sourceRoot, err := os.OpenRoot(".")
	if err != nil {
		log.Printf("[BACKUP] 打开会话目录失败: %v", err)
		return
	}
	defer sourceRoot.Close()
	sourceDir, err := sourceRoot.Open(".")
	if err != nil {
		log.Printf("[BACKUP] 打开会话目录失败: %v", err)
		return
	}
	entries, err := sourceDir.ReadDir(-1)
	_ = sourceDir.Close()
	if err != nil {
		log.Printf("[BACKUP] 读取目录失败: %v", err)
		return
	}
	backedUp := 0
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".session") {
			continue
		}
		src, err := sourceRoot.Open(entry.Name())
		if err != nil {
			log.Printf("[BACKUP] 读取 %s 失败: %v", entry.Name(), err)
			continue
		}
		dstName := entry.Name() + "_" + time.Now().Format("20060102_150405.000")
		dst, err := backupRoot.OpenFile(dstName, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
		if err != nil {
			_ = src.Close()
			log.Printf("[BACKUP] 创建 %s 失败: %v", dstName, err)
			continue
		}
		_, copyErr := io.Copy(dst, src)
		closeDstErr := dst.Close()
		closeSrcErr := src.Close()
		if copyErr != nil || closeDstErr != nil || closeSrcErr != nil {
			log.Printf("[BACKUP] 写入 %s 失败: copy=%v close_dst=%v close_src=%v", dstName, copyErr, closeDstErr, closeSrcErr)
			_ = backupRoot.Remove(dstName)
			continue
		}
		backedUp++
	}
	if backedUp > 0 {
		log.Printf("[BACKUP] 已备份 %d 个会话文件到 %s\n", backedUp, e.backupDir)
	}
}
