package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/devil1210/tsuruyomi-extensions/pkg/core"

	// Register extensions
	_ "github.com/devil1210/tsuruyomi-extensions/extensions/all/mangadex"
	_ "github.com/devil1210/tsuruyomi-extensions/extensions/es/lectormanga"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]
	ctx := context.Background()

	switch command {
	case "list":
		sources := core.ListSources()
		fmt.Printf("Registered Sources (%d):\n", len(sources))
		for _, s := range sources {
			meta := s.Metadata()
			fmt.Printf("  [%s] %s (ID: %d, Lang: %s, Ver: %s)\n", meta.ID, meta.Name, s.SourceID(), meta.Lang, meta.Version)
		}

	case "search":
		searchCmd := flag.NewFlagSet("search", flag.ExitOnError)
		sourceFlag := searchCmd.String("source", "mangadex", "Source ID or name")
		queryFlag := searchCmd.String("query", "", "Search query")
		_ = searchCmd.Parse(os.Args[2:])

		if *queryFlag == "" {
			fmt.Println("Error: --query is required")
			os.Exit(1)
		}

		s := findSource(*sourceFlag)
		if s == nil {
			fmt.Printf("Source %q not found\n", *sourceFlag)
			os.Exit(1)
		}

		res, err := s.SearchManga(ctx, *queryFlag, 1, nil)
		if err != nil {
			fmt.Printf("Search failed: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Found %d mangas:\n", len(res.Mangas))
		for _, m := range res.Mangas {
			fmt.Printf("  - %s (URL: %s)\n", m.Title, m.URL)
		}

	case "chapters":
		chapCmd := flag.NewFlagSet("chapters", flag.ExitOnError)
		sourceFlag := chapCmd.String("source", "mangadex", "Source ID or name")
		urlFlag := chapCmd.String("url", "", "Manga URL/path")
		_ = chapCmd.Parse(os.Args[2:])

		if *urlFlag == "" {
			fmt.Println("Error: --url is required")
			os.Exit(1)
		}

		s := findSource(*sourceFlag)
		if s == nil {
			fmt.Printf("Source %q not found\n", *sourceFlag)
			os.Exit(1)
		}

		chapters, err := s.GetChapterList(ctx, core.Manga{URL: *urlFlag})
		if err != nil {
			fmt.Printf("Failed to get chapters: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Found %d chapters:\n", len(chapters))
		for i, c := range chapters {
			if i >= 10 {
				fmt.Printf("  ... and %d more chapters\n", len(chapters)-10)
				break
			}
			fmt.Printf("  - [%.1f] %s (URL: %s)\n", c.ChapterNumber, c.Name, c.URL)
		}

	case "pages":
		pageCmd := flag.NewFlagSet("pages", flag.ExitOnError)
		sourceFlag := pageCmd.String("source", "mangadex", "Source ID or name")
		urlFlag := pageCmd.String("url", "", "Chapter URL/path")
		_ = pageCmd.Parse(os.Args[2:])

		if *urlFlag == "" {
			fmt.Println("Error: --url is required")
			os.Exit(1)
		}

		s := findSource(*sourceFlag)
		if s == nil {
			fmt.Printf("Source %q not found\n", *sourceFlag)
			os.Exit(1)
		}

		pages, err := s.GetPageList(ctx, core.Chapter{URL: *urlFlag})
		if err != nil {
			fmt.Printf("Failed to get pages: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Extracted %d pages:\n", len(pages))
		for i, p := range pages {
			if i >= 5 {
				fmt.Printf("  ... and %d more pages\n", len(pages)-5)
				break
			}
			fmt.Printf("  [%d] %s\n", i+1, p)
		}

	default:
		printUsage()
	}
}

func findSource(nameOrID string) core.Source {
	for _, s := range core.ListSources() {
		if strings.EqualFold(s.Metadata().ID, nameOrID) || strings.EqualFold(s.Metadata().Name, nameOrID) {
			return s
		}
		if fmt.Sprintf("%d", s.SourceID()) == nameOrID {
			return s
		}
	}
	return nil
}

func printUsage() {
	fmt.Println("Tsuruyomi Extensions CLI (tsuruext)")
	fmt.Println("Usage:")
	fmt.Println("  tsuruext list")
	fmt.Println("  tsuruext search --source mangadex --query \"Berserk\"")
	fmt.Println("  tsuruext chapters --source mangadex --url \"/title/<uuid>\"")
	fmt.Println("  tsuruext pages --source mangadex --url \"/chapter/<uuid>\"")
}
