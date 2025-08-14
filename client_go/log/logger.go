package log

import (
	"fmt"
	"log"
	"os"
	"reflect"
	"runtime"
	"strings"
	"time"
)

// LogLevel 日志级别
type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
)

// String 返回日志级别的字符串表示
func (l LogLevel) String() string {
	switch l {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARN:
		return "WARN"
	case ERROR:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// HotKeyLogger 热key日志接口，对应Java的HotKeyLogger
type HotKeyLogger interface {
	Debug(className interface{}, info string)
	Info(className interface{}, info string)
	Error(className interface{}, info string)
	Warn(className interface{}, info string)
	SetLevel(level LogLevel)
	IsEnabled(level LogLevel) bool
}

// DefaultLogger 默认日志实现，对应Java的DefaultLogger
type DefaultLogger struct {
	level  LogLevel
	logger *log.Logger
}

// NewDefaultLogger 创建默认日志器
func NewDefaultLogger() *DefaultLogger {
	return &DefaultLogger{
		level:  INFO,
		logger: log.New(os.Stdout, "", log.LstdFlags),
	}
}

// NewDefaultLoggerWithLevel 创建指定级别的默认日志器
func NewDefaultLoggerWithLevel(level LogLevel) *DefaultLogger {
	return &DefaultLogger{
		level:  level,
		logger: log.New(os.Stdout, "", log.LstdFlags),
	}
}

// SetLevel 设置日志级别
func (dl *DefaultLogger) SetLevel(level LogLevel) {
	dl.level = level
}

// IsEnabled 检查指定级别是否启用
func (dl *DefaultLogger) IsEnabled(level LogLevel) bool {
	return level >= dl.level
}

// Debug 输出调试日志
func (dl *DefaultLogger) Debug(className interface{}, info string) {
	if dl.IsEnabled(DEBUG) {
		dl.logWithClass(DEBUG, className, info)
	}
}

// Info 输出信息日志
func (dl *DefaultLogger) Info(className interface{}, info string) {
	if dl.IsEnabled(INFO) {
		dl.logWithClass(INFO, className, info)
	}
}

// Warn 输出警告日志
func (dl *DefaultLogger) Warn(className interface{}, info string) {
	if dl.IsEnabled(WARN) {
		dl.logWithClass(WARN, className, info)
	}
}

// Error 输出错误日志
func (dl *DefaultLogger) Error(className interface{}, info string) {
	if dl.IsEnabled(ERROR) {
		dl.logWithClass(ERROR, className, info)
	}
}

// logWithClass 带类名的日志输出
func (dl *DefaultLogger) logWithClass(level LogLevel, className interface{}, info string) {
	classNameStr := dl.getClassName(className)
	timestamp := time.Now().Format("2006-01-02 15:04:05.000")
	
	// 获取调用者信息
	_, file, line, ok := runtime.Caller(3)
	var caller string
	if ok {
		caller = fmt.Sprintf("%s:%d", getShortFileName(file), line)
	} else {
		caller = "unknown"
	}
	
	message := fmt.Sprintf("[%s] [%s] [%s] [%s] %s", 
		timestamp, level.String(), classNameStr, caller, info)
	
	dl.logger.Println(message)
}

// getClassName 获取类名
func (dl *DefaultLogger) getClassName(className interface{}) string {
	if className == nil {
		return "unknown"
	}
	
	switch v := className.(type) {
	case string:
		return v
	case reflect.Type:
		return getShortTypeName(v.String())
	default:
		t := reflect.TypeOf(v)
		if t.Kind() == reflect.Ptr {
			t = t.Elem()
		}
		return getShortTypeName(t.String())
	}
}

// getShortTypeName 获取短类型名
func getShortTypeName(fullName string) string {
	parts := strings.Split(fullName, ".")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return fullName
}

// getShortFileName 获取短文件名
func getShortFileName(fullPath string) string {
	parts := strings.Split(fullPath, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return fullPath
}

// 全局日志器实例
var globalLogger HotKeyLogger = NewDefaultLogger()

// JdLogger 全局日志工具，对应Java的JdLogger
type JdLogger struct{}

// SetGlobalLogger 设置全局日志器
func SetGlobalLogger(logger HotKeyLogger) {
	globalLogger = logger
}

// GetGlobalLogger 获取全局日志器
func GetGlobalLogger() HotKeyLogger {
	return globalLogger
}

// SetLevel 设置全局日志级别
func SetLevel(level LogLevel) {
	globalLogger.SetLevel(level)
}

// Debug 全局调试日志
func Debug(className interface{}, info string) {
	globalLogger.Debug(className, info)
}

// Info 全局信息日志
func Info(className interface{}, info string) {
	globalLogger.Info(className, info)
}

// Warn 全局警告日志
func Warn(className interface{}, info string) {
	globalLogger.Warn(className, info)
}

// Error 全局错误日志
func Error(className interface{}, info string) {
	globalLogger.Error(className, info)
}

// Debugf 格式化调试日志
func Debugf(className interface{}, format string, args ...interface{}) {
	globalLogger.Debug(className, fmt.Sprintf(format, args...))
}

// Infof 格式化信息日志
func Infof(className interface{}, format string, args ...interface{}) {
	globalLogger.Info(className, fmt.Sprintf(format, args...))
}

// Warnf 格式化警告日志
func Warnf(className interface{}, format string, args ...interface{}) {
	globalLogger.Warn(className, fmt.Sprintf(format, args...))
}

// Errorf 格式化错误日志
func Errorf(className interface{}, format string, args ...interface{}) {
	globalLogger.Error(className, fmt.Sprintf(format, args...))
}

// IsDebugEnabled 检查是否启用调试日志
func IsDebugEnabled() bool {
	return globalLogger.IsEnabled(DEBUG)
}

// IsInfoEnabled 检查是否启用信息日志
func IsInfoEnabled() bool {
	return globalLogger.IsEnabled(INFO)
}

// IsWarnEnabled 检查是否启用警告日志
func IsWarnEnabled() bool {
	return globalLogger.IsEnabled(WARN)
}

// IsErrorEnabled 检查是否启用错误日志
func IsErrorEnabled() bool {
	return globalLogger.IsEnabled(ERROR)
}

// FileLogger 文件日志器
type FileLogger struct {
	*DefaultLogger
	file *os.File
}

// NewFileLogger 创建文件日志器
func NewFileLogger(filename string) (*FileLogger, error) {
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return nil, err
	}
	
	return &FileLogger{
		DefaultLogger: &DefaultLogger{
			level:  INFO,
			logger: log.New(file, "", log.LstdFlags),
		},
		file: file,
	}, nil
}

// Close 关闭文件日志器
func (fl *FileLogger) Close() error {
	if fl.file != nil {
		return fl.file.Close()
	}
	return nil
}

// MultiLogger 多重日志器，可以同时输出到多个目标
type MultiLogger struct {
	loggers []HotKeyLogger
	level   LogLevel
}

// NewMultiLogger 创建多重日志器
func NewMultiLogger(loggers ...HotKeyLogger) *MultiLogger {
	return &MultiLogger{
		loggers: loggers,
		level:   INFO,
	}
}

// SetLevel 设置日志级别
func (ml *MultiLogger) SetLevel(level LogLevel) {
	ml.level = level
	for _, logger := range ml.loggers {
		logger.SetLevel(level)
	}
}

// IsEnabled 检查指定级别是否启用
func (ml *MultiLogger) IsEnabled(level LogLevel) bool {
	return level >= ml.level
}

// Debug 输出调试日志
func (ml *MultiLogger) Debug(className interface{}, info string) {
	if ml.IsEnabled(DEBUG) {
		for _, logger := range ml.loggers {
			logger.Debug(className, info)
		}
	}
}

// Info 输出信息日志
func (ml *MultiLogger) Info(className interface{}, info string) {
	if ml.IsEnabled(INFO) {
		for _, logger := range ml.loggers {
			logger.Info(className, info)
		}
	}
}

// Warn 输出警告日志
func (ml *MultiLogger) Warn(className interface{}, info string) {
	if ml.IsEnabled(WARN) {
		for _, logger := range ml.loggers {
			logger.Warn(className, info)
		}
	}
}

// Error 输出错误日志
func (ml *MultiLogger) Error(className interface{}, info string) {
	if ml.IsEnabled(ERROR) {
		for _, logger := range ml.loggers {
			logger.Error(className, info)
		}
	}
}
