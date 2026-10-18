package glog

import (
	"os"
	"path/filepath"

	"github.com/natefinch/lumberjack"
	"github.com/samber/lo"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	Logger     *zap.Logger
	GormLogger *gormLogger

	RunLogger  *zap.Logger
	BinLogger  *zap.Logger
	LinkLogger *zap.Logger
	StatLogger *zap.Logger

	atomLevel zap.AtomicLevel
)

const (
	maxSize    = 50
	maxBackups = 10
)

func SetLevel(level zapcore.Level) {
	atomLevel.SetLevel(level)
}

func InitLoggers(loglevel, format, logdir, processName string) {
	atomLevel = lo.Must(zap.ParseAtomicLevel(loglevel))

	fileEncoder := zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig())
	consoleEncoder := zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig())

	var cores []zapcore.Core

	if logdir != "" && processName != "" {
		if err := os.MkdirAll(logdir, 0o755); err != nil {
			panic(err)
		}

		runWriter := &lumberjack.Logger{
			Filename:   filepath.Join(logdir, processName+".run.log"),
			MaxSize:    maxSize,
			MaxBackups: maxBackups,
			MaxAge:     0,
			Compress:   false,
		}

		binWriter := &lumberjack.Logger{
			Filename:   filepath.Join(logdir, processName+".bin.log"),
			MaxSize:    maxSize,
			MaxBackups: maxBackups,
			MaxAge:     0,
			Compress:   false,
		}

		linkWriter := &lumberjack.Logger{
			Filename:   filepath.Join(logdir, processName+".link.log"),
			MaxSize:    maxSize,
			MaxBackups: maxBackups,
			MaxAge:     0,
			Compress:   false,
		}

		statWriter := &lumberjack.Logger{
			Filename:   filepath.Join(logdir, processName+".stat.log"),
			MaxSize:    maxSize,
			MaxBackups: maxBackups,
			MaxAge:     0,
			Compress:   false,
		}

		RunLogger = zap.New(zapcore.NewTee(
			zapcore.NewCore(fileEncoder, zapcore.AddSync(runWriter), atomLevel),
			zapcore.NewCore(consoleEncoder, zapcore.AddSync(os.Stdout), atomLevel),
		), zap.AddCaller())
		BinLogger = zap.New(zapcore.NewCore(fileEncoder, zapcore.AddSync(binWriter), atomLevel), zap.AddCaller())
		LinkLogger = zap.New(zapcore.NewCore(fileEncoder, zapcore.AddSync(linkWriter), atomLevel), zap.AddCaller())
		StatLogger = zap.New(zapcore.NewCore(fileEncoder, zapcore.AddSync(statWriter), atomLevel), zap.AddCaller())

		cores = append(cores,
			zapcore.NewCore(fileEncoder, zapcore.AddSync(runWriter), atomLevel),
		)
	}

	cores = append(cores, zapcore.NewCore(consoleEncoder, zapcore.AddSync(os.Stdout), atomLevel))

	combinedCore := zapcore.NewTee(cores...)
	Logger = zap.New(combinedCore, zap.AddCaller())
	GormLogger = &gormLogger{Logger: *Logger}
}

func RunS() *zap.SugaredLogger {
	if RunLogger != nil {
		return RunLogger.Sugar()
	}
	return Logger.Sugar()
}

func BinS() *zap.SugaredLogger {
	if BinLogger != nil {
		return BinLogger.Sugar()
	}
	return Logger.Sugar()
}

func StatS() *zap.SugaredLogger {
	if StatLogger != nil {
		return StatLogger.Sugar()
	}
	return Logger.Sugar()
}

func LinkS() *zap.SugaredLogger {
	if LinkLogger != nil {
		return LinkLogger.Sugar()
	}
	return Logger.Sugar()
}

type gormLogger struct {
	Logger zap.Logger
}

func (p *gormLogger) Printf(format string, v ...any) {
	p.Logger.Sugar().Infof(format, v...)
}
