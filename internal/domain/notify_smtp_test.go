package domain

import (
	"os"
	"testing"

	"tg-sender/internal/crypto"
)

func TestDecryptNotifySMTPPass(t *testing.T) {
	t.Setenv("SESSION_ENCRYPTION_KEY", "a-very-long-test-key-for-smtp-encryption-0123456789")

	// 历史明文行没有前缀，必须原样返回，否则老租户升级后发不出邮件。
	if got := DecryptNotifySMTPPass("legacy-plain-pass"); got != "legacy-plain-pass" {
		t.Fatalf("明文兼容失败: %q", got)
	}

	enc, err := crypto.Encrypt([]byte("smtp-app-password"))
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	if got := DecryptNotifySMTPPass(NotifySMTPPassPrefix + enc); got != "smtp-app-password" {
		t.Fatalf("解密失败: %q", got)
	}
}

// 解不开时返回空串（发信会失败），绝不能把密文当密码去认证。
func TestDecryptNotifySMTPPassCorruptReturnsEmpty(t *testing.T) {
	t.Setenv("SESSION_ENCRYPTION_KEY", "a-very-long-test-key-for-smtp-encryption-0123456789")
	if got := DecryptNotifySMTPPass(NotifySMTPPassPrefix + "not-valid-base64!!"); got != "" {
		t.Fatalf("损坏密文应返回空串, 实际 %q", got)
	}
}

func TestIsLegacyMessageVariable(t *testing.T) {
	for _, name := range []string{"first_name", "Group_Name", "user_id", "now", "username", "phone"} {
		if !IsLegacyMessageVariable(name) {
			t.Fatalf("%q 应判定为退役变量", name)
		}
	}
	for _, name := range []string{"活动名", "custom_dict", "welcome"} {
		if IsLegacyMessageVariable(name) {
			t.Fatalf("%q 不该判定为退役变量", name)
		}
	}
}

func TestSMTPConfigDecryptsStoredPass(t *testing.T) {
	t.Setenv("SESSION_ENCRYPTION_KEY", "a-very-long-test-key-for-smtp-encryption-0123456789")
	enc, err := crypto.Encrypt([]byte("app-pass"))
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	tenant := &Tenant{NotifySMTP: []byte(`{"host":"smtp.qq.com","port":465,"user":"a@qq.com","pass":"` + NotifySMTPPassPrefix + enc + `","from":"a@qq.com"}`)}
	cfg := tenant.SMTPConfig()
	if cfg == nil {
		t.Fatal("SMTPConfig 返回 nil")
	}
	if cfg.Pass != "app-pass" {
		t.Fatalf("授权码未解密: %q", cfg.Pass)
	}
	if _, ok := os.LookupEnv("SESSION_ENCRYPTION_KEY"); !ok {
		t.Fatal("测试环境缺少密钥")
	}
}
