package db

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PostRow struct {
	ID             string
	Title          string
	Slug           string
	ContentMD      string
	Tags           []string
	AuthorName     string
	AuthorNickname string
	Status         string
	ScheduledFor   *time.Time
	PublishedAt    *time.Time
	Model          string
	PromptVersion  string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func CreateScheduledPost(ctx context.Context, pool *pgxpool.Pool, p PostRow) error {
	_, err := pool.Exec(ctx, `
		INSERT INTO posts (id, title, slug, content_md, tags, author_name, author_nickname, status, scheduled_for, model, prompt_version)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
	`, p.ID, p.Title, p.Slug, p.ContentMD, p.Tags, p.AuthorName, p.AuthorNickname, p.Status, p.ScheduledFor, p.Model, p.PromptVersion)
	return err
}

func PickDueScheduled(ctx context.Context, pool *pgxpool.Pool, now time.Time) (*PostRow, error) {
	row := pool.QueryRow(ctx, `
		WITH picked AS (
			SELECT id FROM posts
			WHERE status = 'scheduled' AND scheduled_for <= $1
			ORDER BY scheduled_for ASC
			FOR UPDATE SKIP LOCKED
			LIMIT 1
		)
		UPDATE posts
		SET status = 'generating', updated_at = now()
		WHERE id IN (SELECT id FROM picked)
		RETURNING id, title, slug, content_md, tags, author_name, author_nickname, status, scheduled_for, published_at, model, prompt_version, created_at, updated_at
	`, now)

	var p PostRow
	if err := row.Scan(&p.ID, &p.Title, &p.Slug, &p.ContentMD, &p.Tags, &p.AuthorName, &p.AuthorNickname, &p.Status, &p.ScheduledFor, &p.PublishedAt, &p.Model, &p.PromptVersion, &p.CreatedAt, &p.UpdatedAt); err != nil {
		return nil, err
	}
	return &p, nil
}

func PublishPost(ctx context.Context, pool *pgxpool.Pool, id, title, slug, content string, tags []string, model string, publishedAt time.Time) error {
	_, err := pool.Exec(ctx, `
		UPDATE posts
		SET title=$2, slug=$3, content_md=$4, tags=$5, model=$6, status='published', published_at=$7, updated_at=now()
		WHERE id=$1
	`, id, title, slug, content, tags, model, publishedAt)
	return err
}

func FailPost(ctx context.Context, pool *pgxpool.Pool, id string) error {
	_, err := pool.Exec(ctx, `
		UPDATE posts SET status='failed', updated_at=now() WHERE id=$1
	`, id)
	return err
}

func EnsureDailySchedule(ctx context.Context, pool *pgxpool.Pool, day time.Time) (bool, error) {
	var exists bool
	if err := pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM posts
			WHERE scheduled_for >= $1 AND scheduled_for < $2
			AND status IN ('scheduled', 'generating', 'published', 'failed')
		)
	`, day, day.Add(24*time.Hour)).Scan(&exists); err != nil {
		return false, err
	}
	return exists, nil
}

func ListPublished(ctx context.Context, pool *pgxpool.Pool, limit int) ([]PostRow, error) {
	rows, err := pool.Query(ctx, `
		SELECT id, title, slug, content_md, tags, author_name, author_nickname, status, scheduled_for, published_at, model, prompt_version, created_at, updated_at
		FROM posts WHERE status='published'
		ORDER BY published_at DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var res []PostRow
	for rows.Next() {
		var p PostRow
		if err := rows.Scan(&p.ID, &p.Title, &p.Slug, &p.ContentMD, &p.Tags, &p.AuthorName, &p.AuthorNickname, &p.Status, &p.ScheduledFor, &p.PublishedAt, &p.Model, &p.PromptVersion, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		res = append(res, p)
	}
	return res, rows.Err()
}

func GetPostBySlug(ctx context.Context, pool *pgxpool.Pool, slug string) (*PostRow, error) {
	row := pool.QueryRow(ctx, `
		SELECT id, title, slug, content_md, tags, author_name, author_nickname, status, scheduled_for, published_at, model, prompt_version, created_at, updated_at
		FROM posts WHERE slug=$1 AND status='published'
		LIMIT 1
	`, slug)
	var p PostRow
	if err := row.Scan(&p.ID, &p.Title, &p.Slug, &p.ContentMD, &p.Tags, &p.AuthorName, &p.AuthorNickname, &p.Status, &p.ScheduledFor, &p.PublishedAt, &p.Model, &p.PromptVersion, &p.CreatedAt, &p.UpdatedAt); err != nil {
		return nil, err
	}
	return &p, nil
}

func SlugExists(ctx context.Context, pool *pgxpool.Pool, slug, excludeID string) (bool, error) {
	var exists bool
	if err := pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM posts WHERE slug=$1 AND id<>$2
		)
	`, slug, excludeID).Scan(&exists); err != nil {
		return false, err
	}
	return exists, nil
}
