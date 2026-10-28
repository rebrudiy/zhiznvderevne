package config

import "testing"

func valid() *Config {
	var c Config
	c.HTTP.Addr = ":8080"
	c.Postgres.DSN = "postgres://localhost/test"
	c.OpenAI.APIKeyEnv = "OPENAI_API_KEY"
	c.OpenAI.Model = "gpt-4o-mini"
	c.Content.PostsPerDayMin, c.Content.PostsPerDayMax = 1, 3
	c.Content.PromptTemplate = "write about {date}"
	return &c
}

func TestValidate(t *testing.T) {
	if err := validate(valid()); err != nil {
		t.Fatalf("valid config rejected: %v", err)
	}
	broken := map[string]func(*Config){
		"no addr":     func(c *Config) { c.HTTP.Addr = "" },
		"no dsn":      func(c *Config) { c.Postgres.DSN = "" },
		"no model":    func(c *Config) { c.OpenAI.Model = "" },
		"min > max":   func(c *Config) { c.Content.PostsPerDayMin, c.Content.PostsPerDayMax = 5, 2 },
		"negative":    func(c *Config) { c.Content.PostsPerDayMin = -1 },
		"no template": func(c *Config) { c.Content.PromptTemplate = "" },
	}
	for name, mutate := range broken {
		c := valid()
		mutate(c)
		if err := validate(c); err == nil {
			t.Errorf("%s: expected error", name)
		}
	}
}

func TestEnvOverrides(t *testing.T) {
	t.Setenv("POSTGRES_DSN", "postgres://env/db")
	t.Setenv("OPENAI_MAX_TOKENS", "512")
	t.Setenv("HTTP_ADDR", "")
	c := valid()
	c.OpenAI.MaxTokens = 100
	applyEnvOverrides(c)
	if c.Postgres.DSN != "postgres://env/db" || c.OpenAI.MaxTokens != 512 || c.HTTP.Addr != ":8080" {
		t.Fatalf("unexpected overrides: %+v", c)
	}
	t.Setenv("OPENAI_MAX_TOKENS", "lots")
	applyEnvOverrides(c)
	if c.OpenAI.MaxTokens != 512 {
		t.Fatalf("non-numeric value must be ignored, got %d", c.OpenAI.MaxTokens)
	}
}
