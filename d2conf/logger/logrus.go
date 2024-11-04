package logger

import (
	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Config 日志配置
type Config struct {
	// Output 配置日志输出,有以下格式: file:///,stdout,stderr 等,文件路径设置例:
	// /opt/logs/log.%Y%m%d
	// 具体见  strftime(3) 格式
	Output       string
	LogLevel     string // 配置日志输出级别: trace,debug,info,warn,error
	MaxAge       int    // 日志保留天数
	RotationTime int    // 日志分割时间,单位秒,默认86400秒
}

var (
	outputArray []string
)

// NewMultiWriter 返回一个以上writer,可以在 logrus 中使用
func NewMultiWriter(config *Config) ([]io.Writer, error) {
	outputArray = strings.Split(config.Output, ",")
	var writers []io.Writer
	for i := range outputArray {
		output := outputArray[i]
		switch output {
		case "stdout":
			writers = append(writers, os.Stdout)
		case "stderr":
			writers = append(writers, os.Stderr)
		default:
			// TODO 除了标准输出外，先支持文件输出
			if !strings.HasPrefix(output, `file://`) {
				continue
			}
			logPath := strings.ReplaceAll(output, `file://`, "")
			linkName := filepath.Join(filepath.Dir(logPath), "current")

			// 默认保留1年日志
			if config.MaxAge == 0 {
				config.MaxAge = 365
			}

			if config.RotationTime <= 0 {
				config.RotationTime = 86400
			}

			rl, err := rotatelogs.New(logPath,
				rotatelogs.WithMaxAge(24*time.Hour*time.Duration(config.MaxAge)),
				rotatelogs.WithRotationTime(time.Second*time.Duration(config.RotationTime)),
				rotatelogs.WithLinkName(linkName),
			)
			if err != nil {
				return nil, err
			}
			writers = append(writers, rl)
		}
	}
	return writers, nil
}
