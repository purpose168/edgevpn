/*
Copyright © 2021-2022 Ettore Di Giacinto <mudler@mocaccino.org>
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package logger

import (
	"fmt"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/ipfs/go-log"
)

var _ log.StandardLogger = &Logger{}

// Logger 日志记录器结构体
type Logger struct {
	level log.LogLevel      // 日志级别
	zap   *zap.SugaredLogger // Zap日志记录器
}

// New 创建新的日志记录器
// 参数 lvl 为日志级别
func New(lvl log.LogLevel) *Logger {
	cfg := zap.Config{

		Encoding:         "json",                          // 编码格式
		OutputPaths:      []string{"stdout"},              // 输出路径
		ErrorOutputPaths: []string{"stderr"},              // 错误输出路径
		Level:            zap.NewAtomicLevelAt(zapcore.Level(lvl)), // 日志级别
		EncoderConfig: zapcore.EncoderConfig{
			MessageKey:   "message",                       // 消息键
			LevelKey:     "level",                         // 级别键
			EncodeLevel:  zapcore.CapitalLevelEncoder,     // 级别编码器
			TimeKey:      "time",                          // 时间键
			EncodeTime:   zapcore.ISO8601TimeEncoder,      // 时间编码器
			CallerKey:    "caller",                        // 调用者键
			EncodeCaller: zapcore.ShortCallerEncoder,      // 调用者编码器
		},
	}
	logger, err := cfg.Build(zap.AddCallerSkip(1))
	if err != nil {
		panic(err)
	}
	defer logger.Sync()

	sugar := logger.Sugar()

	return &Logger{level: lvl, zap: sugar}
}

// joinMsg 连接多个消息参数
func joinMsg(args ...interface{}) (message string) {
	for _, m := range args {
		message += " " + fmt.Sprintf("%v", m)
	}
	return
}

// Debug 输出调试级别日志
func (l Logger) Debug(args ...interface{}) {
	l.zap.Debug(joinMsg(args...))
}

// Debugf 输出格式化调试级别日志
func (l Logger) Debugf(f string, args ...interface{}) {
	l.zap.Debugf(f+"\n", args...)
}

// Error 输出错误级别日志
func (l Logger) Error(args ...interface{}) {
	l.zap.Error(joinMsg(args...))
}

// Errorf 输出格式化错误级别日志
func (l Logger) Errorf(f string, args ...interface{}) {
	l.zap.Errorf(f+"\n", args...)
}

// Fatal 输出致命级别日志并退出
func (l Logger) Fatal(args ...interface{}) {
	l.zap.Fatal(joinMsg(args...))
}

// Fatalf 输出格式化致命级别日志并退出
func (l Logger) Fatalf(f string, args ...interface{}) {
	l.zap.Fatalf(f+"\n", args...)
}

// Info 输出信息级别日志
func (l Logger) Info(args ...interface{}) {
	l.zap.Info(joinMsg(args...))
}

// Infof 输出格式化信息级别日志
func (l Logger) Infof(f string, args ...interface{}) {
	l.zap.Infof(f+"\n", args...)
}

// Panic 输出恐慌级别日志
func (l Logger) Panic(args ...interface{}) {
	l.Fatal(args...)
}

// Panicf 输出格式化恐慌级别日志
func (l Logger) Panicf(f string, args ...interface{}) {
	l.Fatalf(f, args...)
}

// Warn 输出警告级别日志
func (l Logger) Warn(args ...interface{}) {
	l.zap.Warn(joinMsg(args...))
}

// Warnf 输出格式化警告级别日志
func (l Logger) Warnf(f string, args ...interface{}) {
	l.zap.Warnf(f+"\n", args...)
}

// Warning 输出警告级别日志（Warn的别名）
func (l Logger) Warning(args ...interface{}) {
	l.Warn(args...)
}

// Warningf 输出格式化警告级别日志（Warnf的别名）
func (l Logger) Warningf(f string, args ...interface{}) {
	l.Warnf(f, args...)
}
