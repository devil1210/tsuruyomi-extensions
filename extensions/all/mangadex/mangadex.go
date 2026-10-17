package mangadex

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"time"

	"github.com/devil1210/tsuruyomi-extensions/pkg/core"
	"github.com/devil1210/tsuruyomi-extensions/pkg/utils"
)

const (
	SourceID = 6400665728063187402
	BaseURL  = "https://mangadex.org"
	ApiURL   = "https://api.mangadex.org"
)

func init() {
	core.Register(New())
}

type MangaDex struct{}

func New() *MangaDex {
	return &MangaDex{}
}

func (m *MangaDex) SourceID() int64 {
	return SourceID
}

func (m *MangaDex) Metadata() core.ExtensionMetadata {
	return core.ExtensionMetadata{
		ID:          "mangadex",
		Name:        "MangaDex",
		PkgName:     "eu.kanade.tachiyomi.extension.all.mangadex",
		Version:     "1.0.0",
		VersionCode: 1,
		Lang:        "all",
		IsNsfw:      false,
		IconURL:     "https://mangadex.org/favicon.ico",
		BaseURL:     BaseURL,
	}
}

var reUUID = regexp.MustCompile(`[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}`)

type mangaDexSearchResponse struct {
	Result string `json:"result"`
	Data   []struct {
		ID         string `json:"id"`
		Attributes struct {
			Title       map[string]string `json:"title"`
			Description map[string]string `json:"description"`
			Status      string            `json:"status"`
		} `json:"attributes"`
		Relationships []struct {
			ID         string `json:"id"`
			Type       string `json:"type"`
			Attributes struct {
				FileName string `json:"fileName"`
			} `json:"attributes"`
		} `json:"relationships"`
	} `json:"data"`
	Total  int `json:"total"`
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

func (m *MangaDex) GetPopularManga(ctx context.Context, page int) (*core.MangaPageResult, error) {
	limit := 20
	offset := (page - 1) * limit
	target := fmt.Sprintf("%s/manga?limit=%d&offset=%d&includes[]=cover_art&order[followedCount]=desc&contentRating[]=safe&contentRating[]=suggestive", ApiURL, limit, offset)
	return m.fetchMangaList(ctx, target)
}

func (m *MangaDex) GetLatestUpdates(ctx context.Context, page int) (*core.MangaPageResult, error) {
	limit := 20
	offset := (page - 1) * limit
	target := fmt.Sprintf("%s/manga?limit=%d&offset=%d&includes[]=cover_art&order[latestUploadedChapter]=desc&contentRating[]=safe&contentRating[]=suggestive", ApiURL, limit, offset)
	return m.fetchMangaList(ctx, target)
}

func (m *MangaDex) SearchManga(ctx context.Context, query string, page int, filters []core.Filter) (*core.MangaPageResult, error) {
	limit := 20
	offset := (page - 1) * limit
	target := fmt.Sprintf("%s/manga?limit=%d&offset=%d&title=%s&includes[]=cover_art&contentRating[]=safe&contentRating[]=suggestive", ApiURL, limit, offset, url.QueryEscape(query))
	return m.fetchMangaList(ctx, target)
}

func (m *MangaDex) fetchMangaList(ctx context.Context, targetURL string) (*core.MangaPageResult, error) {
	data, err := utils.FetchJSON(ctx, targetURL, BaseURL)
	if err != nil {
		return nil, err
	}

	var res mangaDexSearchResponse
	if err := json.Unmarshal(data, &res); err != nil {
		return nil, err
	}

	mangas := make([]core.Manga, 0, len(res.Data))
	for _, item := range res.Data {
		title := ""
		for _, t := range item.Attributes.Title {
			title = t
			break
		}

		thumbnail := ""
		for _, rel := range item.Relationships {
			if rel.Type == "cover_art" && rel.Attributes.FileName != "" {
				thumbnail = fmt.Sprintf("https://uploads.mangadex.org/covers/%s/%s.256.jpg", item.ID, rel.Attributes.FileName)
				break
			}
		}

		mangas = append(mangas, core.Manga{
			SourceID:    SourceID,
			URL:         fmt.Sprintf("/title/%s", item.ID),
			Title:       title,
			Thumbnail:   thumbnail,
			Status:      core.StatusOngoing,
			Initialized: true,
		})
	}

	return &core.MangaPageResult{
		Mangas:      mangas,
		HasNextPage: (res.Offset + res.Limit) < res.Total,
	}, nil
}

func (m *MangaDex) GetMangaDetails(ctx context.Context, manga core.Manga) (*core.Manga, error) {
	mUUID := reUUID.FindString(manga.URL)
	if mUUID == "" {
		return &manga, nil
	}

	apiURL := fmt.Sprintf("%s/manga/%s?includes[]=author&includes[]=artist&includes[]=cover_art", ApiURL, mUUID)
	data, err := utils.FetchJSON(ctx, apiURL, BaseURL)
	if err != nil {
		return &manga, err
	}

	var res struct {
		Data struct {
			Attributes struct {
				Title       map[string]string `json:"title"`
				Description map[string]string `json:"description"`
				Status      string            `json:"status"`
			} `json:"attributes"`
			Relationships []struct {
				Type       string `json:"type"`
				Attributes struct {
					Name     string `json:"name"`
					FileName string `json:"fileName"`
				} `json:"attributes"`
			} `json:"relationships"`
		} `json:"data"`
	}

	if err := json.Unmarshal(data, &res); err != nil {
		return &manga, err
	}

	for _, t := range res.Data.Attributes.Title {
		manga.Title = t
		break
	}
	for _, d := range res.Data.Attributes.Description {
		manga.Description = d
		break
	}

	for _, rel := range res.Data.Relationships {
		switch rel.Type {
		case "author":
			manga.Author = rel.Attributes.Name
		case "artist":
			manga.Artist = rel.Attributes.Name
		case "cover_art":
			if rel.Attributes.FileName != "" {
				manga.Thumbnail = fmt.Sprintf("https://uploads.mangadex.org/covers/%s/%s.512.jpg", mUUID, rel.Attributes.FileName)
			}
		}
	}

	manga.Initialized = true
	return &manga, nil
}

func (m *MangaDex) GetChapterList(ctx context.Context, manga core.Manga) ([]core.Chapter, error) {
	mUUID := reUUID.FindString(manga.URL)
	if mUUID == "" {
		return nil, fmt.Errorf("invalid manga UUID")
	}

	var allChapters []core.Chapter
	offset := 0
	limit := 100

	for {
		feedURL := fmt.Sprintf("%s/manga/%s/feed?limit=%d&offset=%d&order[chapter]=desc&translatedLanguage[]=es&translatedLanguage[]=es-la&translatedLanguage[]=en", ApiURL, mUUID, limit, offset)
		data, err := utils.FetchJSON(ctx, feedURL, BaseURL)
		if err != nil {
			break
		}

		var feedRes struct {
			Data []struct {
				ID         string `json:"id"`
				Attributes struct {
					Volume      string `json:"volume"`
					Chapter     string `json:"chapter"`
					Title       string `json:"title"`
					PublishAt   string `json:"publishAt"`
					ReadableAt  string `json:"readableAt"`
				} `json:"attributes"`
			} `json:"data"`
			Total int `json:"total"`
		}

		if err := json.Unmarshal(data, &feedRes); err != nil {
			break
		}

		for _, item := range feedRes.Data {
			chNum, _ := strconv.ParseFloat(item.Attributes.Chapter, 32)
			title := item.Attributes.Title
			if title == "" {
				title = fmt.Sprintf("Capítulo %s", item.Attributes.Chapter)
			}

			var dateUp int64
			if t, err := time.Parse(time.RFC3339, item.Attributes.PublishAt); err == nil {
				dateUp = t.UnixMilli()
			}

			allChapters = append(allChapters, core.Chapter{
				URL:           fmt.Sprintf("/chapter/%s", item.ID),
				Name:          title,
				ChapterNumber: float32(chNum),
				DateUpload:    dateUp,
				SourceOrder:   len(allChapters),
			})
		}

		offset += limit
		if offset >= feedRes.Total || len(feedRes.Data) == 0 {
			break
		}
	}

	return allChapters, nil
}

type mangaDexAtHomeResponse struct {
	Result  string `json:"result"`
	BaseURL string `json:"baseUrl"`
	Chapter struct {
		Hash      string   `json:"hash"`
		Data      []string `json:"data"`
		DataSaver []string `json:"dataSaver"`
	} `json:"chapter"`
}

func (m *MangaDex) GetPageList(ctx context.Context, chapter core.Chapter) ([]string, error) {
	chapterUUID := reUUID.FindString(chapter.URL)
	if chapterUUID == "" {
		return nil, fmt.Errorf("could not extract chapter UUID from MangaDex URL: %s", chapter.URL)
	}

	apiURL := fmt.Sprintf("%s/at-home/server/%s", ApiURL, chapterUUID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "Tsuruyomi/1.0 (https://github.com/devil1210/tsuruyomi)")
	req.Header.Set("Accept", "application/json")

	resp, err := utils.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("calling MangaDex at-home API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("MangaDex API returned status %d for chapter %s", resp.StatusCode, chapterUUID)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var res mangaDexAtHomeResponse
	if err := json.Unmarshal(body, &res); err != nil {
		return nil, fmt.Errorf("parsing MangaDex JSON: %w", err)
	}

	if res.Result != "ok" || len(res.Chapter.Data) == 0 {
		return nil, fmt.Errorf("invalid response from MangaDex API: %s", res.Result)
	}

	pages := make([]string, len(res.Chapter.Data))
	for i, fn := range res.Chapter.Data {
		pages[i] = fmt.Sprintf("%s/data/%s/%s", res.BaseURL, res.Chapter.Hash, fn)
	}

	return pages, nil
}
