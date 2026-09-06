package installer

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"skyimage/internal/data"
	"skyimage/internal/legacy"
)

func closeSQL(db *sql.DB) {
	if db != nil {
		_ = db.Close()
	}
}

func verifyPassword(hash, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}

// LegacySourceInput describes the read-only source database of an existing
// Lsky Pro (open source) installation.
type LegacySourceInput struct {
	// DatabaseType 源数据库类型：mysql（默认，数据库连接）或
	// sqlite（源数据库目录）。
	DatabaseType string `json:"databaseType"`
	// MySQL 连接信息
	Host        string `json:"host"`
	Port        string `json:"port"`
	Database    string `json:"database"`
	Username    string `json:"username"`
	Password    string `json:"password"`
	TablePrefix string `json:"tablePrefix"`
	// SQLiteDir 源数据库目录：SQLite 数据库文件所在目录（或文件路径）。
	SQLiteDir string `json:"sqliteDir"`
	// ImageDir 图片目录：原先的 storage/app/uploads 文件夹，可选；
	// 提供后导入会按原相对路径把图片拷贝进新站点存储。
	ImageDir string `json:"imageDir"`
	// AdminEmail/AdminPassword 为安装向导中创建的管理员账号，仅在执行
	// 导入时用于授权校验（测试连接不需要）。校验在 RunLegacyImport 内进行。
	AdminEmail    string `json:"adminEmail"`
	AdminPassword string `json:"adminPassword"`
}

func legacySource(in LegacySourceInput) legacy.Source {
	return legacy.Source{
		Type:        in.DatabaseType,
		Host:        in.Host,
		Port:        in.Port,
		Database:    in.Database,
		Username:    in.Username,
		Password:    in.Password,
		TablePrefix: in.TablePrefix,
		SQLiteDir:   in.SQLiteDir,
	}
}

// TestLegacySource 连接源数据库并做只读探测：确认是 Lsky Pro 的库，
// 统计可导入的数据量。整个过程只执行 SELECT。
func (s *Service) TestLegacySource(ctx context.Context, in LegacySourceInput) (*legacy.ProbeResult, error) {
	src := legacySource(in)
	srcDB, openReadOnly, err := legacy.OpenSource(ctx, src)
	if err != nil {
		return nil, err
	}
	defer closeSQL(srcDB)

	return legacy.Probe(ctx, srcDB, src, openReadOnly)
}

// RunLegacyImport 从 Lsky Pro 数据库只读导入数据到当前（新安装）数据库。
// 源库全程只读（SELECT-only + 会话只读模式），目标库使用当前运行时连接。
func (s *Service) RunLegacyImport(ctx context.Context, in LegacySourceInput) (legacy.ImportSummary, error) {
	status, err := s.Status(ctx)
	if err != nil {
		return legacy.ImportSummary{}, err
	}
	if !status.Installed {
		return legacy.ImportSummary{}, fmt.Errorf("请先完成系统安装，再执行数据导入")
	}
	if err := s.verifyInstallerAdmin(ctx, in.AdminEmail, in.AdminPassword); err != nil {
		return legacy.ImportSummary{}, err
	}

	src := legacySource(in)
	srcDB, openReadOnly, err := legacy.OpenSource(ctx, src)
	if err != nil {
		return legacy.ImportSummary{}, err
	}
	defer closeSQL(srcDB)

	orphanUserID, err := s.resolveOrphanUserID(ctx)
	if err != nil {
		return legacy.ImportSummary{}, err
	}

	// WithContext 让目标库事务内的写操作同样响应请求取消（客户端断开时
	// 中止导入并回滚）。
	return legacy.RunImport(ctx, srcDB, src, openReadOnly, s.db.WithContext(ctx), legacy.ImportOptions{
		StoragePath:   s.cfg.StoragePath,
		PublicBaseURL: s.cfg.PublicBaseURL,
		ImageDir:      in.ImageDir,
		OrphanUserID:  orphanUserID,
	})
}

// verifyInstallerAdmin 要求请求者持有安装时创建的管理员账号凭据，
// 防止未授权者在安装完成后调用导入接口写入数据。
func (s *Service) verifyInstallerAdmin(ctx context.Context, email, password string) error {
	if email == "" || password == "" {
		return fmt.Errorf("需要管理员账号授权")
	}
	var admin data.User
	err := s.db.WithContext(ctx).Where("email = ?", email).First(&admin).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("管理员账号验证失败")
		}
		return err
	}
	if !admin.IsAdmin && !admin.IsSuperAdmin {
		return fmt.Errorf("管理员账号验证失败")
	}
	if err := verifyPassword(admin.PasswordHash, password); err != nil {
		return fmt.Errorf("管理员账号验证失败")
	}
	return nil
}

// resolveOrphanUserID picks the account that owns images with no user in the
// source database (Lsky Pro guest uploads): prefer the first super admin.
func (s *Service) resolveOrphanUserID(ctx context.Context) (uint, error) {
	var admin data.User
	err := s.db.WithContext(ctx).Where("is_super_admin = ?", true).Order("id ASC").First(&admin).Error
	if err == nil {
		return admin.ID, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, err
	}
	err = s.db.WithContext(ctx).Where("is_adminer = ?", true).Order("id ASC").First(&admin).Error
	if err == nil {
		return admin.ID, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, err
	}
	err = s.db.WithContext(ctx).Order("id ASC").First(&admin).Error
	if err != nil {
		return 0, fmt.Errorf("找不到可用于归属游客图片的用户")
	}
	return admin.ID, nil
}
