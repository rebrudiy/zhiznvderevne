package worker

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"zhiznvderevne/internal/config"
	"zhiznvderevne/internal/db"
	"zhiznvderevne/internal/llm"
	"zhiznvderevne/internal/metrics"
)

type Worker struct {
	cfg   *config.Config
	pool  *pgxpool.Pool
	llm   *llm.Client
	loc   *time.Location
	seed  *rand.Rand
	role  string
	interval time.Duration
}

func New(cfg *config.Config, pool *pgxpool.Pool, llmClient *llm.Client) (*Worker, error) {
	loc, err := time.LoadLocation(cfg.Content.Timezone)
	if err != nil {
		return nil, err
	}
	interval := time.Duration(cfg.Worker.PollIntervalSeconds) * time.Second
	if interval <= 0 {
		interval = 60 * time.Second
	}
	return &Worker{
		cfg:      cfg,
		pool:     pool,
		llm:      llmClient,
		loc:      loc,
		seed:     rand.New(rand.NewSource(time.Now().UnixNano())),
		role:     cfg.Worker.Role,
		interval: interval,
	}, nil
}

func (w *Worker) Run(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	w.tick(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.tick(ctx)
		}
	}
}

func (w *Worker) tick(ctx context.Context) {
	if w.role == "api" {
		return
	}

	locked, err := w.tryLock(ctx)
	if err != nil || !locked {
		return
	}
	defer w.unlock(ctx)

	now := time.Now().In(w.loc)
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, w.loc)
	already, err := db.EnsureDailySchedule(ctx, w.pool, dayStart)
	if err != nil {
		return
	}
	if !already {
		_ = w.createDailySchedule(ctx, dayStart)
	}

	post, err := db.PickDueScheduled(ctx, w.pool, now)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return
		}
		return
	}

	systemPrompt := "Ты пишешь личный дневник. Отвечай строго JSON."
	userPrompt := w.renderPrompt(now)
	draft, err := w.llm.GenerateDraft(ctx, systemPrompt, userPrompt)
	if err != nil {
		metrics.GenerationErrors.Inc()
		_ = db.FailPost(ctx, w.pool, post.ID)
		return
	}

	slug := sanitizeSlug(draft.Slug)
	uniqueSlug, err := w.ensureUniqueSlug(ctx, slug, post.ID)
	if err != nil {
		metrics.GenerationErrors.Inc()
		_ = db.FailPost(ctx, w.pool, post.ID)
		return
	}
	if err := db.PublishPost(ctx, w.pool, post.ID, draft.Title, uniqueSlug, draft.ContentMD, draft.Tags, w.cfg.OpenAI.Model, now); err != nil {
		metrics.GenerationErrors.Inc()
		_ = db.FailPost(ctx, w.pool, post.ID)
		return
	}
	metrics.GeneratedPosts.Inc()
}

func (w *Worker) tryLock(ctx context.Context) (bool, error) {
	var ok bool
	if err := w.pool.QueryRow(ctx, "SELECT pg_try_advisory_lock(424242)").Scan(&ok); err != nil {
		return false, err
	}
	return ok, nil
}

func (w *Worker) unlock(ctx context.Context) {
	_, _ = w.pool.Exec(ctx, "SELECT pg_advisory_unlock(424242)")
}

func (w *Worker) createDailySchedule(ctx context.Context, dayStart time.Time) error {
	count := w.randomBetween(w.cfg.Content.PostsPerDayMin, w.cfg.Content.PostsPerDayMax)
	if count == 0 {
		return nil
	}
	start, err := parseClock(w.cfg.Content.PublishWindowStart, dayStart)
	if err != nil {
		return err
	}
	end, err := parseClock(w.cfg.Content.PublishWindowEnd, dayStart)
	if err != nil {
		return err
	}
	if end.Before(start) {
		end = start.Add(12 * time.Hour)
	}

	for i := 0; i < count; i++ {
		when := randomTimeBetween(w.seed, start, end)
		id := uuid.NewString()
		p := db.PostRow{
			ID:             id,
			Title:          "(scheduled)",
			Slug:           "scheduled-" + strings.ReplaceAll(id, "-", ""),
			ContentMD:      "",
			Tags:           []string{},
			AuthorName:     w.cfg.Site.AuthorName,
			AuthorNickname: w.cfg.Site.AuthorNickname,
			Status:         "scheduled",
			ScheduledFor:   &when,
			Model:          w.cfg.OpenAI.Model,
			PromptVersion:  "v1",
		}
		_ = db.CreateScheduledPost(ctx, w.pool, p)
	}
	return nil
}

func (w *Worker) renderPrompt(now time.Time) string {
	prompt := w.cfg.Content.PromptTemplate
	replacer := strings.NewReplacer(
		"{author_name}", w.cfg.Site.AuthorName,
		"{author_nickname}", w.cfg.Site.AuthorNickname,
		"{date}", now.Format("02.01.2006"),
	)
	return replacer.Replace(prompt)
}

func (w *Worker) ensureUniqueSlug(ctx context.Context, slug, id string) (string, error) {
	candidate := slug
	for i := 0; i < 3; i++ {
		exists, err := db.SlugExists(ctx, w.pool, candidate, id)
		if err != nil {
			return "", err
		}
		if !exists {
			return candidate, nil
		}
		candidate = fmt.Sprintf("%s-%d", slug, i+2)
	}
	return fmt.Sprintf("%s-%s", slug, id[:6]), nil
}

func randomTimeBetween(rng *rand.Rand, start, end time.Time) time.Time {
	delta := end.Sub(start)
	if delta <= 0 {
		return start
	}
	return start.Add(time.Duration(rng.Int63n(int64(delta))))
}

func parseClock(s string, base time.Time) (time.Time, error) {
	parts := strings.Split(s, ":")
	if len(parts) != 2 {
		return time.Time{}, fmt.Errorf("invalid time %s", s)
	}
	hour := atoi(parts[0])
	min := atoi(parts[1])
	return time.Date(base.Year(), base.Month(), base.Day(), hour, min, 0, 0, base.Location()), nil
}

func atoi(s string) int {
	var n int
	for _, ch := range s {
		if ch < '0' || ch > '9' {
			return 0
		}
		n = n*10 + int(ch-'0')
	}
	return n
}

func sanitizeSlug(s string) string {
	if s == "" {
		return "post"
	}
	s = strings.ToLower(s)
	var b strings.Builder
	lastDash := false
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			b.WriteRune('-')
			lastDash = true
		}
	}
	res := strings.Trim(b.String(), "-")
	if res == "" {
		return "post"
	}
	return res
}

func (w *Worker) randomBetween(min, max int) int {
	if max <= min {
		return min
	}
	return min + w.seed.Intn(max-min+1)
}
