package data

import (
	"fmt"
	"strings"

	"gorm.io/gorm"

	"skyimage/internal/config"
)

// NewDatabase connects to the configured database and runs auto migrations.
func NewDatabase(cfg config.Config) (*gorm.DB, error) {
	db, err := OpenDatabase(cfg)
	if err != nil {
		return nil, err
	}
	if err := PrepareSchema(db); err != nil {
		return nil, err
	}
	return db, nil
}

func MustDatabase(cfg config.Config) *gorm.DB {
	db, err := NewDatabase(cfg)
	if err != nil {
		panic(err)
	}
	return db
}

// PrepareSchema runs lightweight column fixes then AutoMigrate for all models.
func PrepareSchema(db *gorm.DB) error {
	if err := ensureRelativePathColumn(db); err != nil {
		return fmt.Errorf("prepare files table: %w", err)
	}
	if err := ensurePublicURLColumn(db); err != nil {
		return fmt.Errorf("prepare files table: %w", err)
	}
	if err := ensureLastUsedAtColumn(db); err != nil {
		return fmt.Errorf("prepare api_tokens table: %w", err)
	}
	if err := ensureAPITokenHashes(db); err != nil {
		return fmt.Errorf("migrate api token hashes: %w", err)
	}
	if err := migrateTurnstileToCaptcha(db); err != nil {
		return fmt.Errorf("migrate turnstile to captcha: %w", err)
	}
	if err := AutoMigrateAll(db); err != nil {
		return fmt.Errorf("auto migrate: %w", err)
	}
	if err := MigrateUserIDsToSixteenDigits(db); err != nil {
		return fmt.Errorf("migrate user ids to 16 digits: %w", err)
	}
	return nil
}

// AutoMigrateAll migrates all application models.
func AutoMigrateAll(db *gorm.DB) error {
	return db.AutoMigrate(AllModels()...)
}

// AllModels returns GORM models in a dependency-friendly order for schema setup.
func AllModels() []interface{} {
	return []interface{}{
		&Group{},
		&User{},
		&UserOAuthBinding{},
		&OAuthState{},
		&UserPasskey{},
		&PasskeyChallenge{},
		&UserNotification{},
		&FileAsset{},
		&ConfigEntry{},
		&AuditProfile{},
		&Strategy{},
		&GroupStrategy{},
		&InstallerState{},
		&SessionEntry{},
		&ApiToken{},
		&Album{},
		&RedeemCode{},
		&RedeemCodeUsage{},
		&ShopProduct{},
		&ShopOrder{},
		&Ticket{},
		&TicketMessage{},
		&TicketAttachment{},
	}
}

// MigrateTables lists tables to copy during cross-database migration (FK-safe order).
func MigrateTables() []MigrateTable {
	return []MigrateTable{
		{Name: "groups", Model: &Group{}},
		{Name: "strategies", Model: &Strategy{}},
		{Name: "group_strategy", Model: &GroupStrategy{}},
		{Name: "users", Model: &User{}},
		{Name: "user_oauth_bindings", Model: &UserOAuthBinding{}},
		{Name: "oauth_states", Model: &OAuthState{}},
		{Name: "user_passkeys", Model: &UserPasskey{}},
		{Name: "passkey_challenges", Model: &PasskeyChallenge{}},
		{Name: "user_notifications", Model: &UserNotification{}},
		{Name: "files", Model: &FileAsset{}},
		{Name: "configs", Model: &ConfigEntry{}},
		{Name: "audit_profiles", Model: &AuditProfile{}},
		{Name: "installer_states", Model: &InstallerState{}},
		{Name: "sessions", Model: &SessionEntry{}},
		{Name: "api_tokens", Model: &ApiToken{}},
		{Name: "albums", Model: &Album{}},
		{Name: "redeem_codes", Model: &RedeemCode{}},
		{Name: "redeem_code_usages", Model: &RedeemCodeUsage{}},
		{Name: "shop_products", Model: &ShopProduct{}},
		{Name: "shop_orders", Model: &ShopOrder{}},
		{Name: "tickets", Model: &Ticket{}},
		{Name: "ticket_messages", Model: &TicketMessage{}},
		{Name: "ticket_attachments", Model: &TicketAttachment{}},
	}
}

// MigrateTable describes one table involved in data migration.
type MigrateTable struct {
	Name  string
	Model interface{}
}

func dialectName(db *gorm.DB) string {
	if db == nil || db.Dialector == nil {
		return ""
	}
	return strings.ToLower(db.Dialector.Name())
}

func quoteIdent(db *gorm.DB, name string) string {
	switch dialectName(db) {
	case "postgres":
		return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
	default:
		return "`" + strings.ReplaceAll(name, "`", "``") + "`"
	}
}

func ensureRelativePathColumn(db *gorm.DB) error {
	if !db.Migrator().HasTable(&FileAsset{}) {
		return nil
	}
	if db.Migrator().HasColumn(&FileAsset{}, "relative_path") {
		return nil
	}
	table := quoteIdent(db, "files")
	col := quoteIdent(db, "relative_path")
	if err := db.Exec(fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s TEXT DEFAULT ''", table, col)).Error; err != nil {
		// 并发启动时另一个进程可能已补齐该列（SQLite 驱动的 HasColumn
		// 存在误判），视为成功。
		if !isDuplicateColumnErr(err) {
			return err
		}
	}
	return db.Exec(fmt.Sprintf("UPDATE %s SET %s = '' WHERE %s IS NULL", table, col, col)).Error
}

// isDuplicateColumnErr reports whether the error means the column already
// exists across sqlite/mysql/postgres.
func isDuplicateColumnErr(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "duplicate column name") ||
		strings.Contains(msg, "duplicate column") ||
		strings.Contains(msg, "already exists")
}

func ensurePublicURLColumn(db *gorm.DB) error {
	if !db.Migrator().HasTable(&FileAsset{}) {
		return nil
	}
	if db.Migrator().HasColumn(&FileAsset{}, "public_url") {
		return nil
	}
	table := quoteIdent(db, "files")
	col := quoteIdent(db, "public_url")
	if err := db.Exec(fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s TEXT DEFAULT ''", table, col)).Error; err != nil {
		return err
	}
	return db.Exec(fmt.Sprintf("UPDATE %s SET %s = '' WHERE %s IS NULL", table, col, col)).Error
}

func ensureLastUsedAtColumn(db *gorm.DB) error {
	if !db.Migrator().HasTable(&ApiToken{}) {
		return nil
	}
	if db.Migrator().HasColumn(&ApiToken{}, "last_used_at") {
		return nil
	}
	table := quoteIdent(db, "api_tokens")
	col := quoteIdent(db, "last_used_at")
	typ := "DATETIME"
	if dialectName(db) == "postgres" {
		typ = "TIMESTAMPTZ"
	}
	return db.Exec(fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", table, col, typ)).Error
}

func ensureAPITokenHashes(db *gorm.DB) error {
	if !db.Migrator().HasTable(&ApiToken{}) {
		return nil
	}
	type row struct {
		ID    uint
		Token string
	}
	var rows []row
	if err := db.Table("api_tokens").Select("id, token").Find(&rows).Error; err != nil {
		return err
	}
	for _, item := range rows {
		if !IsLegacyPlainAPIToken(item.Token) {
			continue
		}
		hashed := HashAPIToken(item.Token)
		if err := db.Table("api_tokens").
			Where("id = ? AND token = ?", item.ID, item.Token).
			Update("token", hashed).Error; err != nil {
			return err
		}
	}
	return nil
}

// migrateTurnstileToCaptcha migrates old turnstile.* config keys to the new captcha.* keys.
// This is needed for upgrades from v0.1.9 and earlier where Cloudflare Turnstile settings
// were stored under "turnstile.*" keys. The new unified captcha system uses "captcha.cloudflare.*"
// and "captcha.*" keys instead.
func migrateTurnstileToCaptcha(db *gorm.DB) error {
	if !db.Migrator().HasTable(&ConfigEntry{}) {
		return nil
	}

	var oldCount int64
	db.Model(&ConfigEntry{}).Where("key = ?", "turnstile.site_key").Count(&oldCount)
	if oldCount == 0 {
		return nil
	}

	var newCount int64
	db.Model(&ConfigEntry{}).Where("key = ?", "captcha.cloudflare.site_key").Count(&newCount)
	if newCount > 0 {
		return nil
	}

	migrations := map[string]string{
		"turnstile.site_key":                "captcha.cloudflare.site_key",
		"turnstile.secret_key":              "captcha.cloudflare.secret_key",
		"turnstile.last_verified_signature": "captcha.cloudflare.last_verified_signature",
		"turnstile.last_verified_at":        "captcha.cloudflare.last_verified_at",
		"turnstile.login":                   "captcha.login",
		"turnstile.register":                "captcha.register",
		"turnstile.register_verify":         "captcha.register_verify",
	}

	for oldKey, newKey := range migrations {
		var entry ConfigEntry
		if err := db.Where("key = ?", oldKey).First(&entry).Error; err != nil {
			continue
		}
		db.Where("key = ?", newKey).Delete(&ConfigEntry{})
		db.Create(&ConfigEntry{Key: newKey, Value: entry.Value})
	}

	var enabledEntry ConfigEntry
	if err := db.Where("key = ?", "turnstile.enabled").First(&enabledEntry).Error; err == nil {
		db.Where("key = ?", "captcha.enabled").Delete(&ConfigEntry{})
		db.Create(&ConfigEntry{Key: "captcha.enabled", Value: enabledEntry.Value})
		if enabledEntry.Value == "true" {
			db.Where("key = ?", "captcha.provider").Delete(&ConfigEntry{})
			db.Create(&ConfigEntry{Key: "captcha.provider", Value: "cloudflare"})
		}
	}

	forgotMigrations := map[string]string{
		"turnstile.forgot_password_request": "captcha.forgot_password_request",
		"turnstile.forgot_password_reset":   "captcha.forgot_password_reset",
	}
	for oldKey, newKey := range forgotMigrations {
		var entry ConfigEntry
		if err := db.Where("key = ?", oldKey).First(&entry).Error; err != nil {
			continue
		}
		db.Where("key = ?", newKey).Delete(&ConfigEntry{})
		db.Create(&ConfigEntry{Key: newKey, Value: entry.Value})
	}

	return nil
}
