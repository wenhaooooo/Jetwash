package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	// Logger 全局日志实例
	Logger *zap.Logger

	// SugarLogger 简化的日志实例
	SugarLogger *zap.SugaredLogger
)

// InitLogger 初始化日志
func InitLogger(mode string) error {
	var config zap.Config
	var encoder zapcore.Encoder
	var writer zapcore.WriteSyncer

	// 设置日志级别
	level := zap.NewAtomicLevel()
	if mode == "debug" {
		level.SetLevel(zapcore.DebugLevel)
	} else {
		level.SetLevel(zapcore.InfoLevel)
	}

	// 设置编码器
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		FunctionKey:    "function",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	// 根据模式设置不同的输出方式
	if mode == "debug" {
		// Debug 模式：输出到控制台
		encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
		writer = zapcore.AddSync(os.Stdout)
	} else {
		// Release 模式：输出到文件
		encoderConfig.EncodeLevel = zapcore.LowercaseLevelEncoder
		encoder = zapcore.NewJSONEncoder(encoderConfig)

		// 创建日志目录
		logDir := "logs"
		if err := os.MkdirAll(logDir, 0755); err != nil {
			return fmt.Errorf("failed to create log directory: %w", err)
		}

		// 按日期创建日志文件
		logFileName := fmt.Sprintf("%s.log", time.Now().Format("2006-01-02"))
		logFilePath := filepath.Join(logDir, logFileName)

		// 使用 lumberjack 进行日志轮转
		writer = zapcore.AddSync(&lumberjack.Logger{
			Filename:   logFilePath,
			MaxSize:    100,  // MB
			MaxBackups: 30,   // 保留最近 30 天的日志
			MaxAge:     30,   // 天
			Compress:   true, // 压缩旧日志
			LocalTime:  true,
		})
	}

	// 创建 Core
	core := zapcore.NewCore(
		encoder,
		writer,
		level,
	)

	// 创建 Logger
	config = zap.Config{
		Level:            level,
		Development:      mode == "debug",
		Encoding:         "json",
		EncoderConfig:    encoderConfig,
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
	}

	if mode == "debug" {
		config.Encoding = "console"
		config.OutputPaths = []string{"stdout"}
	} else {
		config.Encoding = "json"
		config.OutputPaths = []string{filepath.Join("logs", time.Now().Format("2006-01-02")+".log")}
	}

	Logger = zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))
	SugarLogger = Logger.Sugar()

	return nil
}

// WithRequestID 添加请求 ID 到日志上下文
func WithRequestID(requestID string) *zap.Logger {
	return Logger.With(zap.String("request_id", requestID))
}

// WithTenantID 添加租户 ID 到日志上下文
func WithTenantID(tenantID uuid.UUID) *zap.Logger {
	return Logger.With(zap.String("tenant_id", tenantID.String()))
}

// WithUserID 添加用户 ID 到日志上下文
func WithUserID(userID string) *zap.Logger {
	return Logger.With(zap.String("user_id", userID))
}

// WithFields 添加自定义字段到日志上下文
func WithFields(fields ...zap.Field) *zap.Logger {
	return Logger.With(fields...)
}

// Debug 记录 Debug 级别日志
func Debug(msg string, fields ...zap.Field) {
	Logger.Debug(msg, fields...)
}

// Info 记录 Info 级别日志
func Info(msg string, fields ...zap.Field) {
	Logger.Info(msg, fields...)
}

// Warn 记录 Warn 级别日志
func Warn(msg string, fields ...zap.Field) {
	Logger.Warn(msg, fields...)
}

// Error 记录 Error 级别日志
func Error(msg string, fields ...zap.Field) {
	Logger.Error(msg, fields...)
}

// Fatal 记录 Fatal 级别日志
func Fatal(msg string, fields ...zap.Field) {
	Logger.Fatal(msg, fields...)
}

// Sync 同步日志
func Sync() error {
	return Logger.Sync()
}

// GetLogger 获取 Logger 实例
func GetLogger() *zap.Logger {
	return Logger
}

// GetSugarLogger 获取 SugarLogger 实例
func GetSugarLogger() *zap.SugaredLogger {
	return SugarLogger
}

// RequestLogger 请求日志记录器
type RequestLogger struct {
	logger *zap.Logger
	fields []zap.Field
}

// NewRequestLogger 创建请求日志记录器
func NewRequestLogger(requestID, method, path string) *RequestLogger {
	fields := []zap.Field{
		zap.String("request_id", requestID),
		zap.String("method", method),
		zap.String("path", path),
		zap.Time("start_time", time.Now()),
	}
	return &RequestLogger{
		logger: Logger.With(fields...),
		fields: fields,
	}
}

// WithTenantID 添加租户 ID
func (rl *RequestLogger) WithTenantID(tenantID uuid.UUID) *RequestLogger {
	rl.fields = append(rl.fields, zap.String("tenant_id", tenantID.String()))
	rl.logger = Logger.With(rl.fields...)
	return rl
}

// WithUserID 添加用户 ID
func (rl *RequestLogger) WithUserID(userID string) *RequestLogger {
	rl.fields = append(rl.fields, zap.String("user_id", userID))
	rl.logger = Logger.With(rl.fields...)
	return rl
}

// WithField 添加自定义字段
func (rl *RequestLogger) WithField(field zap.Field) *RequestLogger {
	rl.fields = append(rl.fields, field)
	rl.logger = Logger.With(rl.fields...)
	return rl
}

// WithFields 添加多个自定义字段
func (rl *RequestLogger) WithFields(fields ...zap.Field) *RequestLogger {
	rl.fields = append(rl.fields, fields...)
	rl.logger = Logger.With(rl.fields...)
	return rl
}

// Debug 记录 Debug 级别日志
func (rl *RequestLogger) Debug(msg string, fields ...zap.Field) {
	rl.logger.Debug(msg, fields...)
}

// Info 记录 Info 级别日志
func (rl *RequestLogger) Info(msg string, fields ...zap.Field) {
	rl.logger.Info(msg, fields...)
}

// Warn 记录 Warn 级别日志
func (rl *RequestLogger) Warn(msg string, fields ...zap.Field) {
	rl.logger.Warn(msg, fields...)
}

// Error 记录 Error 级别日志
func (rl *RequestLogger) Error(msg string, fields ...zap.Field) {
	rl.logger.Error(msg, fields...)
}

// Complete 完成请求日志记录
func (rl *RequestLogger) Complete(statusCode int, latency time.Duration) {
	rl.logger.Info("request completed",
		zap.Int("status_code", statusCode),
		zap.Duration("latency", latency),
		zap.Time("end_time", time.Now()),
	)
}

// ErrorComplete 完成请求日志记录（带错误）
func (rl *RequestLogger) ErrorComplete(err error, statusCode int, latency time.Duration) {
	rl.logger.Error("request completed with error",
		zap.Error(err),
		zap.Int("status_code", statusCode),
		zap.Duration("latency", latency),
		zap.Time("end_time", time.Now()),
	)
}
