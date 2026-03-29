package blog

import (
	"bytes"
	"embed"
	"errors"
	"html/template"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/gorilla/mux"
	"github.com/rs/zerolog/log"
	"github.com/yuin/goldmark"
	"go.yaml.in/yaml/v3"
)

//go:embed templates
var templateFS embed.FS

type Post struct {
	Slug        string
	Title       string
	Date        string
	HTMLContent template.HTML
}

type Blog struct {
	contentPath string
	indexTmpl   *template.Template
	postTmpl    *template.Template
}

func New(contentPath string) (*Blog, error) {
	indexTmpl, err := template.ParseFS(templateFS, "templates/index.html")
	if err != nil {
		return nil, err
	}
	postTmpl, err := template.ParseFS(templateFS, "templates/post.html")
	if err != nil {
		return nil, err
	}
	return &Blog{
		contentPath: contentPath,
		indexTmpl:   indexTmpl,
		postTmpl:    postTmpl,
	}, nil
}

func (b *Blog) Register(r *mux.Router) {
	log.Trace().
		Str("method", "GET").
		Str("path", "/blog").
		Msg("Adding route")
	r.HandleFunc("/blog", b.handleIndex).Methods("GET")
	r.HandleFunc("/blog/{slug}", b.handlePost).Methods("GET")
}

func (b *Blog) loadPosts() ([]Post, error) {
	entries, err := os.ReadDir(b.contentPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}

	var posts []Post
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".md" {
			continue
		}

		data, err := os.ReadFile(filepath.Join(b.contentPath, entry.Name()))
		if err != nil {
			log.Error().Err(err).Str("file", entry.Name()).Msg("failed to read blog post")
			continue
		}

		post, err := parsePost(data)
		if err != nil {
			log.Error().Err(err).Str("file", entry.Name()).Msg("failed to parse blog post")
			continue
		}

		post.Slug = strings.TrimSuffix(entry.Name(), ".md")
		posts = append(posts, post)
	}

	sort.Slice(posts, func(i, j int) bool {
		return posts[i].Date > posts[j].Date
	})

	return posts, nil
}

func (b *Blog) loadPost(slug string) (*Post, error) {
	// slug is pre-validated to contain only [a-zA-Z0-9-], no path traversal possible
	path := filepath.Join(b.contentPath, slug+".md")
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}

	post, err := parsePost(data)
	if err != nil {
		return nil, err
	}
	post.Slug = slug
	return &post, nil
}

func parsePost(data []byte) (Post, error) {
	var post Post
	content := strings.ReplaceAll(string(data), "\r\n", "\n")

	const delimiter = "---\n"
	if strings.HasPrefix(content, delimiter) {
		second := strings.Index(content[len(delimiter):], delimiter)
		if second != -1 {
			frontmatter := content[len(delimiter) : len(delimiter)+second]
			content = strings.TrimSpace(content[len(delimiter)+second+len(delimiter):])

			var metadata struct {
				Title string `yaml:"title"`
				Date  string `yaml:"date"`
			}
			if err := yaml.Unmarshal([]byte(frontmatter), &metadata); err != nil {
				return Post{}, err
			}

			post.Title = strings.TrimSpace(metadata.Title)
			post.Date = strings.TrimSpace(metadata.Date)
		}
	}

	var buf bytes.Buffer
	if err := goldmark.New().Convert([]byte(content), &buf); err != nil {
		return Post{}, err
	}
	// Content originates from trusted server-side markdown files, not user input.
	post.HTMLContent = template.HTML(buf.String()) //nolint:gosec

	return post, nil
}

func (b *Blog) handleIndex(w http.ResponseWriter, r *http.Request) {
	posts, err := b.loadPosts()
	if err != nil {
		log.Error().Err(err).Msg("failed to load blog posts")
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := b.indexTmpl.Execute(w, map[string]interface{}{"Posts": posts}); err != nil {
		log.Error().Err(err).Msg("failed to render blog index")
	}
}

func (b *Blog) handlePost(w http.ResponseWriter, r *http.Request) {
	slug := mux.Vars(r)["slug"]

	for _, c := range slug {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-') {
			http.NotFound(w, r)
			return
		}
	}

	post, err := b.loadPost(slug)
	if err != nil {
		log.Error().Err(err).Msg("failed to load blog post")
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if post == nil {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := b.postTmpl.Execute(w, post); err != nil {
		log.Error().Err(err).Msg("failed to render blog post")
	}
}
