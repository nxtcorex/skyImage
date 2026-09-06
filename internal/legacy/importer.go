package legacy

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"skyimage/internal/data"
	"skyimage/internal/users"
)

const batchSize = 200

// ImportOptions controls one import run from a Lsky Pro database into the
// local (target) database.
type ImportOptions struct {
	// StoragePath is the local storage root of the new deployment.
	StoragePath string
	// PublicBaseURL is the public base URL of the new deployment.
	PublicBaseURL string
	// ImageDir 可选：旧版 Lsky Pro 的图片保存目录（原先的
	// storage/app/uploads 文件夹）。提供后导入会按原相对路径把图片文件
	// 拷贝进 StoragePath；留空则只迁移数据库记录。
	ImageDir string
	// OrphanUserID receives images whose user_id is NULL in Lsky Pro
	// (guest uploads).
	OrphanUserID uint
}

// ImportSummary reports what the import wrote into the target database.
type ImportSummary struct {
	AppName        string `json:"appName"`
	AppVersion     string `json:"appVersion"`
	SourceReadOnly bool   `json:"sourceReadOnly"`

	Groups          int64 `json:"groups"`
	Strategies      int64 `json:"strategies"`
	GroupStrategies int64 `json:"groupStrategies"`
	Users           int64 `json:"users"`
	UsersMerged     int64 `json:"usersMerged"`
	Albums          int64 `json:"albums"`
	Files           int64 `json:"files"`
	FilesCopied     int64 `json:"filesCopied"`
	FilesMissing    int64 `json:"filesMissing"`
	FilesForbidden  int64 `json:"filesForbidden"`
	Settings        int64 `json:"settings"`

	// ForbiddenPaths 列出因路径被禁止而未能导入的图片（最多 10 条示例），
	// 例如包含 ".." 的路径、绝对路径或与系统保留路径冲突的路径。
	ForbiddenPaths []string `json:"forbiddenPaths,omitempty"`
}

type strategyInfo struct {
	Driver string
	Root   string
	URL    string
}

type importer struct {
	q      queryRunner
	srcCfg Source
	tx     *gorm.DB
	opts   ImportOptions

	strategies   map[uint]strategyInfo
	userIDMap    map[uint]uint
	strategyIDs  map[uint]struct{}
	initialBytes float64

	// stagedStrategies 在事务前读取并解析，事务阶段只负责写入。
	stagedStrategies []data.Strategy

	forbiddenPaths []string

	summary ImportSummary
}

// RunImport copies a Lsky Pro database into the target database. The source
// database is only read (SELECT): MySQL sessions are forced read-only, SQLite
// files are opened read-only and PostgreSQL/SQL Server reads run inside
// read-only transactions.
//
// 图片文件的拷贝发生在事务之前的预备阶段，数据库事务只写记录：保持事务
// 短小（目标库锁持有时间与图片数量无关），且取消信号可随时中止导入。
func RunImport(ctx context.Context, src *sql.DB, srcCfg Source, openReadOnly bool, db *gorm.DB, opts ImportOptions) (ImportSummary, error) {
	srcCfg = srcCfg.Sanitize()
	runner, readOnly, cleanup := srcCfg.readOnlyRunner(ctx, src, openReadOnly)
	defer cleanup()

	im := &importer{
		q:           runner,
		srcCfg:      srcCfg,
		opts:        opts,
		strategies:  map[uint]strategyInfo{},
		userIDMap:   map[uint]uint{},
		strategyIDs: map[uint]struct{}{},
	}
	im.summary.SourceReadOnly = readOnly
	im.initialBytes = im.initialCapacityBytes(ctx)

	// 事务前的预备阶段：只读源库、拷贝文件，不写目标数据库。
	if err := im.prepareStrategies(ctx); err != nil {
		return ImportSummary{}, fmt.Errorf("导入 strategies 失败: %w", err)
	}
	if err := im.precopyImages(ctx); err != nil {
		return ImportSummary{}, fmt.Errorf("拷贝图片文件失败: %w", err)
	}

	err := db.Transaction(func(txCtx *gorm.DB) error {
		im.tx = txCtx
		steps := []struct {
			name string
			run  func(context.Context) error
		}{
			{"strategies", im.importStrategies},
			{"groups", im.importGroups},
			{"group_strategy", im.importGroupStrategies},
			{"users", im.importUsers},
			{"albums", im.importAlbums},
			{"images", im.importImages},
			{"configs", im.importSettings},
		}
		for _, step := range steps {
			if err := step.run(ctx); err != nil {
				return fmt.Errorf("导入 %s 失败: %w", step.name, err)
			}
		}
		return nil
	})
	if err != nil {
		return ImportSummary{}, err
	}
	im.summary.AppName, im.summary.AppVersion = probeAppVersion(ctx, im.q, srcCfg)
	im.summary.ForbiddenPaths = im.forbiddenPaths
	return im.summary, nil
}

// --- strategies -----------------------------------------------------------

type legacyStrategy struct {
	ID        uint
	Key       uint8
	Name      string
	Intro     string
	Configs   []byte
	CreatedAt flexTime
	UpdatedAt flexTime
}

// prepareStrategies 在事务前读取源库策略并解析配置，供图片文件定位与
// 事务阶段的写入使用。
func (im *importer) prepareStrategies(ctx context.Context) error {
	return im.batch(ctx, "strategies", "id", []string{"id", "key", "name", "intro", "configs", "created_at", "updated_at"}, func(rows *sql.Rows) (int64, error) {
		var s legacyStrategy
		if err := rows.Scan(&s.ID, &s.Key, &s.Name, &s.Intro, &s.Configs, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return 0, err
		}
		im.strategies[s.ID] = parseStrategyInfo(s.Configs)
		im.strategyIDs[s.ID] = struct{}{}
		im.stagedStrategies = append(im.stagedStrategies, data.Strategy{
			ID:        s.ID,
			Key:       s.Key,
			Name:      truncate(s.Name, 64),
			Intro:     truncate(s.Intro, 255),
			Configs:   mapStrategyConfigs(s.Configs, im.opts),
			CreatedAt: nullTimeOrNow(s.CreatedAt),
			UpdatedAt: nullTimeOrNow(s.UpdatedAt),
		})
		return int64(s.ID), nil
	})
}

// importStrategies 在事务阶段写入预备阶段已解析的策略记录。
func (im *importer) importStrategies(_ context.Context) error {
	for i := range im.stagedStrategies {
		if err := im.upsert(&im.stagedStrategies[i]); err != nil {
			return err
		}
		im.summary.Strategies++
	}
	return nil
}

func parseStrategyInfo(raw []byte) strategyInfo {
	info := strategyInfo{Driver: "local"}
	var cfg map[string]interface{}
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return info
	}
	if v, _ := cfg["driver"].(string); v != "" {
		info.Driver = v
	}
	info.Root, _ = cfg["root"].(string)
	info.URL, _ = cfg["url"].(string)
	return info
}

// mapStrategyConfigs rewrites a Lsky Pro strategy config so the strategy can
// serve from the new deployment: local storage strategies point at the new
// storage root and public base URL; cloud strategies keep their raw configs.
func mapStrategyConfigs(raw []byte, opts ImportOptions) datatypes.JSON {
	info := parseStrategyInfo(raw)
	if !strings.EqualFold(info.Driver, "local") {
		if len(raw) == 0 {
			return datatypes.JSON("{}")
		}
		return datatypes.JSON(raw)
	}
	mapped := map[string]string{
		"driver":   "local",
		"root":     opts.StoragePath,
		"url":      opts.PublicBaseURL,
		"base_url": opts.PublicBaseURL,
	}
	if info.Root != "" {
		mapped["legacy_root"] = info.Root
	}
	if info.URL != "" {
		mapped["legacy_url"] = info.URL
	}
	out, _ := json.Marshal(mapped)
	return datatypes.JSON(out)
}

// --- groups ---------------------------------------------------------------

type legacyGroup struct {
	ID        uint
	Name      string
	IsDefault bool
	IsGuest   bool
	Configs   []byte
	CreatedAt flexTime
	UpdatedAt flexTime
}

func (im *importer) importGroups(ctx context.Context) error {
	return im.batch(ctx, "groups", "id", []string{"id", "name", "is_default", "is_guest", "configs", "created_at", "updated_at"}, func(rows *sql.Rows) (int64, error) {
		var g legacyGroup
		if err := rows.Scan(&g.ID, &g.Name, &g.IsDefault, &g.IsGuest, &g.Configs, &g.CreatedAt, &g.UpdatedAt); err != nil {
			return 0, err
		}
		// Group names are unique in skyImage; disambiguate collisions,
		// including collisions with previously renamed groups.
		name := truncate(g.Name, 64)
		for i := 0; ; i++ {
			var existing data.Group
			if err := im.tx.Where("name = ? AND id <> ?", name, g.ID).First(&existing).Error; err != nil {
				break
			}
			suffix := fmt.Sprintf(" (%d)", g.ID)
			if i > 0 {
				suffix = fmt.Sprintf(" (%d.%d)", g.ID, i+1)
			}
			name = truncate(g.Name, 64-len([]rune(suffix))) + suffix
		}
		group := data.Group{
			ID:        g.ID,
			Name:      name,
			IsDefault: g.IsDefault,
			IsGuest:   g.IsGuest,
			Configs:   mapGroupConfigs(g.Configs, im.initialBytes),
			CreatedAt: nullTimeOrNow(g.CreatedAt),
			UpdatedAt: nullTimeOrNow(g.UpdatedAt),
		}
		if err := im.upsert(&group); err != nil {
			return 0, err
		}
		im.summary.Groups++
		return int64(g.ID), nil
	})
}

// initialCapacityBytes reads the Lsky Pro "user_initial_capacity" setting
// (KB) and converts it to bytes; it becomes the group base capacity so each
// imported user keeps their individual allocation via capacity_bonus.
func (im *importer) initialCapacityBytes(ctx context.Context) float64 {
	var value sql.NullString
	query := im.srcCfg.firstRowQuery(im.srcCfg.quote("value"),
		im.srcCfg.quote(im.srcCfg.table("configs"))+" WHERE "+im.srcCfg.quote("name")+" = 'user_initial_capacity'")
	if err := im.q.QueryRowContext(ctx, query).Scan(&value); err != nil || !value.Valid {
		return defaultInitialCapacityBytes
	}
	kb, err := strconv.ParseFloat(strings.TrimSpace(value.String), 64)
	if err != nil || kb <= 0 {
		return defaultInitialCapacityBytes
	}
	return kb * 1024
}

const defaultInitialCapacityBytes = 500 * 1024 * 1024

// mapGroupConfigs converts a Lsky Pro group config JSON into the skyImage
// group config schema (byte-based sizes, per-minute/hour rate limits).
func mapGroupConfigs(raw []byte, initialCapacityBytes float64) datatypes.JSON {
	mapped := map[string]interface{}{
		"max_file_size":      int64(10 * 1024 * 1024),
		"max_capacity":       initialCapacityBytes,
		"default_visibility": "private",
		"upload_rate_minute": 0,
		"upload_rate_hour":   0,
	}
	var cfg map[string]interface{}
	if err := json.Unmarshal(raw, &cfg); err == nil {
		if v, ok := numberValue(cfg["maximum_file_size"]); ok && v > 0 {
			mapped["max_file_size"] = int64(v * 1024)
		}
		if v, ok := numberValue(cfg["limit_per_minute"]); ok && v >= 0 {
			mapped["upload_rate_minute"] = int64(v)
		}
		if v, ok := numberValue(cfg["limit_per_hour"]); ok && v >= 0 {
			mapped["upload_rate_hour"] = int64(v)
		}
	}
	out, _ := json.Marshal(mapped)
	return datatypes.JSON(out)
}

func numberValue(raw interface{}) (float64, bool) {
	switch v := raw.(type) {
	case float64:
		return v, true
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	case string:
		if f, err := strconv.ParseFloat(strings.TrimSpace(v), 64); err == nil {
			return f, true
		}
	}
	return 0, false
}

// --- group_strategy -------------------------------------------------------

func (im *importer) importGroupStrategies(ctx context.Context) error {
	rows, err := im.q.QueryContext(ctx, "SELECT "+im.srcCfg.quote("group_id")+", "+im.srcCfg.quote("strategy_id")+
		" FROM "+im.srcCfg.quote(im.srcCfg.table("group_strategy")))
	if err != nil {
		// The pivot table only exists on complete installations.
		if isMissingTableErr(err) {
			return nil
		}
		return err
	}
	defer rows.Close()

	type pair struct {
		GroupID    uint
		StrategyID uint
	}
	var pairs []pair
	for rows.Next() {
		var p pair
		if err := rows.Scan(&p.GroupID, &p.StrategyID); err != nil {
			return err
		}
		pairs = append(pairs, p)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for _, p := range pairs {
		if _, ok := im.strategyIDs[p.StrategyID]; !ok {
			continue
		}
		link := data.GroupStrategy{GroupID: p.GroupID, StrategyID: p.StrategyID}
		if err := im.tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&link).Error; err != nil {
			return err
		}
		im.summary.GroupStrategies++
	}
	return nil
}

// --- users ----------------------------------------------------------------

type legacyUser struct {
	ID            uint
	GroupID       sql.NullInt64
	Name          string
	Email         string
	Password      string
	RememberToken sql.NullString
	URL           string
	Capacity      flexFloat
	Configs       []byte
	IsAdmin       bool
	ImageNum      sql.NullInt64
	AlbumNum      sql.NullInt64
	RegisteredIP  sql.NullString
	Status        uint8
	EmailVerified flexTime
	CreatedAt     flexTime
	UpdatedAt     flexTime
}

func (im *importer) importUsers(ctx context.Context) error {
	return im.batch(ctx, "users", "id", []string{
		"id", "group_id", "name", "email", "password", "remember_token", "url",
		"capacity", "configs", "is_adminer", "image_num", "album_num",
		"registered_ip", "status", "email_verified_at", "created_at", "updated_at",
	}, func(rows *sql.Rows) (int64, error) {
		var u legacyUser
		if err := rows.Scan(&u.ID, &u.GroupID, &u.Name, &u.Email, &u.Password, &u.RememberToken,
			&u.URL, &u.Capacity, &u.Configs, &u.IsAdmin, &u.ImageNum, &u.AlbumNum,
			&u.RegisteredIP, &u.Status, &u.EmailVerified, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return 0, err
		}

		// Lsky Pro passwords are bcrypt hashes and stay verifiable.
		capacityBytes := 0.0
		if u.Capacity.valid {
			capacityBytes = u.Capacity.v * 1024
		}

		var existing data.User
		if err := im.tx.Where("email = ?", u.Email).First(&existing).Error; err == nil {
			// Same email already exists (e.g. the admin account created by the
			// installer): merge, i.e. their images are re-pointed to the
			// existing account instead of duplicating it.
			im.userIDMap[u.ID] = existing.ID
			im.summary.UsersMerged++
			return int64(u.ID), nil
		} else if err != gorm.ErrRecordNotFound {
			return 0, err
		}

		newID, err := users.GenerateUserID(im.tx)
		if err != nil {
			return 0, err
		}
		var groupID *uint
		if u.GroupID.Valid {
			id := uint(u.GroupID.Int64)
			groupID = &id
		}
		user := data.User{
			ID:            newID,
			GroupID:       groupID,
			Name:          truncate(u.Name, 128),
			Email:         u.Email,
			PasswordHash:  u.Password,
			RememberToken: u.RememberToken.String,
			URL:           truncate(u.URL, 255),
			Capacity:      capacityBytes,
			CapacityBonus: capacityBytes - im.initialBytes,
			Configs:       mapUserConfigs(u.Configs),
			IsAdmin:       u.IsAdmin,
			Status:        u.Status,
			ImageCount:    uint64(nullInt64Or(u.ImageNum)),
			AlbumCount:    uint64(nullInt64Or(u.AlbumNum)),
			RegisteredIP:  truncate(u.RegisteredIP.String, 64),
			EmailVerified: nullTimePtr(u.EmailVerified),
			CreatedAt:     nullTimeOrNow(u.CreatedAt),
			UpdatedAt:     nullTimeOrNow(u.UpdatedAt),
		}
		if err := im.upsert(&user); err != nil {
			return 0, err
		}
		im.userIDMap[u.ID] = user.ID
		im.summary.Users++
		return int64(u.ID), nil
	})
}

// mapUserConfigs converts the Lsky Pro user config JSON (default_permission as
// 0/1, default_strategy) into the skyImage user config schema.
func mapUserConfigs(raw []byte) datatypes.JSON {
	mapped := map[string]interface{}{}
	var cfg map[string]interface{}
	if err := json.Unmarshal(raw, &cfg); err == nil {
		if v, ok := numberValue(cfg["default_permission"]); ok {
			if int(v) == 1 {
				mapped["default_visibility"] = "public"
			} else {
				mapped["default_visibility"] = "private"
			}
		}
		if v, ok := numberValue(cfg["default_strategy"]); ok && v > 0 {
			mapped["default_strategy"] = int64(v)
		}
	}
	out, _ := json.Marshal(mapped)
	return datatypes.JSON(out)
}

// --- albums ---------------------------------------------------------------

type legacyAlbum struct {
	ID        uint
	UserID    uint
	Name      string
	Intro     string
	ImageNum  sql.NullInt64
	CreatedAt flexTime
	UpdatedAt flexTime
}

func (im *importer) importAlbums(ctx context.Context) error {
	return im.batch(ctx, "albums", "id", []string{"id", "user_id", "name", "intro", "image_num", "created_at", "updated_at"}, func(rows *sql.Rows) (int64, error) {
		var a legacyAlbum
		if err := rows.Scan(&a.ID, &a.UserID, &a.Name, &a.Intro, &a.ImageNum, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return 0, err
		}
		ownerID, ok := im.userIDMap[a.UserID]
		if !ok {
			return int64(a.ID), nil
		}
		album := data.Album{
			ID:        a.ID,
			UserID:    ownerID,
			Name:      truncate(a.Name, 255),
			Intro:     truncate(a.Intro, 512),
			ImageNum:  uint64(nullInt64Or(a.ImageNum)),
			CreatedAt: nullTimeOrNow(a.CreatedAt),
			UpdatedAt: nullTimeOrNow(a.UpdatedAt),
		}
		if err := im.upsert(&album); err != nil {
			return 0, err
		}
		im.summary.Albums++
		return int64(a.ID), nil
	})
}

// --- images ---------------------------------------------------------------

type legacyImage struct {
	ID          uint
	UserID      sql.NullInt64
	GroupID     sql.NullInt64
	StrategyID  sql.NullInt64
	Key         string
	Path        string
	Name        string
	OriginName  sql.NullString
	AliasName   sql.NullString
	Size        flexFloat
	Mimetype    string
	Extension   string
	MD5         sql.NullString
	SHA1        sql.NullString
	Width       int
	Height      int
	Permission  int8
	IsUnhealthy bool
	UploadedIP  sql.NullString
	CreatedAt   flexTime
	UpdatedAt   flexTime
}

func (im *importer) importImages(ctx context.Context) error {
	columns := []string{
		"id", "user_id", "group_id", "strategy_id", "key", "path",
		"name", "origin_name", "alias_name", "size", "mimetype", "extension",
		"md5", "sha1", "width", "height", "permission", "is_unhealthy",
		"uploaded_ip", "created_at", "updated_at",
	}
	return im.batch(ctx, "images", "id", columns, func(rows *sql.Rows) (int64, error) {
		var img legacyImage
		if err := rows.Scan(&img.ID, &img.UserID, &img.GroupID, &img.StrategyID, &img.Key,
			&img.Path, &img.Name, &img.OriginName, &img.AliasName, &img.Size, &img.Mimetype,
			&img.Extension, &img.MD5, &img.SHA1, &img.Width, &img.Height, &img.Permission,
			&img.IsUnhealthy, &img.UploadedIP, &img.CreatedAt, &img.UpdatedAt); err != nil {
			return 0, err
		}
		if err := im.importImage(ctx, img); err != nil {
			return 0, err
		}
		return int64(img.ID), nil
	})
}

// imagePathname 拼 Lsky Pro 图片的 pathname = path/name（相对策略存储根），
// path 可能为空（命名规则不含目录部分）。
func imagePathname(path, name string) string {
	pathname := strings.Trim(path, "/")
	if pathname != "" {
		pathname += "/"
	}
	return strings.TrimPrefix(filepath.ToSlash(pathname+name), "/")
}

// imageStrategy 解析图片所属策略；源库中策略已缺失时回退到 ID 1。
func (im *importer) imageStrategy(strategyID sql.NullInt64) (uint, strategyInfo) {
	id := uint(1)
	if strategyID.Valid {
		if _, ok := im.strategyIDs[uint(strategyID.Int64)]; ok {
			id = uint(strategyID.Int64)
		}
	}
	return id, im.strategies[id]
}

// destImageFile 返回图片在新存储根下的目标绝对路径；未配置存储根时为空。
func (im *importer) destImageFile(pathname string) string {
	if im.opts.StoragePath == "" || pathname == "" {
		return ""
	}
	return filepath.Join(im.opts.StoragePath, filepath.FromSlash(pathname))
}

// precopyImages 在数据库事务之前把旧站图片文件拷贝进新存储：事务阶段因此
// 不做任何文件 I/O，同时路径被禁止/文件缺失的统计也在此阶段完成。
func (im *importer) precopyImages(ctx context.Context) error {
	columns := []string{"id", "strategy_id", "path", "name"}
	return im.batch(ctx, "images", "id", columns, func(rows *sql.Rows) (int64, error) {
		var img struct {
			ID         uint
			StrategyID sql.NullInt64
			Path       string
			Name       string
		}
		if err := rows.Scan(&img.ID, &img.StrategyID, &img.Path, &img.Name); err != nil {
			return 0, err
		}
		_, info := im.imageStrategy(img.StrategyID)
		pathname := imagePathname(img.Path, img.Name)
		cleanPath, ok := sanitizeImportRelPath(pathname)
		if !ok {
			im.summary.FilesForbidden++
			if len(im.forbiddenPaths) < 10 {
				im.forbiddenPaths = append(im.forbiddenPaths, fmt.Sprintf("%s（路径被禁止导入）", pathname))
			}
			return int64(img.ID), nil
		}
		if _, copied := im.copyImageFile(info.Root, cleanPath); copied {
			im.summary.FilesCopied++
		} else {
			im.summary.FilesMissing++
		}
		return int64(img.ID), nil
	})
}

func (im *importer) importImage(_ context.Context, img legacyImage) error {
	pathname := imagePathname(img.Path, img.Name)

	ownerID := im.opts.OrphanUserID
	if img.UserID.Valid {
		if mapped, ok := im.userIDMap[uint(img.UserID.Int64)]; ok {
			ownerID = mapped
		}
	}

	var groupID *uint
	if img.GroupID.Valid {
		id := uint(img.GroupID.Int64)
		groupID = &id
	}

	strategyID, info := im.imageStrategy(img.StrategyID)

	// 图片路径保持原样（path/name），但必须通过安全校验；被禁止的路径
	// 不写入数据库（已在 precopyImages 中统计并报告）。
	cleanPath, ok := sanitizeImportRelPath(pathname)
	if !ok {
		return nil
	}

	sizeBytes := int64(0)
	if img.Size.valid {
		sizeBytes = int64(img.Size.v * 1024)
	}

	// 文件已在事务前的 precopyImages 阶段拷贝完成，这里只探测结果：
	// 文件已就位则用新站点 URL，否则保留旧站 URL（文件仍可从旧地址访问）。
	relPath := filepath.ToSlash(cleanPath)
	destPath := im.destImageFile(cleanPath)
	fileInPlace := false
	if destPath != "" {
		if _, err := os.Stat(destPath); err == nil {
			fileInPlace = true
		}
	}
	publicURL := joinURL(info.URL, relPath)
	if fileInPlace {
		publicURL = joinURL(im.opts.PublicBaseURL, relPath)
	}

	visibility := "private"
	if img.Permission == 1 {
		visibility = "public"
	}
	auditStatus := "none"
	if img.IsUnhealthy {
		auditStatus = "rejected"
	}

	recordPath := ""
	if fileInPlace {
		recordPath = destPath
	}
	file := data.FileAsset{
		ID:              img.ID,
		UserID:          ownerID,
		GroupID:         groupID,
		StrategyID:      strategyID,
		Key:             img.Key,
		Path:            recordPath,
		RelativePath:    relPath,
		PublicURL:       publicURL,
		Name:            truncate(img.Name, 255),
		OriginalName:    truncate(img.OriginName.String, 255),
		Size:            sizeBytes,
		MimeType:        truncate(img.Mimetype, 64),
		Extension:       truncate(img.Extension, 32),
		ChecksumMD5:     img.MD5.String,
		ChecksumSHA1:    img.SHA1.String,
		Width:           img.Width,
		Height:          img.Height,
		Visibility:      visibility,
		StorageProvider: "local",
		AuditStatus:     auditStatus,
		UploadedIP:      truncate(img.UploadedIP.String, 64),
		CreatedAt:       nullTimeOrNow(img.CreatedAt),
		UpdatedAt:       nullTimeOrNow(img.UpdatedAt),
	}
	if err := im.upsert(&file); err != nil {
		return err
	}
	im.summary.Files++
	return nil
}

// sanitizeImportRelPath validates a Lsky Pro image relative path before it is
// used in the new storage. It returns false for paths that must not be
// imported: traversal attempts, absolute/drive paths, control characters, and
// first segments that collide with reserved system paths (api/assets 等，
// 这些路径在 skyImage 中有特殊用途，图片放进去无法正常访问).
func sanitizeImportRelPath(pathname string) (string, bool) {
	p := strings.TrimSpace(filepath.ToSlash(strings.TrimSpace(pathname)))
	if p == "" || strings.HasPrefix(p, "/") {
		return "", false
	}
	if strings.ContainsRune(p, ':') { // Windows drive letter like C:/...
		return "", false
	}
	for _, r := range p {
		if r < 0x20 || r == 0x7f {
			return "", false
		}
	}
	segments := strings.Split(p, "/")
	reserved := map[string]struct{}{
		"api":        {},
		"assets":     {},
		"robots.txt": {},
	}
	for i, seg := range segments {
		if seg == "" || seg == "." || seg == ".." {
			return "", false
		}
		if i == 0 {
			if _, banned := reserved[strings.ToLower(seg)]; banned {
				return "", false
			}
		}
	}
	return p, true
}

// copyImageFile copies the physical file of one image into the new storage
// root, keeping the original relative path unchanged. It probes plausible
// source locations:
//  1. the 图片目录 provided by the user (the original storage/app/uploads)
//  2. the absolute root configured in the Lsky Pro strategy
func (im *importer) copyImageFile(strategyRoot, pathname string) (string, bool) {
	destPath := im.destImageFile(pathname)
	if destPath == "" {
		return "", false
	}
	slashPath := filepath.FromSlash(pathname)

	candidates := make([]string, 0, 2)
	if im.opts.ImageDir != "" {
		candidates = append(candidates, filepath.Join(im.opts.ImageDir, slashPath))
	}
	if strategyRoot != "" && filepath.IsAbs(strategyRoot) {
		candidates = append(candidates, filepath.Join(strategyRoot, slashPath))
	}
	if len(candidates) == 0 {
		return "", false
	}

	var srcPath string
	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			srcPath = candidate
			break
		}
	}
	if srcPath == "" {
		return "", false
	}

	if _, err := os.Stat(destPath); err == nil {
		return destPath, true
	}
	if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		return "", false
	}
	if err := copyFileContents(srcPath, destPath); err != nil {
		return "", false
	}
	return destPath, true
}

func copyFileContents(src, dest string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}

// --- settings -------------------------------------------------------------

// importSettings converts selected Lsky Pro system settings (configs table)
// into skyImage settings keys. Names not recognised are intentionally ignored.
func (im *importer) importSettings(ctx context.Context) error {
	rows, err := im.q.QueryContext(ctx, "SELECT "+im.srcCfg.quote("name")+", "+im.srcCfg.quote("value")+
		" FROM "+im.srcCfg.quote(im.srcCfg.table("configs")))
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var name string
		var value sql.NullString
		if err := rows.Scan(&name, &value); err != nil {
			return err
		}
		for _, kv := range mapSetting(name, value.String) {
			if err := upsertSetting(im.tx, kv[0], kv[1]); err != nil {
				return err
			}
			im.summary.Settings++
		}
	}
	return rows.Err()
}

func mapSetting(name, value string) [][2]string {
	boolStr := func(v string) string {
		v = strings.TrimSpace(v)
		if v == "1" || strings.EqualFold(v, "true") {
			return "true"
		}
		return "false"
	}
	switch name {
	case "app_name":
		if v := strings.TrimSpace(value); v != "" {
			return [][2]string{{"site.name", v}, {"site.title", v}}
		}
	case "site_description":
		return [][2]string{{"site.description", value}}
	case "is_enable_gallery":
		return [][2]string{{"features.gallery", boolStr(value)}}
	case "is_enable_api":
		return [][2]string{{"features.api", boolStr(value)}}
	case "is_enable_registration":
		return [][2]string{{"features.allow_registration", boolStr(value)}}
	case "is_user_need_verify":
		return [][2]string{{"mail.register.verify", boolStr(value)}}
	case "mail":
		var mail struct {
			Mailers struct {
				SMTP struct {
					Host       string      `json:"host"`
					Port       interface{} `json:"port"`
					Encryption string      `json:"encryption"`
					Username   string      `json:"username"`
					Password   string      `json:"password"`
				} `json:"smtp"`
			} `json:"mailers"`
		}
		if err := json.Unmarshal([]byte(value), &mail); err != nil {
			return nil
		}
		smtp := mail.Mailers.SMTP
		if smtp.Host == "" || smtp.Username == "" {
			return nil
		}
		secure := "false"
		switch strings.ToLower(smtp.Encryption) {
		case "tls", "ssl":
			secure = "true"
		}
		return [][2]string{
			{"mail.smtp.host", smtp.Host},
			{"mail.smtp.port", strings.TrimSpace(fmt.Sprintf("%v", smtp.Port))},
			{"mail.smtp.username", smtp.Username},
			{"mail.smtp.password", smtp.Password},
			{"mail.smtp.from", smtp.Username},
			{"mail.smtp.secure", secure},
		}
	}
	return nil
}

func upsertSetting(tx *gorm.DB, key, value string) error {
	entry := data.ConfigEntry{Key: key, Value: value}
	return tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "key"}},
		DoUpdates: clause.Assignments(map[string]interface{}{"value": value, "updated_at": time.Now()}),
	}).Create(&entry).Error
}

// --- helpers --------------------------------------------------------------

// batch streams rows of one table ordered by the key column in pages, so very
// large Lsky Pro databases are imported with bounded memory. The callback
// returns the key of the processed row. Each page is capped at 10 minutes but
// keeps the parent ctx: 取消外层请求（如客户端断开）会中止导入。
func (im *importer) batch(ctx context.Context, table, keyColumn string, columns []string, fn func(*sql.Rows) (int64, error)) error {
	lastID := int64(0)
	for {
		query := im.srcCfg.pageQuery(table, keyColumn, columns, batchSize)
		pageCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
		rows, err := im.q.QueryContext(pageCtx, query, lastID)
		if err != nil {
			cancel()
			if isMissingTableErr(err) {
				return nil
			}
			return err
		}
		count := 0
		var scanErr error
		var maxSeen int64
		for rows.Next() {
			var id int64
			id, scanErr = fn(rows)
			if scanErr != nil {
				break
			}
			if id > maxSeen {
				maxSeen = id
			}
			count++
		}
		if scanErr == nil {
			scanErr = rows.Err()
		}
		rows.Close()
		cancel()
		if scanErr != nil {
			return scanErr
		}
		if count < batchSize || maxSeen <= lastID {
			return nil
		}
		lastID = maxSeen
	}
}

// upsert inserts the record, replacing an existing row with the same primary
// key (imports are idempotent and can be re-run after a partial run).
func (im *importer) upsert(model interface{}) error {
	return im.tx.Clauses(clause.OnConflict{UpdateAll: true}).Create(model).Error
}

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n])
}

func nullTimeOrNow(t flexTime) time.Time {
	if t.valid && !t.t.IsZero() {
		return t.t
	}
	return time.Now()
}

func nullTimePtr(t flexTime) *time.Time {
	if t.valid && !t.t.IsZero() {
		return &t.t
	}
	return nil
}

// flexTime scans timestamps from both MySQL (time.Time via parseTime) and
// SQLite (datetime strings) sources.
type flexTime struct {
	valid bool
	t     time.Time
}

func (f *flexTime) Scan(src interface{}) error {
	switch v := src.(type) {
	case nil:
		f.valid = false
	case time.Time:
		f.t, f.valid = v, !v.IsZero()
	case []byte:
		return f.parse(string(v))
	case string:
		return f.parse(v)
	case interface{ String() string }:
		return f.parse(v.String())
	default:
		return fmt.Errorf("无法读取时间字段类型 %T", src)
	}
	return nil
}

func (f *flexTime) parse(s string) error {
	s = strings.TrimSpace(s)
	if s == "" {
		f.valid = false
		return nil
	}
	layouts := []string{
		"2006-01-02 15:04:05.999999999 -07:00",
		"2006-01-02 15:04:05.999999999",
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05",
		time.RFC3339,
	}
	for _, layout := range layouts {
		if t, err := time.ParseInLocation(layout, s, time.Local); err == nil {
			f.t, f.valid = t, true
			return nil
		}
	}
	return fmt.Errorf("无法解析时间 %q", s)
}

// flexFloat scans numeric fields that may arrive as float64 (MySQL decimal),
// int64 (SQLite numeric) or text.
type flexFloat struct {
	valid bool
	v     float64
}

func (f *flexFloat) Scan(src interface{}) error {
	switch v := src.(type) {
	case nil:
		f.valid = false
	case float64:
		f.v, f.valid = v, true
	case int64:
		f.v, f.valid = float64(v), true
	case []byte:
		return f.parse(string(v))
	case string:
		return f.parse(v)
	case interface{ String() string }:
		// 例如 PostgreSQL 的 numeric 在部分驱动路径下以此形式返回。
		return f.parse(v.String())
	default:
		return fmt.Errorf("无法读取数值字段类型 %T", src)
	}
	return nil
}

func (f *flexFloat) parse(s string) error {
	s = strings.TrimSpace(s)
	if s == "" {
		f.valid = false
		return nil
	}
	parsed, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return fmt.Errorf("无法解析数值 %q", s)
	}
	f.v, f.valid = parsed, true
	return nil
}

func nullInt64Or(v sql.NullInt64) int64 {
	if v.Valid {
		return v.Int64
	}
	return 0
}

func joinURL(base, path string) string {
	base = strings.TrimRight(strings.TrimSpace(base), "/")
	path = strings.TrimLeft(path, "/")
	if base == "" || path == "" {
		return path
	}
	return base + "/" + path
}

func isMissingTableErr(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	// MySQL error 1146: table doesn't exist; SQLite: no such table;
	// PostgreSQL: ... does not exist (SQLSTATE 42P01); SQL Server:
	// Invalid object name.
	return strings.Contains(msg, "doesn't exist") ||
		strings.Contains(msg, "does not exist") ||
		strings.Contains(msg, "Error 1146") ||
		strings.Contains(msg, "42P01") ||
		strings.Contains(msg, "Invalid object name") ||
		strings.Contains(msg, "no such table")
}
