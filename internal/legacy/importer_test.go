package legacy

import (
	"testing"
)

func TestMapGroupConfigs(t *testing.T) {
	raw := []byte(`{"maximum_file_size":5120,"limit_per_minute":20,"limit_per_hour":100}`)
	out := string(mapGroupConfigs(raw, 524288000))
	want := `{"default_visibility":"private","max_capacity":524288000,"max_file_size":5242880,"upload_rate_hour":100,"upload_rate_minute":20}`
	if out != want {
		t.Fatalf("mapGroupConfigs = %s, want %s", out, want)
	}

	// 空配置回退到默认值（10MB 文件限制、不限速）
	out = string(mapGroupConfigs(nil, 524288000))
	wantDefault := `{"default_visibility":"private","max_capacity":524288000,"max_file_size":10485760,"upload_rate_hour":0,"upload_rate_minute":0}`
	if out != wantDefault {
		t.Fatalf("mapGroupConfigs(nil) = %s, want %s", out, wantDefault)
	}
}

func TestMapStrategyConfigs(t *testing.T) {
	opts := ImportOptions{
		StoragePath:   "/data/storage",
		PublicBaseURL: "http://new.example.com",
	}
	raw := []byte(`{"driver":"local","root":"/old/lsky/storage/app/uploads","url":"http://old.example.com/i","visibility":"public"}`)
	out := string(mapStrategyConfigs(raw, opts))
	want := `{"base_url":"http://new.example.com","driver":"local","legacy_root":"/old/lsky/storage/app/uploads","legacy_url":"http://old.example.com/i","root":"/data/storage","url":"http://new.example.com"}`
	if out != want {
		t.Fatalf("mapStrategyConfigs local = %s, want %s", out, want)
	}

	// 云存储策略保留原始配置
	cloud := []byte(`{"driver":"s3","bucket":"b","root":"/x"}`)
	if got := string(mapStrategyConfigs(cloud, opts)); got != string(cloud) {
		t.Fatalf("mapStrategyConfigs cloud = %s, want %s", got, string(cloud))
	}
}

func TestMapUserConfigs(t *testing.T) {
	out := string(mapUserConfigs([]byte(`{"default_permission":1,"default_strategy":1}`)))
	if out != `{"default_strategy":1,"default_visibility":"public"}` {
		t.Fatalf("mapUserConfigs public = %s", out)
	}
	out = string(mapUserConfigs([]byte(`{"default_permission":0}`)))
	if out != `{"default_visibility":"private"}` {
		t.Fatalf("mapUserConfigs private = %s", out)
	}
}

func TestMapSetting(t *testing.T) {
	if got := mapSetting("app_name", "Lsky Pro"); len(got) != 2 {
		t.Fatalf("app_name should map to site.name and site.title, got %v", got)
	}
	if got := mapSetting("is_enable_gallery", "0"); len(got) != 1 || got[0][1] != "false" {
		t.Fatalf("is_enable_gallery 0 should map to false, got %v", got)
	}
	if got := mapSetting("is_enable_gallery", "1"); got[0][1] != "true" {
		t.Fatalf("is_enable_gallery 1 should map to true, got %v", got)
	}
	if got := mapSetting("storage.root", "/x"); got != nil {
		t.Fatalf("unknown keys must be ignored, got %v", got)
	}

	mail := mapSetting("mail", `{"default":"smtp","mailers":{"smtp":{"transport":"smtp","host":"smtp.test","port":587,"encryption":"tls","username":"u@test","password":"p"}}}`)
	if len(mail) != 6 {
		t.Fatalf("mail should map to 6 settings, got %v", mail)
	}
	foundSecure := false
	for _, kv := range mail {
		if kv[0] == "mail.smtp.secure" && kv[1] == "true" {
			foundSecure = true
		}
	}
	if !foundSecure {
		t.Fatalf("tls encryption should map to mail.smtp.secure=true, got %v", mail)
	}
}

func TestTruncate(t *testing.T) {
	if got := truncate("你好世界", 2); got != "你好" {
		t.Fatalf("truncate rune-aware failed: %q", got)
	}
	if got := truncate("short", 10); got != "short" {
		t.Fatalf("truncate should keep short strings: %q", got)
	}
}
