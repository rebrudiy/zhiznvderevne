package httpapp

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/gomarkdown/markdown"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"zhiznvderevne/internal/cache"
	"zhiznvderevne/internal/config"
	"zhiznvderevne/internal/db"
	"zhiznvderevne/internal/metrics"
)

//go:embed templates/*.html
var templatesFS embed.FS

type Server struct {
	cfg       *config.Config
	pool      *pgxpool.Pool
	router    *chi.Mux
	templates *template.Template
	metrics   *prometheus.Registry
	cache     cache.Cache
}

func New(cfg *config.Config, pool *pgxpool.Pool, metricsReg *prometheus.Registry, cacheClient cache.Cache) (*Server, error) {
	tmpl, err := template.New("base").ParseFS(templatesFS, "templates/*.html")
	if err != nil {
		return nil, err
	}
	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	r.Use(middleware.RealIP)
	r.Use(middleware.RequestID)

	s := &Server{cfg: cfg, pool: pool, router: r, templates: tmpl, metrics: metricsReg, cache: cacheClient}
	r.Get("/", s.handleIndex)
	r.Get("/post/{slug}", s.handlePost)
	r.Get("/api/posts", s.handlePostsAPI)
	r.Get("/api/posts/{slug}", s.handlePostAPI)

	if cfg.Observability.MetricsPath != "" {
		r.Handle(cfg.Observability.MetricsPath, promhttp.HandlerFor(metricsReg, promhttp.HandlerOpts{}))
	}
	return s, nil
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	rw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
	s.router.ServeHTTP(rw, r)
	route := chi.RouteContext(r.Context()).RoutePattern()
	if route == "" {
		route = r.URL.Path
	}
	metrics.HTTPRequests.WithLabelValues(r.Method, route, fmt.Sprintf("%d", rw.status)).Inc()
	metrics.HTTPDuration.WithLabelValues(r.Method, route).Observe(time.Since(start).Seconds())
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	posts, err := db.ListPublished(r.Context(), s.pool, 20)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	type postView struct {
		Title       string
		Slug        string
		PublishedAt string
		Tags        string
	}
	views := make([]postView, 0, len(posts))
	for _, p := range posts {
		published := ""
		if p.PublishedAt != nil {
			published = p.PublishedAt.Format("02.01.2006 15:04")
		}
		views = append(views, postView{
			Title:       p.Title,
			Slug:        p.Slug,
			PublishedAt: published,
			Tags:        strings.Join(p.Tags, ", "),
		})
	}

	data := map[string]any{
		"SiteTitle":      s.cfg.Site.Title,
		"Description":    s.cfg.Site.Description,
		"AuthorNickname": s.cfg.Site.AuthorNickname,
		"Posts":          views,
	}
	_ = s.templates.ExecuteTemplate(w, "index", data)
}

func (s *Server) handlePost(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	post, err := db.GetPostBySlug(r.Context(), s.pool, slug)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	published := ""
	if post.PublishedAt != nil {
		published = post.PublishedAt.Format("02.01.2006 15:04")
	}
	content := string(markdown.ToHTML([]byte(post.ContentMD), nil, nil))
	data := map[string]any{
		"SiteTitle":      s.cfg.Site.Title,
		"AuthorNickname": s.cfg.Site.AuthorNickname,
		"Post": map[string]any{
			"Title":       post.Title,
			"PublishedAt": published,
			"ContentHTML": template.HTML(content),
			"Tags":        strings.Join(post.Tags, ", "),
		},
	}
	_ = s.templates.ExecuteTemplate(w, "post", data)
}

func (s *Server) handlePostsAPI(w http.ResponseWriter, r *http.Request) {
	if s.cache != nil {
		if cached, ok := s.cache.Get(r.Context(), "posts:list:v1"); ok {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			_, _ = w.Write([]byte(cached))
			return
		}
	}
	posts, err := db.ListPublished(r.Context(), s.pool, 50)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	payload, err := json.Marshal(posts)
	if err != nil {
		http.Error(w, "encode error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_, _ = w.Write(payload)
	if s.cache != nil {
		s.cache.Set(r.Context(), "posts:list:v1", string(payload), 30*time.Second)
	}
}

func (s *Server) handlePostAPI(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	post, err := db.GetPostBySlug(r.Context(), s.pool, slug)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	writeJSON(w, post)
}

func (s *Server) Shutdown(ctx context.Context) error {
	return nil
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(v)
}
