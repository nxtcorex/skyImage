package admin

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"skyimage/internal/data"
)

type Service struct {
	db *gorm.DB
}

func New(db *gorm.DB) *Service {
	return &Service{db: db}
}

type DashboardMetrics struct {
	UserCount     int64            `json:"userCount"`
	FileCount     int64            `json:"fileCount"`
	StorageUsed   int64            `json:"storageUsed"`
	LastUploadAt  *time.Time       `json:"lastUploadAt"`
	RecentUploads []data.FileAsset `json:"recentUploads"`
}

type TrendData struct {
	Date          string `json:"date"`
	Uploads       int64  `json:"uploads"`
	Registrations int64  `json:"registrations"`
}

func (s *Service) Dashboard(ctx context.Context) (DashboardMetrics, error) {
	var metrics DashboardMetrics
	if err := s.db.WithContext(ctx).Model(&data.User{}).Count(&metrics.UserCount).Error; err != nil {
		return metrics, err
	}
	if err := s.db.WithContext(ctx).Model(&data.FileAsset{}).Count(&metrics.FileCount).Error; err != nil {
		return metrics, err
	}
	if err := s.db.WithContext(ctx).Model(&data.FileAsset{}).Select("COALESCE(SUM(size),0)").Scan(&metrics.StorageUsed).Error; err != nil {
		return metrics, err
	}
	var last data.FileAsset
	if err := s.db.WithContext(ctx).Order("created_at DESC").First(&last).Error; err == nil {
		metrics.LastUploadAt = &last.CreatedAt
	}
	if err := s.db.WithContext(ctx).Order("created_at DESC").Limit(5).Find(&metrics.RecentUploads).Error; err != nil {
		return metrics, err
	}
	return metrics, nil
}

func (s *Service) GetTrends(ctx context.Context, days int) ([]TrendData, error) {
	if days <= 0 {
		days = 90
	}
	if days > 365 {
		days = 365
	}

	startDate := time.Now().AddDate(0, 0, -days).Format("2006-01-02")

	trends := make([]TrendData, 0, 365)
	now := time.Now()
	for i := days - 1; i >= 0; i-- {
		date := now.AddDate(0, 0, -i).Format("2006-01-02")
		trends = append(trends, TrendData{
			Date:          date,
			Uploads:       0,
			Registrations: 0,
		})
	}

	// 查询上传数据
	type DailyCount struct {
		Date  string
		Count int64
	}

	var uploadCounts []DailyCount
	err := s.db.WithContext(ctx).
		Model(&data.FileAsset{}).
		Select("DATE(created_at) as date, COUNT(*) as count").
		Where("DATE(created_at) >= ?", startDate).
		Group("DATE(created_at)").
		Scan(&uploadCounts).Error
	if err != nil {
		return nil, err
	}

	// 查询注册数据
	var registrationCounts []DailyCount
	err = s.db.WithContext(ctx).
		Model(&data.User{}).
		Select("DATE(created_at) as date, COUNT(*) as count").
		Where("DATE(created_at) >= ?", startDate).
		Group("DATE(created_at)").
		Scan(&registrationCounts).Error
	if err != nil {
		return nil, err
	}

	// 填充数据
	uploadMap := make(map[string]int64)
	for _, uc := range uploadCounts {
		uploadMap[uc.Date] = uc.Count
	}

	registrationMap := make(map[string]int64)
	for _, rc := range registrationCounts {
		registrationMap[rc.Date] = rc.Count
	}

	for i := range trends {
		if count, ok := uploadMap[trends[i].Date]; ok {
			trends[i].Uploads = count
		}
		if count, ok := registrationMap[trends[i].Date]; ok {
			trends[i].Registrations = count
		}
	}

	return trends, nil
}

func (s *Service) GetSettings(ctx context.Context) (map[string]string, error) {
	var entries []data.ConfigEntry
	if err := s.db.WithContext(ctx).Find(&entries).Error; err != nil {
		return nil, err
	}
	settings := make(map[string]string, len(entries))
	for _, entry := range entries {
		settings[entry.Key] = entry.Value
	}
	return settings, nil
}

func (s *Service) UpdateSettings(ctx context.Context, kv map[string]string) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for key, value := range kv {
			entry := data.ConfigEntry{
				Key:   key,
				Value: value,
			}
			if err := tx.Clauses(clause.OnConflict{
				Columns: []clause.Column{{Name: "key"}},
				DoUpdates: clause.Assignments(map[string]interface{}{
					"value":      value,
					"updated_at": gorm.Expr("CURRENT_TIMESTAMP"),
				}),
			}).Create(&entry).Error; err != nil {
				return fmt.Errorf("save config %s: %w", key, err)
			}
		}
		return nil
	})
}

type GroupPayload struct {
	Name      string                 `json:"name"`
	IsDefault bool                   `json:"isDefault"`
	IsGuest   bool                   `json:"isGuest"`
	Configs   map[string]interface{} `json:"configs"`
}

func (s *Service) ListGroups(ctx context.Context) ([]data.Group, error) {
	var groups []data.Group
	err := s.db.WithContext(ctx).Order("id ASC").Find(&groups).Error
	return groups, err
}

func (s *Service) CreateGroup(ctx context.Context, payload GroupPayload) (data.Group, error) {
	// Validate configs
	if err := validateGroupConfigs(payload.Configs); err != nil {
		return data.Group{}, err
	}

	cfgBytes, _ := json.Marshal(payload.Configs)
	group := data.Group{
		Name:      payload.Name,
		IsDefault: payload.IsDefault,
		IsGuest:   payload.IsGuest,
		Configs:   datatypes.JSON(cfgBytes),
	}
	err := s.db.WithContext(ctx).Create(&group).Error
	if err != nil {
		return group, err
	}
	if payload.IsDefault {
		if err := s.ensureSingleDefaultGroup(ctx, group.ID); err != nil {
			return group, err
		}
	}
	return group, nil
}

func (s *Service) UpdateGroup(ctx context.Context, id uint, payload GroupPayload) (data.Group, error) {
	// Validate configs
	if err := validateGroupConfigs(payload.Configs); err != nil {
		return data.Group{}, err
	}

	group := data.Group{}
	if err := s.db.WithContext(ctx).First(&group, id).Error; err != nil {
		return group, err
	}
	cfgBytes, _ := json.Marshal(payload.Configs)
	group.Name = payload.Name
	group.IsGuest = payload.IsGuest
	group.IsDefault = payload.IsDefault
	group.Configs = datatypes.JSON(cfgBytes)
	if err := s.db.WithContext(ctx).Save(&group).Error; err != nil {
		return group, err
	}
	if payload.IsDefault {
		if err := s.ensureSingleDefaultGroup(ctx, group.ID); err != nil {
			return group, err
		}
	}
	return group, nil
}

func (s *Service) DeleteGroup(ctx context.Context, id uint) error {
	return s.db.WithContext(ctx).Delete(&data.Group{}, id).Error
}

func (s *Service) ensureSingleDefaultGroup(ctx context.Context, id uint) error {
	return s.db.WithContext(ctx).
		Model(&data.Group{}).
		Where("id <> ?", id).
		Update("is_default", false).Error
}

type StrategyPayload struct {
	Key      uint8                  `json:"key"`
	Name     string                 `json:"name"`
	Intro    string                 `json:"intro"`
	Configs  map[string]interface{} `json:"configs"`
	GroupIDs []uint                 `json:"groupIds"`
}

func (s *Service) ListStrategies(ctx context.Context) ([]data.Strategy, error) {
	var items []data.Strategy
	err := s.db.WithContext(ctx).
		Preload("Groups").
		Order("id ASC").
		Find(&items).Error
	return items, err
}

func (s *Service) FindStrategyByID(ctx context.Context, id uint) (data.Strategy, error) {
	var strategy data.Strategy
	err := s.db.WithContext(ctx).
		Preload("Groups").
		First(&strategy, id).Error
	return strategy, err
}

func (s *Service) CreateStrategy(ctx context.Context, payload StrategyPayload) (data.Strategy, error) {
	if err := validateStrategyConfigs(payload.Configs); err != nil {
		return data.Strategy{}, err
	}
	if err := s.ensureAuditProfileExistsInConfigs(ctx, payload.Configs); err != nil {
		return data.Strategy{}, err
	}
	cfgBytes, _ := json.Marshal(payload.Configs)
	strategy := data.Strategy{
		Key:     payload.Key,
		Name:    payload.Name,
		Intro:   payload.Intro,
		Configs: datatypes.JSON(cfgBytes),
	}
	err := s.db.WithContext(ctx).Create(&strategy).Error
	if err != nil {
		return strategy, err
	}
	if err := s.replaceStrategyGroups(ctx, strategy.ID, payload.GroupIDs); err != nil {
		return strategy, err
	}
	return strategy, nil
}

func (s *Service) UpdateStrategy(ctx context.Context, id uint, payload StrategyPayload) (data.Strategy, error) {
	if err := validateStrategyConfigs(payload.Configs); err != nil {
		return data.Strategy{}, err
	}
	if err := s.ensureAuditProfileExistsInConfigs(ctx, payload.Configs); err != nil {
		return data.Strategy{}, err
	}
	var strategy data.Strategy
	if err := s.db.WithContext(ctx).First(&strategy, id).Error; err != nil {
		return strategy, err
	}
	cfgBytes, _ := json.Marshal(payload.Configs)
	strategy.Key = payload.Key
	strategy.Name = payload.Name
	strategy.Intro = payload.Intro
	strategy.Configs = datatypes.JSON(cfgBytes)
	if err := s.db.WithContext(ctx).Save(&strategy).Error; err != nil {
		return strategy, err
	}
	if err := s.replaceStrategyGroups(ctx, strategy.ID, payload.GroupIDs); err != nil {
		return strategy, err
	}
	return strategy, nil
}

func (s *Service) DeleteStrategy(ctx context.Context, id uint) error {
	return s.db.WithContext(ctx).Delete(&data.Strategy{}, id).Error
}

func (s *Service) ListAllFiles(ctx context.Context, limit, offset int, auditStatus string) ([]data.FileAsset, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	var files []data.FileAsset
	query := s.db.WithContext(ctx).
		Preload("User").
		Preload("Strategy").
		Order("created_at DESC")
	normalizedAuditStatus := normalizeAuditStatusFilter(auditStatus)
	if normalizedAuditStatus != "" {
		query = query.Where("audit_status = ?", normalizedAuditStatus)
	}
	err := query.Limit(limit).Offset(offset).Find(&files).Error
	return files, err
}

func (s *Service) DeleteFile(ctx context.Context, id uint) error {
	return s.db.WithContext(ctx).Delete(&data.FileAsset{}, id).Error
}

func (s *Service) replaceStrategyGroups(ctx context.Context, strategyID uint, groupIDs []uint) error {
	ids := uniqueUint(groupIDs)
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("strategy_id = ?", strategyID).Delete(&data.GroupStrategy{}).Error; err != nil {
			return err
		}
		if len(ids) == 0 {
			return nil
		}
		for _, id := range ids {
			link := data.GroupStrategy{GroupID: id, StrategyID: strategyID}
			if err := tx.Create(&link).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func uniqueUint(values []uint) []uint {
	seen := make(map[uint]struct{}, len(values))
	result := make([]uint, 0, len(values))
	for _, v := range values {
		if v == 0 {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		result = append(result, v)
	}
	return result
}

func validateGroupConfigs(configs map[string]interface{}) error {
	if configs == nil {
		return nil
	}

	// Validate max_file_size
	if maxFileSize, ok := configs["max_file_size"]; ok {
		var size float64
		switch v := maxFileSize.(type) {
		case float64:
			size = v
		case int:
			size = float64(v)
		case int64:
			size = float64(v)
		default:
			return fmt.Errorf("max_file_size 必须是数字")
		}
		if size < 0 {
			return fmt.Errorf("最大单文件大小必须大于等于 0")
		}
	}

	// Validate max_capacity
	if maxCapacity, ok := configs["max_capacity"]; ok {
		var capacity float64
		switch v := maxCapacity.(type) {
		case float64:
			capacity = v
		case int:
			capacity = float64(v)
		case int64:
			capacity = float64(v)
		default:
			return fmt.Errorf("max_capacity 必须是数字")
		}
		if capacity < 0 {
			return fmt.Errorf("容量上限必须大于等于 0")
		}
	}

	// Validate upload_rate_minute
	if raw, ok := configs["upload_rate_minute"]; ok {
		limit, err := asPositiveInt(raw)
		if err != nil {
			return fmt.Errorf("upload_rate_minute 必须是数字")
		}
		if limit < 0 {
			return fmt.Errorf("upload_rate_minute 必须大于等于 0")
		}
	}

	// Validate upload_rate_hour
	if raw, ok := configs["upload_rate_hour"]; ok {
		limit, err := asPositiveInt(raw)
		if err != nil {
			return fmt.Errorf("upload_rate_hour 必须是数字")
		}
		if limit < 0 {
			return fmt.Errorf("upload_rate_hour 必须大于等于 0")
		}
	}

	return nil
}

func validateStrategyConfigs(configs map[string]interface{}) error {
	if configs == nil {
		return nil
	}
	driver := strings.ToLower(strings.TrimSpace(firstConfigString(configs, "driver")))
	if driver == "" {
		driver = "local"
	}
	for _, rawURL := range configStrings(configs, "url", "base_url", "baseUrl") {
		for _, item := range splitExternalDomains(rawURL) {
			if err := validateExternalDomain(item); err != nil {
				return err
			}
			if segment := extractPathPrefix(item); segment != "" {
				if _, blocked := reservedStrategyPathSegments()[strings.ToLower(segment)]; blocked {
					return fmt.Errorf("路径前缀 %q 为系统保留路径，不能用于存储策略", segment)
				}
			}
		}
	}
	if configBool(configs, "image_only_domain") && len(configDomainHosts(configs)) == 0 {
		return fmt.Errorf("开启「域名仅限图片访问」前需要先为该存储策略配置外部访问域名")
	}
	if driver == "webdav" {
		if err := validateWebDAVConfigs(configs); err != nil {
			return err
		}
	}
	if driver == "ftp" {
		if err := validateFTPConfigs(configs); err != nil {
			return err
		}
	}
	if driver == "sftp" {
		if err := validateSFTPConfigs(configs); err != nil {
			return err
		}
	}
	if driver == "s3" || driver == "minio" {
		if strings.TrimSpace(firstConfigString(configs, "s3_bucket")) == "" {
			return fmt.Errorf("s3_bucket 不能为空")
		}
		if strings.TrimSpace(firstConfigString(configs, "s3_access_key")) == "" {
			return fmt.Errorf("s3_access_key 不能为空")
		}
		if strings.TrimSpace(firstConfigString(configs, "s3_secret_key")) == "" {
			return fmt.Errorf("s3_secret_key 不能为空")
		}
	}
	template := ""
	if values := configStrings(configs, "path_template", "pattern"); len(values) > 0 {
		template = strings.TrimSpace(values[0])
	}
	if template != "" && !strings.Contains(template, "{uuid}") {
		return fmt.Errorf("路径模板必须包含 {uuid} 以确保唯一性")
	}
	// Thumbnails use a fixed "*_thumb.*" suffix on relative paths; forbid that token in the template.
	lowerTemplate := strings.ToLower(template)
	if template != "" && strings.Contains(lowerTemplate, "_thumb.") {
		return fmt.Errorf("路径模板不能包含 _thumb.，该后缀保留给系统缩略图")
	}
	// Ticket attachments use a reserved "tickets/" path prefix.
	if template != "" && strings.Contains(lowerTemplate, "tickets/") {
		return fmt.Errorf("路径模板不能包含 tickets/，该前缀保留给工单附件")
	}
	if template != "" && strings.Contains(lowerTemplate, "_ticket.") {
		return fmt.Errorf("路径模板不能包含 _ticket.，该后缀保留给系统工单附件")
	}
	if raw, ok := configs["image_audit_block_action"]; ok {
		action := strings.ToLower(strings.TrimSpace(firstConfigString(map[string]interface{}{"value": raw}, "value")))
		if action != "" && action != "delete" && action != "keep" {
			return fmt.Errorf("image_audit_block_action 仅支持 delete 或 keep")
		}
	}
	if raw, ok := configs["image_audit_error_action"]; ok {
		action := strings.ToLower(strings.TrimSpace(firstConfigString(map[string]interface{}{"value": raw}, "value")))
		if action != "" && action != "delete" && action != "keep" {
			return fmt.Errorf("image_audit_error_action 仅支持 delete 或 keep")
		}
	}
	return nil
}

func reservedStrategyPathSegments() map[string]struct{} {
	return map[string]struct{}{
		"api":             {},
		"assets":          {},
		"forgot-password": {},
		"reset-password":  {},
		"u":               {},
	}
}

func extractPathPrefix(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if strings.HasPrefix(raw, "//") {
		raw = "http:" + raw
	}
	if strings.HasPrefix(raw, "/") {
		return strings.Trim(strings.Trim(raw, "/"), "/")
	}
	if strings.Contains(raw, "://") {
		u, err := url.Parse(raw)
		if err == nil {
			return strings.Trim(strings.Trim(u.Path, "/"), "/")
		}
	}
	if looksLikeHost(raw) {
		u, err := url.Parse("http://" + raw)
		if err == nil {
			return strings.Trim(strings.Trim(u.Path, "/"), "/")
		}
	}
	return strings.Trim(strings.Trim(raw, "/"), "/")
}

func splitExternalDomains(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	// Multiple domains are separated by semicolon (ASCII or fullwidth).
	raw = strings.ReplaceAll(raw, "；", ";")
	parts := strings.Split(raw, ";")
	out := make([]string, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))
	for _, item := range parts {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		key := strings.ToLower(item)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, item)
	}
	return out
}

func validateExternalDomain(raw string) error {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil
	}
	// Pure relative path (e.g. /uploads) is not allowed as a domain entry.
	if strings.HasPrefix(trimmed, "/") && !strings.HasPrefix(trimmed, "//") {
		return fmt.Errorf("外部访问域名仅支持域名，不允许包含路径: %s", trimmed)
	}
	normalized := trimmed
	if strings.HasPrefix(normalized, "//") {
		normalized = "http:" + normalized
	}
	if !strings.Contains(normalized, "://") {
		// bare host[:port]
		if looksLikeHost(normalized) {
			normalized = "http://" + normalized
		} else {
			return fmt.Errorf("外部访问域名格式不正确: %s", trimmed)
		}
	}
	parsed, err := url.Parse(normalized)
	if err != nil || parsed.Host == "" {
		return fmt.Errorf("外部访问域名格式不正确: %s", trimmed)
	}
	// Reject path/query/fragment. Empty path or "/" is fine.
	if path := strings.TrimSpace(parsed.EscapedPath()); path != "" && path != "/" {
		return fmt.Errorf("外部访问域名仅支持域名，不允许包含路径: %s", trimmed)
	}
	if parsed.RawQuery != "" || parsed.Fragment != "" {
		return fmt.Errorf("外部访问域名不允许包含参数或片段: %s", trimmed)
	}
	return nil
}

func looksLikeHost(raw string) bool {
	lower := strings.ToLower(raw)
	return strings.Contains(raw, ".") || strings.Contains(raw, ":") || strings.HasPrefix(lower, "localhost")
}

func configBool(configs map[string]interface{}, key string) bool {
	switch v := configs[key].(type) {
	case bool:
		return v
	case string:
		normalized := strings.ToLower(strings.TrimSpace(v))
		return normalized == "true" || normalized == "1" || normalized == "yes" || normalized == "on"
	default:
		return false
	}
}

// configDomainHosts returns the hosts a strategy binds for external file access; relative path
// prefixes are not domains and are skipped.
func configDomainHosts(configs map[string]interface{}) []string {
	out := make([]string, 0, 2)
	for _, rawURL := range configStrings(configs, "url", "base_url", "baseUrl") {
		for _, item := range splitExternalDomains(rawURL) {
			normalized := strings.TrimSpace(item)
			if strings.HasPrefix(normalized, "/") {
				continue
			}
			if !strings.Contains(normalized, "://") {
				if !looksLikeHost(normalized) {
					continue
				}
				normalized = "http://" + normalized
			}
			if parsed, err := url.Parse(normalized); err == nil && parsed.Host != "" {
				out = append(out, strings.ToLower(parsed.Host))
			}
		}
	}
	return out
}

func firstConfigString(configs map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		if v, ok := configs[key]; ok {
			if s, ok := v.(string); ok {
				return s
			}
		}
	}
	return ""
}

func configStrings(configs map[string]interface{}, keys ...string) []string {
	values := make([]string, 0, len(keys))
	for _, key := range keys {
		if v, ok := configs[key]; ok {
			if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
				values = append(values, s)
			}
		}
	}
	return values
}

func asPositiveInt(value interface{}) (int, error) {
	switch v := value.(type) {
	case float64:
		return int(v), nil
	case int:
		return v, nil
	case int64:
		return int(v), nil
	default:
		return 0, fmt.Errorf("invalid number")
	}
}

func validateWebDAVConfigs(configs map[string]interface{}) error {
	endpoint := strings.TrimSpace(firstConfigString(configs, "webdav_endpoint", "webdav_url", "webdavUrl"))
	if endpoint == "" {
		return fmt.Errorf("WebDAV endpoint 不能为空")
	}
	parsed, err := url.Parse(endpoint)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return fmt.Errorf("WebDAV endpoint 格式不正确")
	}
	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" {
		return fmt.Errorf("WebDAV endpoint 仅支持 http/https")
	}
	return nil
}

func validateFTPConfigs(configs map[string]interface{}) error {
	host := strings.TrimSpace(firstConfigString(configs, "ftp_host", "ftp_endpoint"))
	if host == "" {
		return fmt.Errorf("FTP 主机不能为空")
	}
	normalized := host
	if !strings.Contains(normalized, "://") {
		normalized = "ftp://" + normalized
	}
	parsed, err := url.Parse(normalized)
	if err != nil || parsed.Host == "" {
		return fmt.Errorf("FTP 主机格式不正确")
	}
	if parsed.Scheme != "" {
		scheme := strings.ToLower(parsed.Scheme)
		if scheme != "ftp" && scheme != "ftps" {
			return fmt.Errorf("FTP 仅支持 ftp/ftps 协议")
		}
	}
	if port := strings.TrimSpace(firstConfigString(configs, "ftp_port")); port != "" {
		if _, err := strconv.Atoi(port); err != nil {
			return fmt.Errorf("FTP 端口格式不正确")
		}
	}
	return nil
}

func validateSFTPConfigs(configs map[string]interface{}) error {
	host := strings.TrimSpace(firstConfigString(configs, "sftp_host", "sftp_endpoint"))
	if host == "" {
		return fmt.Errorf("SFTP 主机不能为空")
	}
	normalized := host
	if !strings.Contains(normalized, "://") {
		normalized = "ssh://" + normalized
	}
	parsed, err := url.Parse(normalized)
	if err != nil || parsed.Host == "" {
		return fmt.Errorf("SFTP 主机格式不正确")
	}
	if parsed.Scheme != "" {
		scheme := strings.ToLower(parsed.Scheme)
		if scheme != "ssh" && scheme != "sftp" {
			return fmt.Errorf("SFTP 仅支持 ssh/sftp 协议")
		}
	}
	password := strings.TrimSpace(firstConfigString(configs, "sftp_password", "sftp_pass"))
	privateKey := strings.TrimSpace(firstConfigString(configs, "sftp_private_key", "sftpPrivateKey"))
	if password == "" && privateKey == "" {
		return fmt.Errorf("SFTP 至少需要密码或私钥进行认证")
	}
	if port := strings.TrimSpace(firstConfigString(configs, "sftp_port")); port != "" {
		if _, err := strconv.Atoi(port); err != nil {
			return fmt.Errorf("SFTP 端口格式不正确")
		}
	}
	return nil
}

func normalizeAuditStatusFilter(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "all":
		return ""
	case "none", "approved", "pending", "rejected", "error":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return ""
	}
}
