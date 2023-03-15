package utils

import (
	rotate "github.com/lestrrat-go/file-rotatelogs"
	log "github.com/sirupsen/logrus"
	"os"
	"time"
)

var Logger = log.New()

func InitLogger(path, name, level string) {
	if name == "" {
		Logger.SetOutput(os.Stdout)
	} else {
		path = path + "%Y%m%d_" + name
		/* 日志轮转相关函数
		`WithLinkName` 为最新的日志建立软连接
		`WithRotationTime` 设置日志分割的时间，隔多久分割一次
		WithMaxAge 和 WithRotationCount二者只能设置一个
		 `WithMaxAge` 设置文件清理前的最长保存时间
		 `WithRotationCount` 设置文件清理前最多保存的个数
		*/
		// 下面配置日志每隔 24小时轮转一个新文件，保留最近 7 天的日志文件，多余的自动清理掉。
		writer, _ := rotate.New(
			path,
			rotate.WithLinkName(path),
			rotate.WithMaxAge(time.Duration(7)*time.Duration(24)*time.Hour),
			rotate.WithRotationTime(time.Duration(24)*time.Hour),
		)
		Logger.SetOutput(writer)
	}
	//log.SetFormatter(&log.JSONFormatter{})
	Logger.SetFormatter(&log.TextFormatter{})
	var (
		lvl log.Level
		err error
	)
	if lvl, err = log.ParseLevel(level); err != nil {
		lvl = log.InfoLevel
	}
	Logger.SetLevel(lvl)
	Logger.Info("日志服务初始化成功...")
}
