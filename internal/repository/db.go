package repository

import (
	"fmt"
	"log"
	"time"

	"jetwash/internal/config"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// DB 数据库实例
var DB *gorm.DB

// InitDB 初始化数据库连接
func InitDB(cfg *config.DatabaseConfig) (*gorm.DB, error) {
	// 配置 GORM 日志
	logLevel := logger.Silent
	if cfg.Host == "localhost" {
		logLevel = logger.Info
	}

	// 创建数据库连接
	db, err := gorm.Open(postgres.Open(cfg.GetDSN()), &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// 获取底层 SQL DB 实例
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database instance: %w", err)
	}

	// 设置连接池参数
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(time.Hour)

	// 测试连接
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// 创建 pgvector 扩展（如果不存在）
	/*	if err := createVectorExtension(db); err != nil {
		return nil, fmt.Errorf("failed to create vector extension: %w", err)
	}*/

	// 自动迁移数据库表结构
	/* 	if err := autoMigrate(db); err != nil {
		return nil, fmt.Errorf("failed to auto migrate: %w", err)
	} */

	DB = db
	log.Println("Database initialized successfully")
	return db, nil
}

// createVectorExtension 创建 pgvector 扩展
/*func createVectorExtension(db *gorm.DB) error {
	result := db.Exec("CREATE EXTENSION IF NOT EXISTS vector")
	if result.Error != nil {
		return result.Error
	}
	log.Println("pgvector extension created/verified")
	return nil
}*/

// autoMigrate 自动迁移数据库表结构
/* func autoMigrate(db *gorm.DB) error {
	// 在迁移前删除引用表的视图，以避免 "cannot alter type of a column used by a view or rule" 错误
	dropViewsSQL := `
		DROP VIEW IF EXISTS active_sensitive_words CASCADE;
		DROP VIEW IF EXISTS active_tenants CASCADE;
	`
	if err := db.Exec(dropViewsSQL).Error; err != nil {
		log.Printf("Warning: failed to drop views: %v", err)
		// 不返回错误，继续执行迁移
	}

	// 迁移所有模型
	err := db.AutoMigrate(
		&models.Tenant{},
		&models.SensitiveWord{},
	)
	if err != nil {
		return err
	}

	// 迁移后重新创建视图
	recreateViewsSQL := `
		-- 视图：活跃的敏感词
		CREATE OR REPLACE VIEW active_sensitive_words AS
		SELECT
			id,
			tenant_id,
			word_text,
			category,
			risk_level,
			embedding,
			status,
			created_at,
			updated_at
		FROM sensitive_words
		WHERE deleted_at IS NULL
		  AND status = 'active';

		-- 视图：活跃的租户
		CREATE OR REPLACE VIEW active_tenants AS
		SELECT
			id,
			api_key,
			name,
			status,
			created_at,
			updated_at
		FROM tenants
		WHERE deleted_at IS NULL
		  AND status = 'active';
	`
	if err := db.Exec(recreateViewsSQL).Error; err != nil {
		log.Printf("Warning: failed to recreate views: %v", err)
		// 不返回错误，因为表结构已经迁移成功
	}

	log.Println("Database tables migrated successfully")
	return nil
} */

// GetDB 获取数据库实例
func GetDB() *gorm.DB {
	return DB
}

// CloseDB 关闭数据库连接
func CloseDB() error {
	if DB == nil {
		return nil
	}

	sqlDB, err := DB.DB()
	if err != nil {
		return err
	}

	return sqlDB.Close()
}
