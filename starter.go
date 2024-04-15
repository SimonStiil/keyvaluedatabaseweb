package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/viper"
)

var (
	configFileName string
	requests       = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "http_endpoint_equests_count",
		Help: "The amount of requests to an endpoint",
	}, []string{"endpoint", "method", "type"},
	)
)

type ConfigType struct {
	Logging    ConfigLogging    `mapstructure:"logging"`
	Port       string           `mapstructure:"port"`
	Backend    ConfigBackend    `mapstructure:"backend"`
	Prometheus ConfigPrometheus `mapstructure:"prometheus"`
}
type ConfigLogging struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
	File   string `mapstructure:"file"`
}

type ConfigBackend struct {
	Host     string `mapstructure:"host"`
	Port     string `mapstructure:"port"`
	Protocol string `mapstructure:"protocol"`
	Cert     string `mapstructure:"cert"`
	CertDir  string `mapstructure:"certificateDirectory"`
	insecure bool   `mapstructure:"insecure"`
	Key      string `mapstructure:"key"`
	Username string `mapstructure:"username"`
}

type ConfigPrometheus struct {
	Enabled  bool   `mapstructure:"enabled"`
	Endpoint string `mapstructure:"endpoint"`
}

const (
	BaseENVname = "KVDBW"
)

func ConfigRead(configFileName string, configOutput *ConfigType) {
	configReader := viper.New()
	configReader.SetConfigName(configFileName)
	configReader.SetConfigType("yaml")
	configReader.AddConfigPath("/app/")
	configReader.AddConfigPath(".")
	configReader.SetEnvPrefix(BaseENVname)
	configReader.SetDefault("logging.level", "Debug")
	configReader.SetDefault("logging.format", "text")
	configReader.SetDefault("port", 8080)
	configReader.SetDefault("backend.host", "kvdb")
	// https://en.wikipedia.org/wiki/List_of_TCP_and_UDP_port_numbers
	configReader.SetDefault("backend.port", 443)
	configReader.SetDefault("backend.protocol", "https")
	configReader.SetDefault("backend.cert", "client.crt")
	configReader.SetDefault("backend.key", "client.key")
	configReader.SetDefault("backend.certificateDirectory", "/certificates/")
	configReader.SetDefault("backend.username", "system")
	configReader.SetDefault("backend.insecure", false)
	configReader.SetDefault("prometheus.enabled", true)
	configReader.SetDefault("prometheus.endpoint", "/system/metrics")
	err := configReader.ReadInConfig() // Find and read the config file
	if err != nil {                    // Handle errors reading the config file
		panic(fmt.Errorf("fatal error config file: %w", err))
	}
	configReader.AutomaticEnv()
	configReader.Unmarshal(configOutput)
}

type Health struct {
	Status string `json:"status"`
}

func main() {
	flag.StringVar(&configFileName, "config", "config", "Use a different config file name")
	flag.Parse()
	App := new(Application)
	ConfigRead(configFileName, &App.Config)
	App.setupLogging()

	httpClient := InitClient(App.Config.Backend)
	App.KVDBClient = httpClient
	if App.Config.Prometheus.Enabled {
		App.Logger.Info(fmt.Sprintf("Metrics enabled at %v", App.Config.Prometheus.Endpoint))
		http.Handle(App.Config.Prometheus.Endpoint, promhttp.Handler())
	}
	http.HandleFunc("/", http.HandlerFunc(App.RootController))
	http.HandleFunc("/system/health", http.HandlerFunc(App.HealthActuator))
	App.Logger.Info(fmt.Sprintf("Serving on port %v", App.Config.Port))
	log.Fatal(http.ListenAndServe(":"+App.Config.Port, nil))
}
