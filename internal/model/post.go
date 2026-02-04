package model

import "time"

type Post struct {
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
