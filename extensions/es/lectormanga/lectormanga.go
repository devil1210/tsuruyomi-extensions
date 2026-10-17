package lectormanga

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/devil1210/tsuruyomi-extensions/pkg/core"
	"github.com/devil1210/tsuruyomi-extensions/pkg/utils"
)

const (
	SourceID = 7178038133263703638
	BaseURL  = "https://lector-mangas.lat"
	ApiURL   = "https://api.zerocomics.net/api"
)

func init() {
	core.Register(New())
}

type LectorManga struct{}

func New() *LectorManga {
	return &LectorManga{}
}

func (l *LectorManga) SourceID() int64 {
	return SourceID
}

func (l *LectorManga) Metadata() core.ExtensionMetadata {
	return core.ExtensionMetadata{
		ID:          "lectormanga",
		Name:        "LectorManga.lat",
		PkgName:     "eu.kanade.tachiyomi.extension.es.lectormangalat",
		Version:     "1.0.0",
		VersionCode: 1,
		Lang:        "es",
		IsNsfw:      false,
		IconURL:     "https://lector-mangas.lat/favicon.ico",
		BaseURL:     BaseURL,
	}
}

var reDigits = regexp.MustCompile(`\d+`)

func (l *LectorManga) GetPopularManga(ctx context.Context, page int) (*core.MangaPageResult, error) {
	return &core.MangaPageResult{Mangas: []core.Manga{}, HasNextPage: false}, nil
}

func (l *LectorManga) GetLatestUpdates(ctx context.Context, page int) (*core.MangaPageResult, error) {
	return &core.MangaPageResult{Mangas: []core.Manga{}, HasNextPage: false}, nil
}

func (l *LectorManga) SearchManga(ctx context.Context, query string, page int, filters []core.Filter) (*core.MangaPageResult, error) {
	return &core.MangaPageResult{Mangas: []core.Manga{}, HasNextPage: false}, nil
}

func (l *LectorManga) GetMangaDetails(ctx context.Context, manga core.Manga) (*core.Manga, error) {
	manga.Initialized = true
	return &manga, nil
}

func (l *LectorManga) GetChapterList(ctx context.Context, manga core.Manga) ([]core.Chapter, error) {
	return []core.Chapter{}, nil
}

type zeroComicsResponse struct {
	Data struct {
		ID      int      `json:"id"`
		Numero  float64  `json:"numero"`
		Paginas []string `json:"paginas"`
	} `json:"data"`
}

func (l *LectorManga) GetPageList(ctx context.Context, chapter core.Chapter) ([]string, error) {
	matches := reDigits.FindAllString(chapter.URL, -1)
	if len(matches) == 0 {
		return nil, fmt.Errorf("could not extract chapter ID from URL: %s", chapter.URL)
	}
	chapterID := matches[len(matches)-1]

	endpoint := fmt.Sprintf("%s/capitulos/%s", ApiURL, chapterID)
	data, err := utils.FetchJSON(ctx, endpoint, BaseURL)
	if err != nil {
		return nil, fmt.Errorf("calling ZeroComics API: %w", err)
	}

	var res zeroComicsResponse
	if err := json.Unmarshal(data, &res); err != nil {
		return nil, fmt.Errorf("parsing ZeroComics JSON: %w", err)
	}

	if len(res.Data.Paginas) == 0 {
		return nil, fmt.Errorf("no pages returned from ZeroComics API for chapter %s", chapterID)
	}

	pages := make([]string, len(res.Data.Paginas))
	for i, p := range res.Data.Paginas {
		pages[i] = strings.ReplaceAll(p, `\/`, `/`)
	}

	return pages, nil
}
