package core

// MangaStatus represents the publication status of a manga.
type MangaStatus int

const (
	StatusUnknown            MangaStatus = 0
	StatusOngoing            MangaStatus = 1
	StatusCompleted          MangaStatus = 2
	StatusLicensed           MangaStatus = 3
	StatusPublishingFinished MangaStatus = 4
	StatusCancelled          MangaStatus = 5
	StatusOnHiatus           MangaStatus = 6
)

// Manga represents a manga title and its associated metadata.
type Manga struct {
	ID          int64       `json:"id"`
	SourceID    int64       `json:"sourceId"`
	URL         string      `json:"url"`
	Title       string      `json:"title"`
	Artist      string      `json:"artist,omitempty"`
	Author      string      `json:"author,omitempty"`
	Description string      `json:"description,omitempty"`
	Genre       []string    `json:"genre,omitempty"`
	Status      MangaStatus `json:"status"`
	Thumbnail   string      `json:"thumbnail,omitempty"`
	Initialized bool        `json:"initialized"`
}

// Chapter represents a chapter of a manga.
type Chapter struct {
	ID            int64   `json:"id"`
	MangaID       int64   `json:"mangaId"`
	URL           string  `json:"url"`
	Name          string  `json:"name"`
	DateUpload    int64   `json:"dateUpload"`
	ChapterNumber float32 `json:"chapterNumber"`
	Scanlator     string  `json:"scanlator,omitempty"`
	SourceOrder   int     `json:"sourceOrder"`
}

// Page represents a single page of a chapter.
type Page struct {
	Index    int    `json:"index"`
	URL      string `json:"url"`
	ImageURL string `json:"imageUrl"`
}

// Filter represents a search filter option.
type Filter struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	Type    string   `json:"type"` // "select", "text", "checkbox", "tri-state"
	Values  []string `json:"values,omitempty"`
	Current string   `json:"current,omitempty"`
}

// MangaPageResult represents a paginated list of mangas.
type MangaPageResult struct {
	Mangas      []Manga `json:"mangas"`
	HasNextPage bool    `json:"hasNextPage"`
}
