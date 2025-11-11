package config

import (
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/the127/dockyard/internal/args"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/env/v2"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Kv       KvConfig
}

type ServerConfig struct {
	Port           int
	Host           string
	ExternalUrl    string
	ExternalDomain string
	AllowedOrigins []string
}

type DatabaseMode string

const (
	DatabaseModeInMemory DatabaseMode = "memory"
)

type DatabaseConfig struct {
	Mode DatabaseMode
}

type KvMode string

const (
	KvModeInMemory KvMode = "memory"
	KvModeRedis    KvMode = "redis"
)

type KvConfig struct {
	Mode  KvMode
	Redis struct {
		Host     string
		Port     int
		Username string
		Password string
		Database int
	}
}

type BlobStorageMode string

const (
	BlobStorageModeInMemory BlobStorageMode = "memory"
	BlobStorageModeS3       BlobStorageMode = "s3"
)

type BlobStorageConfig struct {
	Mode BlobStorageMode
	S3   struct {
		// TODO: add config
		Proxy bool
	}
}

var C Config

var k = koanf.New(".")

func Init() {
	if args.ConfigFilePath() != "" {
		_, err := os.Stat(args.ConfigFilePath())
		if err != nil {
			panic(fmt.Errorf("failed to stat config file: %w", err))
		}

		err = k.Load(file.Provider(args.ConfigFilePath()), yaml.Parser())
		if err != nil {
			panic(fmt.Errorf("failed to load config file: %w", err))
		}
	}

	err := k.Load(env.Provider(".", env.Opt{
		Prefix: "DOCKYARD_",
		TransformFunc: func(k, v string) (string, any) {
			// Transform the key.
			k = strings.ReplaceAll(strings.ToLower(strings.TrimPrefix(k, "DOCKYARD_")), "_", ".")

			if strings.Contains(v, " ") {
				return k, strings.Split(v, " ")
			}

			return k, v
		},
	}), nil)
	if err != nil {
		panic(fmt.Errorf("failed to load env provider: %w", err))
	}

	err = k.Unmarshal("", &C)
	if err != nil {
		panic(fmt.Errorf("failed to unmarshal config: %w", err))
	}

	setDefaultsOrPanic()
}

func setDefaultsOrPanic() {
	setServerDefaultsOrPanic()
	setDatabaseDefaultsOrPanic()
	setKvDefaultsOrPanic()
}

func setServerDefaultsOrPanic() {
	if C.Server.Host == "" {
		if args.IsProduction() {
			panic("Server.Host must be set in production.")
		}

		C.Server.Host = "localhost"
	}

	if C.Server.Port == 0 {
		C.Server.Port = 8080
	}

	if C.Server.ExternalUrl == "" {
		if args.IsProduction() {
			panic("Server.ExternalUrl must be set in production.")
		}

		C.Server.ExternalUrl = fmt.Sprintf("http://%s:%d", C.Server.Host, C.Server.Port)
	}

	if C.Server.ExternalDomain == "" {
		externalUrl, err := url.Parse(C.Server.ExternalUrl)
		if err != nil {
			panic(fmt.Errorf("failed to parse external url: %w", err))
		}

		C.Server.ExternalDomain = externalUrl.Hostname()
	}
}

func setDatabaseDefaultsOrPanic() {}

func setKvDefaultsOrPanic() {
	if C.Kv.Mode == "" {
		if args.IsProduction() {
			panic("Kv.Mode must be set in production.")
		}

		C.Kv.Mode = KvModeInMemory
	}

	if C.Kv.Mode == KvModeRedis {
		setKvRedisDefaultsOrPanic()
	}
}

func setKvRedisDefaultsOrPanic() {
	if C.Kv.Redis.Host == "" {
		if args.IsProduction() {
			panic("Kv.Redis.Host must be set in production.")
		}

		C.Kv.Redis.Host = "localhost"
	}

	if C.Kv.Redis.Port == 0 {
		C.Kv.Redis.Port = 6379
	}
}
