package config

import (
	"fmt"
	"log"

	"github.com/spf13/viper"
)

// Config 应用配置
type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	LLM      LLMConfig      `mapstructure:"llm"`
	JWT      JWTConfig      `mapstructure:"jwt"`
	Redis    RedisConfig    `mapstructure:"redis"`
	CORS     CORSConfig     `mapstructure:"cors"`
}

// CORSConfig CORS配置
type CORSConfig struct {
	AllowedOrigins []string `mapstructure:"allowed_origins"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Port         int    `mapstructure:"port"`
	Mode         string `mapstructure:"mode"` // debug, release
	ReadTimeout  int    `mapstructure:"read_timeout"`
	WriteTimeout int    `mapstructure:"write_timeout"`
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Host         string `mapstructure:"host"`
	Port         int    `mapstructure:"port"`
	User         string `mapstructure:"user"`
	Password     string `mapstructure:"password"`
	DBName       string `mapstructure:"dbname"`
	SSLMode      string `mapstructure:"sslmode"`
	MaxOpenConns int    `mapstructure:"max_open_conns"`
	MaxIdleConns int    `mapstructure:"max_idle_conns"`
}

type LLMConfig struct {
	Ollama   OllamaConfig `mapstructure:"ollama"`
	Online   OnlineConfig `mapstructure:"online"`
	Provider string       `mapstructure:"provider"` // online, ollama
}

type OllamaConfig struct {
	Host           string `mapstructure:"host"`
	Port           int    `mapstructure:"port"`
	EmbeddingModel string `mapstructure:"embedding_model"`
	ReasoningModel string `mapstructure:"reasoning_model"`
}

type OnlineConfig struct {
	APIKey         string `mapstructure:"api_key"`
	Model          string `mapstructure:"model"`
	BaseURL        string `mapstructure:"base_url"`
	EmbeddingModel string `mapstructure:"embedding_model"`
}

type JWTConfig struct {
	Secret     string `mapstructure:"secret"`
	ExpireHour int    `mapstructure:"expire_hour"`
}

// RedisConfig Redis配置
type RedisConfig struct {
	Addr     string `mapstructure:"addr"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

// LoadConfig 加载配置文件
func LoadConfig(configPath string) (*Config, error) {
	viper.SetConfigFile(configPath)
	viper.SetConfigType("yaml")

	// 设置默认值
	setDefaults()

	// 读取配置文件
	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &config, nil
}

// setDefaults 设置默认配置值
func setDefaults() {
	// 服务器默认配置
	viper.SetDefault("server.port", 13142)
	viper.SetDefault("server.mode", "debug")
	viper.SetDefault("server.read_timeout", 60)
	viper.SetDefault("server.write_timeout", 60)

	// 数据库默认配置
	viper.SetDefault("database.host", "localhost")
	viper.SetDefault("database.port", 15432)
	viper.SetDefault("database.user", "postgres")
	viper.SetDefault("database.password", "123456")
	viper.SetDefault("database.dbname", "jetwash")
	viper.SetDefault("database.sslmode", "disable")
	viper.SetDefault("database.max_open_conns", 100)
	viper.SetDefault("database.max_idle_conns", 10)

	// ollama 默认配置
	viper.SetDefault("llm.ollama.host", "http://localhost")
	viper.SetDefault("llm.ollama.port", 11434)
	viper.SetDefault("llm.ollama.embedding_model", "nomic-embed-text:v1.5")
	viper.SetDefault("llm.ollama.reasoning_model", "llama3.2:latest")

	// online 默认配置
	viper.SetDefault("llm.online.api_key", "")
	viper.SetDefault("llm.online.model", "glm-4.7")
	viper.SetDefault("llm.online.base_url", "https://open.bigmodel.cn/api/paas/v4")

	// LLM 提供者默认配置
	viper.SetDefault("llm.provider", "online")

	// Redis 默认配置
	viper.SetDefault("redis.addr", "localhost:6379")
	viper.SetDefault("redis.password", "")
	viper.SetDefault("redis.db", 0)

	// CORS 默认配置
	viper.SetDefault("cors.allowed_origins", []string{"*"})
}

// GetDSN 获取数据库连接字符串
func (c *DatabaseConfig) GetDSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.DBName, c.SSLMode,
	)
}

// MustLoadConfig 加载配置文件，失败时 panic
func MustLoadConfig(configPath string) *Config {
	config, err := LoadConfig(configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	return config
}
