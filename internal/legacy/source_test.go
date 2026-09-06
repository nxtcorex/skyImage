package legacy

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"skyimage/internal/data"
)

// createSQLiteLskyFixture 在临时目录构造一个最小可用的 Lsky Pro SQLite 源库。
func createSQLiteLskyFixture(t *testing.T) (dir string, uploadsDir string) {
	t.Helper()
	dir = t.TempDir()
	db, err := sql.Open("sqlite", "file:"+filepath.ToSlash(filepath.Join(dir, "database.sqlite")))
	if err != nil {
		t.Fatalf("open fixture: %v", err)
	}
	defer db.Close()

	script := `
CREATE TABLE groups (id INTEGER PRIMARY KEY, name TEXT, is_default INTEGER, is_guest INTEGER, configs TEXT, created_at TIMESTAMP, updated_at TIMESTAMP);
CREATE TABLE users (id INTEGER PRIMARY KEY, group_id INTEGER, name TEXT, email TEXT, password TEXT, remember_token TEXT, url TEXT, capacity NUMERIC, configs TEXT, is_adminer INTEGER, image_num INTEGER, album_num INTEGER, registered_ip TEXT, status INTEGER, email_verified_at TIMESTAMP, created_at TIMESTAMP, updated_at TIMESTAMP);
CREATE TABLE albums (id INTEGER PRIMARY KEY, user_id INTEGER, name TEXT, intro TEXT, image_num INTEGER, created_at TIMESTAMP, updated_at TIMESTAMP);
CREATE TABLE strategies (id INTEGER PRIMARY KEY, "key" INTEGER, name TEXT, intro TEXT, configs TEXT, created_at TIMESTAMP, updated_at TIMESTAMP);
CREATE TABLE images (id INTEGER PRIMARY KEY, user_id INTEGER, album_id INTEGER, group_id INTEGER, strategy_id INTEGER, "key" TEXT, path TEXT, name TEXT, origin_name TEXT, alias_name TEXT, size NUMERIC, mimetype TEXT, extension TEXT, md5 TEXT, sha1 TEXT, width INTEGER, height INTEGER, permission INTEGER, is_unhealthy INTEGER, uploaded_ip TEXT, created_at TIMESTAMP, updated_at TIMESTAMP);
CREATE TABLE configs (name TEXT PRIMARY KEY, value TEXT, created_at TIMESTAMP, updated_at TIMESTAMP);
CREATE TABLE group_strategy (group_id INTEGER, strategy_id INTEGER);
INSERT INTO groups VALUES (1, '系统默认组', 1, 1, '{"maximum_file_size":5120,"limit_per_minute":20,"limit_per_hour":100}', '2026-01-02 15:04:05', '2026-01-02 15:04:05');
INSERT INTO users VALUES (7, 1, '张三', 'zhangsan@lsky.local', '$2y$10$hash', 'tok', '', 512000, '{"default_permission":1}', 0, 1, 0, '192.168.1.10', 1, NULL, '2026-01-02 15:04:05', '2026-01-02 15:04:05');
INSERT INTO albums VALUES (1, 7, '相册', '', 1, '2026-01-02 15:04:05', '2026-01-02 15:04:05');
INSERT INTO strategies VALUES (1, 1, '默认本地策略', '', '{"driver":"local","root":"/old/uploads","url":"http://old/i"}', '2026-01-02 15:04:05', '2026-01-02 15:04:05');
INSERT INTO group_strategy VALUES (1, 1);
INSERT INTO images VALUES (1, 7, 1, 1, 1, 'keyok', '2026/01/02', 'ok.jpg', 'ok.jpg', '', 2.5, 'image/jpeg', 'jpg', 'md5', 'sha1', 10, 10, 1, 0, '127.0.0.1', '2026-01-02 15:04:05', '2026-01-02 15:04:05');
INSERT INTO images VALUES (2, 7, NULL, 1, 1, 'keymissing', '2026/01/02', 'missing.jpg', 'missing.jpg', '', 2.5, 'image/jpeg', 'jpg', 'md5', 'sha1', 10, 10, 1, 0, '127.0.0.1', '2026-01-02 15:04:05', '2026-01-02 15:04:05');
INSERT INTO images VALUES (3, 7, NULL, 1, 1, 'keyevil', '../evil', 'evil.jpg', 'evil.jpg', '', 2.5, 'image/jpeg', 'jpg', 'md5', 'sha1', 10, 10, 1, 0, '127.0.0.1', '2026-01-02 15:04:05', '2026-01-02 15:04:05');
INSERT INTO images VALUES (4, 7, NULL, 1, 1, 'keyapi', 'api', 'api.jpg', 'api.jpg', '', 2.5, 'image/jpeg', 'jpg', 'md5', 'sha1', 10, 10, 1, 0, '127.0.0.1', '2026-01-02 15:04:05', '2026-01-02 15:04:05');
INSERT INTO configs VALUES ('app_name', 'Lsky Pro', '2026-01-02 15:04:05', '2026-01-02 15:04:05');
INSERT INTO configs VALUES ('app_version', 'V 2.1', '2026-01-02 15:04:05', '2026-01-02 15:04:05');
INSERT INTO configs VALUES ('user_initial_capacity', '512000', '2026-01-02 15:04:05', '2026-01-02 15:04:05');
`
	if _, err := db.Exec(script); err != nil {
		t.Fatalf("seed fixture: %v", err)
	}

	uploadsDir = filepath.Join(dir, "uploads")
	if err := os.MkdirAll(filepath.Join(uploadsDir, "2026", "01", "02"), 0o755); err != nil {
		t.Fatalf("mkdir uploads: %v", err)
	}
	if err := os.WriteFile(filepath.Join(uploadsDir, "2026", "01", "02", "ok.jpg"), []byte("jpeg-bytes"), 0o644); err != nil {
		t.Fatalf("write fixture image: %v", err)
	}
	return dir, uploadsDir
}

func TestSQLiteSourceProbeAndImport(t *testing.T) {
	dir, uploadsDir := createSQLiteLskyFixture(t)

	src := Source{Type: SourceTypeSQLite, SQLiteDir: dir}
	srcDB, readOnly, err := OpenSource(context.Background(), src)
	if err != nil {
		t.Fatalf("OpenSource: %v", err)
	}
	defer srcDB.Close()
	if !readOnly {
		t.Fatal("sqlite source must be read-only")
	}

	probe, err := Probe(context.Background(), srcDB, src, true)
	if err != nil {
		t.Fatalf("Probe: %v", err)
	}
	if !probe.OK() {
		t.Fatalf("probe missing tables: %v", probe.MissingTables)
	}
	if probe.Counts["users"] != 1 || probe.Counts["images"] != 4 || probe.AppVersion != "V 2.1" {
		t.Fatalf("unexpected probe: %+v", probe)
	}

	target := t.TempDir()
	gdb, err := gorm.Open(sqlite.Open("file:import_test?mode=memory&cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open target: %v", err)
	}
	if err := gdb.AutoMigrate(&data.Group{}, &data.User{}, &data.Album{}, &data.Strategy{},
		&data.GroupStrategy{}, &data.FileAsset{}, &data.ConfigEntry{}); err != nil {
		t.Fatalf("migrate target: %v", err)
	}
	admin := data.User{ID: 1000000000000001, Name: "admin", Email: "admin@skyimage.local", IsSuperAdmin: true}
	if err := gdb.Create(&admin).Error; err != nil {
		t.Fatalf("seed admin: %v", err)
	}

	summary, err := RunImport(context.Background(), srcDB, src, readOnly, gdb, ImportOptions{
		StoragePath:   target,
		PublicBaseURL: "http://new.test",
		ImageDir:      uploadsDir,
		OrphanUserID:  admin.ID,
	})
	if err != nil {
		t.Fatalf("RunImport: %v", err)
	}

	if summary.Users != 1 || summary.Groups != 1 || summary.Strategies != 1 || summary.Albums != 1 {
		t.Fatalf("unexpected counts: %+v", summary)
	}
	// 4 条图片记录：2 条写入（1 张拷贝成功、1 张文件缺失）、2 条路径被禁止跳过
	if summary.Files != 2 || summary.FilesCopied != 1 || summary.FilesMissing != 1 || summary.FilesForbidden != 2 {
		t.Fatalf("unexpected file summary: %+v", summary)
	}
	if len(summary.ForbiddenPaths) != 2 {
		t.Fatalf("forbidden paths not reported: %v", summary.ForbiddenPaths)
	}
	if summary.Settings != 2 || summary.AppName != "Lsky Pro" {
		t.Fatalf("unexpected settings summary: %+v", summary)
	}

	// 拷贝的文件保持原路径
	copied := filepath.Join(target, "2026", "01", "02", "ok.jpg")
	if content, err := os.ReadFile(copied); err != nil || string(content) != "jpeg-bytes" {
		t.Fatalf("copied file mismatch: %v %q", err, content)
	}

	// 用户重新分配 16 位 ID 且配置完成映射
	var imported data.User
	if err := gdb.Where("email = ?", "zhangsan@lsky.local").First(&imported).Error; err != nil {
		t.Fatalf("imported user missing: %v", err)
	}
	if imported.ID < 1_000_000_000_000_000 || imported.ID > 9_999_999_999_999_999 {
		t.Fatalf("user id not 16 digits: %d", imported.ID)
	}
	if imported.CapacityBonus != 0 { // 512000KB == user_initial_capacity，差额为 0
		t.Fatalf("unexpected capacity bonus: %v", imported.CapacityBonus)
	}

	// 组配置换算为字节
	var group data.Group
	if err := gdb.First(&group, 1).Error; err != nil {
		t.Fatalf("group missing: %v", err)
	}
	if got := string(group.Configs); got == "" || !strings.Contains(got, `"max_file_size":5242880`) {
		t.Fatalf("group configs not converted: %s", got)
	}

	// 策略指向新存储
	var strategy data.Strategy
	if err := gdb.First(&strategy, 1).Error; err != nil {
		t.Fatalf("strategy missing: %v", err)
	}
	var cfgMap map[string]string
	if err := json.Unmarshal(strategy.Configs, &cfgMap); err != nil {
		t.Fatalf("strategy configs not json: %v", err)
	}
	if cfgMap["root"] != target {
		t.Fatalf("strategy root not remapped: %v", cfgMap)
	}
	if cfgMap["base_url"] != "http://new.test" {
		t.Fatalf("strategy base_url not remapped: %v", cfgMap)
	}
}

func TestSanitizeImportRelPath(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"2026/01/02/ok.jpg", true},
		{"ok.jpg", true},
		{"", false},
		{"../evil/evil.jpg", false},
		{"a/../../b.jpg", false},
		{"/abs/path.jpg", false},
		{"C:/windows/x.jpg", false},
		{"api/x.jpg", false},
		{"assets/x.jpg", false},
		{"robots.txt", false},
		{"a/./b.jpg", false},
		{"a//b.jpg", false},
	}
	for _, c := range cases {
		if _, ok := sanitizeImportRelPath(c.in); ok != c.want {
			t.Errorf("sanitizeImportRelPath(%q) = %v, want %v", c.in, ok, c.want)
		}
	}
}

func TestDialectQueries(t *testing.T) {
	cols := []string{"id", "key", "name"}

	pg := Source{Type: SourceTypePostgres}
	if got := pg.pageQuery("images", "id", cols, 200); got != `SELECT "id", "key", "name" FROM "images" WHERE "id" > $1 ORDER BY "id" ASC LIMIT 200` {
		t.Fatalf("pg pageQuery = %s", got)
	}
	if got := pg.quote("configs"); got != `"configs"` {
		t.Fatalf("pg quote = %s", got)
	}
	if got := pg.ph(3); got != "$3" {
		t.Fatalf("pg ph = %s", got)
	}

	ms := Source{Type: SourceTypeSQLServer}
	if got := ms.pageQuery("images", "id", cols, 200); got != `SELECT TOP (200) [id], [key], [name] FROM [images] WHERE [id] > @p1 ORDER BY [id] ASC` {
		t.Fatalf("mssql pageQuery = %s", got)
	}
	if got := ms.firstRowQuery("[value]", "[configs] WHERE [name] = 'app_version'"); got != "SELECT TOP (1) [value] FROM [configs] WHERE [name] = 'app_version'" {
		t.Fatalf("mssql firstRowQuery = %s", got)
	}

	my := Source{Type: SourceTypeMySQL}
	if got := my.pageQuery("images", "id", cols, 200); got != "SELECT `id`, `key`, `name` FROM `images` WHERE `id` > ? ORDER BY `id` ASC LIMIT 200" {
		t.Fatalf("mysql pageQuery = %s", got)
	}

	li := Source{Type: SourceTypeSQLite}
	if got := li.pageQuery("images", "id", cols, 200); got != "SELECT `id`, `key`, `name` FROM `images` WHERE `id` > ? ORDER BY `id` ASC LIMIT 200" {
		t.Fatalf("sqlite pageQuery = %s", got)
	}

	for _, raw := range []string{"postgresql", "pgsql", "pg"} {
		if NormalizeType(raw) != SourceTypePostgres {
			t.Fatalf("NormalizeType(%q) != postgres", raw)
		}
	}
	for _, raw := range []string{"mssql", "sql server", "SQLServer"} {
		if NormalizeType(raw) != SourceTypeSQLServer {
			t.Fatalf("NormalizeType(%q) != sqlserver", raw)
		}
	}
}

func TestFlexTimeParse(t *testing.T) {
	var f flexTime
	if err := f.Scan("2026-01-02 15:04:05"); err != nil || !f.valid {
		t.Fatalf("sqlite datetime parse failed: %v", err)
	}
	if err := f.Scan(nil); err != nil || f.valid {
		t.Fatalf("nil should be invalid: %v", err)
	}
	if err := f.Scan(""); err != nil || f.valid {
		t.Fatalf("empty should be invalid: %v", err)
	}
}
