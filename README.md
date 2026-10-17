# 鶴読み Tsuruyomi Extensions

[![Go Report Card](https://goreportcard.com/badge/github.com/devil1210/tsuruyomi-extensions)](https://goreportcard.com/report/github.com/devil1210/tsuruyomi-extensions)
[![License: MIT](https://img.shields.io/badge/License-MIT-crimson.svg)](LICENSE)

Native Go extensions for **[Tsuruyomi](https://github.com/devil1210/tsuruyomi)**.

Designed to replace bloated JVM / Android APK sidecars (Tachiyomi APK extensions) with high-performance, lightweight, and type-safe native Go scrapers.

---

## ⚡ Features

- **🚀 Native Performance:** Up to 100x faster than Android Dalvik/ART sidecar execution with negligible memory footprint (~50KB per extension vs ~30MB per APK).
- **🔒 Type Safe:** Built with standard Go interfaces (`core.Source`, `core.Manga`, `core.Chapter`).
- **🛡️ FlareSolverr Integration:** First-class Cloudflare Turnstile & DDoS-Guard bypass support.
- **📦 Modular & Plug-and-Play:** Zero native CGO required; extensions can be compiled in, imported as Go packages, or invoked via CLI.
- **🌐 Universal Multi-Language Support:** Standardized metadata matching Tachiyomi package naming conventions for 100% frontend compatibility.

---

## 📁 Repository Structure

```
tsuruyomi-extensions/
├── cmd/
│   └── tsuruext/              # CLI runner for testing extensions
├── pkg/
│   ├── core/                  # Source interfaces, data models, and registry
│   │   ├── extension.go
│   │   └── models.go
│   └── utils/                 # HTTP client, realistic headers, text parsers
│       └── http.go
└── extensions/
    ├── all/
    │   └── mangadex/          # MangaDex (Multi-language)
    └── es/
        ├── lectormanga/       # LectorManga.lat (Spanish)
        ├── shadowmanga/       # ShadowManga (Spanish)
        └── mangaoni/          # MangaOni (Spanish)
```

---

## 🛠️ CLI Usage (`tsuruext`)

You can test any scraper directly from your terminal:

```bash
# List all registered sources
go run ./cmd/tsuruext list

# Search for manga
go run ./cmd/tsuruext search --source mangadex --query "Frieren"

# List chapters of a manga
go run ./cmd/tsuruext chapters --source mangadex --url "/title/<uuid>"

# Extract page images of a chapter
go run ./cmd/tsuruext pages --source mangadex --url "/chapter/<uuid>"
```

---

## 🧩 Implementing a New Extension

To create a new scraper, implement the `core.Source` interface in `extensions/<lang>/<source_name>`:

```go
package mysource

import (
    "context"
    "github.com/devil1210/tsuruyomi-extensions/pkg/core"
)

func init() {
    core.Register(&MySource{})
}

type MySource struct{}

func (s *MySource) SourceID() int64 { return 1234567890 }
func (s *MySource) Metadata() core.ExtensionMetadata {
    return core.ExtensionMetadata{
        ID:      "mysource",
        Name:    "My Source",
        PkgName: "eu.kanade.tachiyomi.extension.en.mysource",
        Lang:    "en",
        Version: "1.0.0",
        BaseURL: "https://example.com",
    }
}

func (s *MySource) GetPopularManga(ctx context.Context, page int) (*core.MangaPageResult, error) { ... }
func (s *MySource) GetLatestUpdates(ctx context.Context, page int) (*core.MangaPageResult, error) { ... }
func (s *MySource) SearchManga(ctx context.Context, query string, page int, filters []core.Filter) (*core.MangaPageResult, error) { ... }
func (s *MySource) GetMangaDetails(ctx context.Context, manga core.Manga) (*core.Manga, error) { ... }
func (s *MySource) GetChapterList(ctx context.Context, manga core.Manga) ([]core.Chapter, error) { ... }
func (s *MySource) GetPageList(ctx context.Context, chapter core.Chapter) ([]string, error) { ... }
```

---

## 📄 License

MIT © [devil1210](https://github.com/devil1210)
