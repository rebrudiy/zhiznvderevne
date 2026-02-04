package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Site struct {
		Domain         string `yaml:"domain"`
		Title          string `yaml:"title"`
		AuthorName     string `yaml:"author_name"`
		AuthorNickname string `yaml:"author_nickname"`
		Description    string `yaml:"description"`
	} `yaml:"site"`
	HTTP struct {
		Addr          string `yaml:"addr"`
		PublicBaseURL string `yaml:"public_base_url"`
	} `yaml:"http"`
	Postgres struct {
		DSN string `yaml:"dsn"`
	} `yaml:"postgres"`
	Redis struct {
		Addr string `yaml:"addr"`
		DB   int    `yaml:"db"`
	} `yaml:"redis"`
	OpenAI struct {
		APIKeyEnv  string  `yaml:"api_key_env"`
		Model      string  `yaml:"model"`
		Temperature float64 `yaml:"temperature"`
		MaxTokens  int     `yaml:"max_tokens"`
	} `yaml:"openai"`
	Content struct {
		Timezone          string `yaml:"timezone"`
		PostsPerDayMin    int    `yaml:"posts_per_day_min"`
		PostsPerDayMax    int    `yaml:"posts_per_day_max"`
		PublishWindowStart string `yaml:"publish_window_start"`
		PublishWindowEnd   string `yaml:"publish_window_end"`
		PromptTemplate    string `yaml:"prompt_template"`
	} `yaml:"content"`
	Worker struct {
		Role                string `yaml:"role"`
		PollIntervalSeconds int    `yaml:"poll_interval_seconds"`
	} `yaml:"worker"`
	Observability struct {
		MetricsPath string `yaml:"metrics_path"`
	} `yaml:"observability"`
}

func Load() (*Config, error) {
	path := os.Getenv("CONFIG_PATH")
	if path == "" {
		path = "./config/config.yaml"
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	applyEnvOverrides(&cfg)
	if err := validate(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func applyEnvOverrides(cfg *Config) {
	if v := os.Getenv("HTTP_ADDR"); v != "" {
		cfg.HTTP.Addr = v
	}
	if v := os.Getenv("PUBLIC_BASE_URL"); v != "" {
		cfg.HTTP.PublicBaseURL = v
	}
	if v := os.Getenv("POSTGRES_DSN"); v != "" {
		cfg.Postgres.DSN = v
	}
	if v := os.Getenv("REDIS_ADDR"); v != "" {
		cfg.Redis.Addr = v
	}
	if v := os.Getenv("OPENAI_MODEL"); v != "" {
		cfg.OpenAI.Model = v
	}
	if v := os.Getenv("OPENAI_MAX_TOKENS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.OpenAI.MaxTokens = n
		}
	}
}

func validate(cfg *Config) error {
	if cfg.HTTP.Addr == "" {
		return errors.New("http.addr is required")
	}
	if cfg.Postgres.DSN == "" {
		return errors.New("postgres.dsn is required")
	}
	if cfg.OpenAI.APIKeyEnv == "" {
		return errors.New("openai.api_key_env is required")
	}
	if cfg.OpenAI.Model == "" {
		return errors.New("openai.model is required")
	}
	if cfg.Content.PostsPerDayMin < 0 || cfg.Content.PostsPerDayMax < cfg.Content.PostsPerDayMin {
		return errors.New("content.posts_per_day_min/max invalid")
	}
	if cfg.Content.PromptTemplate == "" {
		return errors.New("content.prompt_template is required")
	}
	return nil
}
