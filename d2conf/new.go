package d2conf

import (
	"flag"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/iatneh/dodo-common/d2conf/cache"
	"github.com/iatneh/dodo-common/d2conf/env"
	"github.com/iatneh/dodo-common/d2conf/general"
	"github.com/iatneh/dodo-common/d2conf/http"
	"github.com/iatneh/dodo-common/d2conf/logger"
	"github.com/iatneh/dodo-common/d2conf/orm"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"io"
)

type Config struct {
	Env     string
	Http    *http.Config
	Orm     *orm.Config
	Orm2    *orm.Config
	Redis   *cache.Config
	Logger  *logger.Config
	General *general.Config // 这里存的 key 都会转小写字母
}

var (
	currentEnv string
	configPath string
)

func New() *Config {
	cfg := &Config{
		General: general.New(),
	}
	env.EnvStringVar(&currentEnv, "env", "dev", "application run env=[dev|sit|uat|prod]")
	env.EnvStringVar(&configPath, "config-path", ".", "set config search path")
	flag.Parse()
	v := viper.New()
	v.SetConfigName("app-" + currentEnv)
	v.AddConfigPath(configPath)
	v.AddConfigPath(".")
	v.AddConfigPath("./etc")
	v.AddConfigPath("./conf")
	v.AddConfigPath("./config")
	err := v.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("Fatal error config file: %s \n", err))
	}
	if err := v.Unmarshal(&cfg); err != nil {
		panic(fmt.Errorf("Fatal error config file: %s \n", err))
	}
	cfg.Env = currentEnv
	if len(v.GetStringMap("general")) > 0 {
		cfg.General.PutAll(v.GetStringMap("general"))
	}

	switch currentEnv {
	case "prod":
		gin.SetMode(gin.ReleaseMode)
		gin.DisableConsoleColor()
	case "dev":
		gin.SetMode(gin.DebugMode)
	case "sit", "uat":
		gin.DisableConsoleColor()
		gin.SetMode(gin.DebugMode)
	}

	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})

	{
		if cfg.Logger != nil && len(cfg.Logger.LogLevel) == 0 {
			cfg.Logger.LogLevel = "debug"
		}
		ll, err := logrus.ParseLevel(cfg.Logger.LogLevel)
		if err != nil {
			ll = logrus.DebugLevel
		}
		logrus.SetLevel(ll)
	}

	rl, err := logger.NewMultiWriter(cfg.Logger)
	if err != nil {
		panic(err.Error())
	}
	logrus.SetOutput(io.MultiWriter(rl...))

	logrus.Trace("logrus trace level active")
	logrus.Debug("logrus debug level active")
	logrus.Info("logrus info level active")
	logrus.Warnf("logrus warn level active")
	logrus.Error("logrus error level active")

	return cfg
}
