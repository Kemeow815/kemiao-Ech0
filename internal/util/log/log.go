package util

import (
	"os"
	"path/filepath"

	model "github.com/lin-snow/ech0/internal/model/common"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// Logger 全局日志记录器
var Logger *zap.Logger

// LogConfig 日志配置
type LogConfig struct {
	// 日志级别: debug, info, warn, error, panic
	Level string `yaml:"level" json:"level"`
	// 日志格式: json, console
	Format string `yaml:"format" json:"format"`
	// 是否输出到控制台
	Console bool `yaml:"console" json:"console"`
	// 文件输出配置
	File FileConfig `yaml:"file" json:"file"`
}

// FileConfig 文件输出配置
type FileConfig struct {
	// 是否启用文件输出
	Enable bool `yaml:"enable" json:"enable"`
	// 日志文件路径
	Filename string `yaml:"filename" json:"filename"`
	// 单个文件最大大小（MB）
	MaxSize int `yaml:"maxsize" json:"maxsize"`
	// 保留的旧文件数量
	MaxBackups int `yaml:"maxbackups" json:"maxbackups"`
	// 保留天数
	MaxAge int `yaml:"maxage" json:"maxage"`
	// 是否压缩旧文件
	Compress bool `yaml:"compress" json:"compress"`
}

// DefaultLogConfig 默认日志配置
func DefaultLogConfig() LogConfig {
	return LogConfig{
		Level:   "error",
		Format:  "json",
		Console: false,
		File: FileConfig{
			Enable:     true,
			Filename:   "data/app.log",
			MaxSize:    100,
			MaxBackups: 5,
			MaxAge:     30,
			Compress:   true,
		},
	}
}

// InitLogger 使用默认配置初始化日志记录器
func InitLogger() {
	config := DefaultLogConfig()
	InitLoggerWithConfig(config)
}

// InitLoggerWithConfig 使用自定义配置初始化日志记录器
func InitLoggerWithConfig(config LogConfig) {
	// 解析日志级别
	level, err := zapcore.ParseLevel(config.Level)
	if err != nil {
		level = zapcore.InfoLevel
	}

	// 创建编码器配置
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		FunctionKey:    zapcore.OmitKey,
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	var cores []zapcore.Core

	// 控制台输出
	if config.Console {
		consoleConfig := encoderConfig
		consoleConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder

		var consoleEncoder zapcore.Encoder
		if config.Format == "json" {
			consoleEncoder = zapcore.NewJSONEncoder(consoleConfig)
		} else {
			consoleEncoder = zapcore.NewConsoleEncoder(consoleConfig)
		}

		consoleCore := zapcore.NewCore(
			consoleEncoder,
			zapcore.AddSync(os.Stdout),
			level,
		)
		cores = append(cores, consoleCore)
	}

	// 文件输出
	if config.File.Enable {
		// 确保日志目录存在
		logDir := filepath.Dir(config.File.Filename)
		if err := os.MkdirAll(logDir, 0755); err != nil {
			panic(model.INIT_LOGGER_PANIC + ": 创建日志目录失败: " + err.Error())
		}

		// 配置日志轮转
		writer := &lumberjack.Logger{
			Filename:   config.File.Filename,
			MaxSize:    config.File.MaxSize,
			MaxBackups: config.File.MaxBackups,
			MaxAge:     config.File.MaxAge,
			Compress:   config.File.Compress,
			LocalTime:  true,
		}

		var fileEncoder zapcore.Encoder
		if config.Format == "json" {
			fileEncoder = zapcore.NewJSONEncoder(encoderConfig)
		} else {
			fileEncoder = zapcore.NewConsoleEncoder(encoderConfig)
		}

		fileCore := zapcore.NewCore(
			fileEncoder,
			zapcore.AddSync(writer),
			level,
		)
		cores = append(cores, fileCore)
	}

	// 如果没有配置任何输出，使用默认控制台输出
	if len(cores) == 0 {
		cores = append(cores, zapcore.NewCore(
			zapcore.NewConsoleEncoder(encoderConfig),
			zapcore.AddSync(os.Stdout),
			level,
		))
	}

	// 合并所有核心
	core := zapcore.NewTee(cores...)

	// 创建 logger
	Logger = zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))
}

// GetLogger 获取日志记录器实例
func GetLogger() *zap.Logger {
	if Logger == nil {
		InitLogger()
	}
	return Logger
}

// 便捷的日志方法
func Debug(msg string, fields ...zap.Field) {
	GetLogger().Debug(msg, fields...)
}

func Info(msg string, fields ...zap.Field) {
	GetLogger().Info(msg, fields...)
}

func Warn(msg string, fields ...zap.Field) {
	GetLogger().Warn(msg, fields...)
}

func Error(msg string, fields ...zap.Field) {
	GetLogger().Error(msg, fields...)
}

func Panic(msg string, fields ...zap.Field) {
	GetLogger().Panic(msg, fields...)
}

func Fatal(msg string, fields ...zap.Field) {
	GetLogger().Fatal(msg, fields...)
}
