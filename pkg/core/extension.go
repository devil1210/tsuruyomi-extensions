package core

import (
	"context"
	"fmt"
	"sync"
)

// ExtensionMetadata contains information about a compiled or dynamic extension.
type ExtensionMetadata struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	PkgName     string `json:"pkgName"`
	Version     string `json:"version"`
	VersionCode int    `json:"versionCode"`
	Lang        string `json:"lang"`
	IsNsfw      bool   `json:"isNsfw"`
	IconURL     string `json:"iconUrl"`
	BaseURL     string `json:"baseUrl"`
}

// Source represents a manga catalog source.
type Source interface {
	// Metadata returns the source's metadata.
	Metadata() ExtensionMetadata

	// SourceID returns the numeric unique ID of this source.
	SourceID() int64

	// GetPopularManga fetches the most popular manga.
	GetPopularManga(ctx context.Context, page int) (*MangaPageResult, error)

	// GetLatestUpdates fetches newly updated manga.
	GetLatestUpdates(ctx context.Context, page int) (*MangaPageResult, error)

	// SearchManga searches manga by query and filters.
	SearchManga(ctx context.Context, query string, page int, filters []Filter) (*MangaPageResult, error)

	// GetMangaDetails fetches full details for a manga.
	GetMangaDetails(ctx context.Context, manga Manga) (*Manga, error)

	// GetChapterList fetches the list of chapters for a manga.
	GetChapterList(ctx context.Context, manga Manga) ([]Chapter, error)

	// GetPageList fetches page URLs for a chapter.
	GetPageList(ctx context.Context, chapter Chapter) ([]string, error)
}

var (
	registryMu sync.RWMutex
	registry   = make(map[int64]Source)
	byPkg      = make(map[string]Source)
)

// Register registers a source instance.
func Register(s Source) {
	registryMu.Lock()
	defer registryMu.Unlock()
	registry[s.SourceID()] = s
	byPkg[s.Metadata().PkgName] = s
}

// GetSource retrieves a source by its numeric ID.
func GetSource(sourceID int64) (Source, bool) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	s, ok := registry[sourceID]
	return s, ok
}

// GetSourceByPkg retrieves a source by package name.
func GetSourceByPkg(pkgName string) (Source, bool) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	s, ok := byPkg[pkgName]
	return s, ok
}

// ListSources returns all registered sources.
func ListSources() []Source {
	registryMu.RLock()
	defer registryMu.RUnlock()
	list := make([]Source, 0, len(registry))
	for _, s := range registry {
		list = append(list, s)
	}
	return list
}

// SourceNotFoundError is returned when a requested source cannot be found.
type SourceNotFoundError struct {
	SourceID int64
}

func (e *SourceNotFoundError) Error() string {
	return fmt.Sprintf("source with ID %d not found", e.SourceID)
}
