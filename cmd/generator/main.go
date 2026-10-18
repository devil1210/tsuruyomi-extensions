package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// KeiyoushiRepo represents the top-level structure of Keiyoushi's index.json
type KeiyoushiRepo struct {
	Name          string `json:"name"`
	BadgeLabel    string `json:"badgeLabel"`
	ExtensionList struct {
		Extensions []KeiyoushiExtension `json:"extensions"`
	} `json:"extensionList"`
}

type KeiyoushiExtension struct {
	Name           string `json:"name"`
	PackageName    string `json:"packageName"`
	ExtensionLib   string `json:"extensionLib"`
	VersionCode    any    `json:"versionCode"`
	VersionName    string `json:"versionName"`
	ContentWarning string `json:"contentWarning"`
	Resources      struct {
		ApkURL  string `json:"apkUrl"`
		IconURL string `json:"iconUrl"`
	} `json:"resources"`
	Sources []KeiyoushiSource `json:"sources"`
}

type KeiyoushiSource struct {
	ID       any    `json:"id"`
	Name     string `json:"name"`
	Language string `json:"language"`
	HomeURL  string `json:"homeUrl"`
}

// TsuruExtension matches the format expected by Tsuruyomi index.json
type TsuruExtension struct {
	Name         string        `json:"name"`
	Pkg          string        `json:"pkg"`
	Apk          string        `json:"apk"`
	Lang         string        `json:"lang"`
	Code         int64         `json:"code"`
	Version      string        `json:"version"`
	Nsfw         int           `json:"nsfw"`
	HasReadme    int           `json:"hasReadme"`
	HasChangelog int           `json:"hasChangelog"`
	Sources      []TsuruSource `json:"sources"`
}

type TsuruSource struct {
	Name           string `json:"name"`
	ID             string `json:"id"`
	BaseURL        string `json:"baseUrl"`
	Lang           string `json:"lang"`
	Engine         string `json:"engine"`
	ContentWarning int    `json:"contentWarning,omitempty"`
}

type ThemeMapping struct {
	Folder  string `json:"Folder"`
	Name    string `json:"Name"`
	Theme   string `json:"Theme"`
	BaseURL string `json:"BaseURL"`
}

func main() {
	fmt.Println("=== Tsuruyomi Extension Catalog Generator ===")

	// 1. Load existing index.json
	indexPath := filepath.Join(".", "index.json")
	var existing []TsuruExtension
	if data, err := os.ReadFile(indexPath); err == nil {
		_ = json.Unmarshal(data, &existing)
		fmt.Printf("Loaded %d existing curated extensions from %s\n", len(existing), indexPath)
	}

	existingByPkg := make(map[string]TsuruExtension)
	for _, ext := range existing {
		existingByPkg[ext.Pkg] = ext
	}

	// 2. Load theme mappings
	themeMap := make(map[string]ThemeMapping)
	if data, err := os.ReadFile("keiyoushi_es_themes.json"); err == nil {
		var list []ThemeMapping
		if err := json.Unmarshal(data, &list); err == nil {
			for _, m := range list {
				themeMap[strings.ToLower(m.Folder)] = m
				themeMap[strings.ToLower(strings.ReplaceAll(m.Name, " ", ""))] = m
			}
			fmt.Printf("Loaded %d theme mappings from keiyoushi_es_themes.json\n", len(list))
		}
	}

	// 3. Fetch Keiyoushi index.json
	fmt.Println("Fetching Keiyoushi index.json...")
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get("https://raw.githubusercontent.com/keiyoushi/extensions/repo/index.json")
	if err != nil {
		log.Fatalf("Error fetching Keiyoushi index: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Error reading response: %v", err)
	}

	var keiRepo KeiyoushiRepo
	if err := json.Unmarshal(body, &keiRepo); err != nil {
		log.Fatalf("Error unmarshaling Keiyoushi repo: %v", err)
	}

	fmt.Printf("Parsed %d total extensions from Keiyoushi\n", len(keiRepo.ExtensionList.Extensions))

	// 4. Merge and process extensions
	addedCount := 0
	updatedCount := 0

	for _, kExt := range keiRepo.ExtensionList.Extensions {
		pkg := kExt.PackageName
		if pkg == "" {
			continue
		}

		// Check if already in our curated repository
		if cur, ok := existingByPkg[pkg]; ok {
			// Update version/icon if newer
			cur.Name = kExt.Name
			if kExt.VersionName != "" {
				cur.Version = kExt.VersionName
			}
			existingByPkg[pkg] = cur
			updatedCount++
			continue
		}

		// Detect if it's Spanish or relevant
		isSpanish := strings.Contains(pkg, ".extension.es.")
		if !isSpanish {
			for _, s := range kExt.Sources {
				if s.Language == "es" || s.Language == "es-419" {
					isSpanish = true
					break
				}
			}
		}

		// If not Spanish, skip for now to keep catalog focused and clean
		if !isSpanish {
			continue
		}

		// Look up theme from our theme mapping
		parts := strings.Split(pkg, ".")
		shortName := strings.ToLower(parts[len(parts)-1])

		mapping, hasTheme := themeMap[shortName]
		if !hasTheme {
			mapping, hasTheme = themeMap[strings.ToLower(strings.ReplaceAll(kExt.Name, " ", ""))]
		}

		engine := ""
		if hasTheme {
			engine = mapping.Theme
		}

		// Only import if it uses a supported native engine (madara, mangathemesia)
		if engine != "madara" && engine != "mangathemesia" {
			continue
		}

		// Build sources list
		var tSources []TsuruSource
		for _, s := range kExt.Sources {
			idStr := fmt.Sprintf("%v", s.ID)
			base := s.HomeURL
			if base == "" && hasTheme && mapping.BaseURL != "" {
				base = mapping.BaseURL
			}
			tSources = append(tSources, TsuruSource{
				Name:    s.Name,
				ID:      idStr,
				BaseURL: base,
				Lang:    s.Language,
				Engine:  engine,
			})
		}

		if len(tSources) == 0 {
			continue
		}

		nsfwVal := 0
		if strings.Contains(kExt.ContentWarning, "NSFW") {
			nsfwVal = 1
		}

		var vCode int64 = 1
		switch vc := kExt.VersionCode.(type) {
		case float64:
			vCode = int64(vc)
		case int64:
			vCode = vc
		case int:
			vCode = int64(vc)
		}

		lang := "es"
		if len(tSources) > 0 && tSources[0].Lang != "" {
			lang = tSources[0].Lang
		}

		newExt := TsuruExtension{
			Name:    kExt.Name,
			Pkg:     pkg,
			Apk:     shortName + ".json",
			Lang:    lang,
			Code:    vCode,
			Version: kExt.VersionName,
			Nsfw:    nsfwVal,
			Sources: tSources,
		}

		existingByPkg[pkg] = newExt
		addedCount++
	}

	// 5. Build final list maintaining stable ordering
	var finalList []TsuruExtension
	// Put original existing items first
	seen := make(map[string]bool)
	for _, ext := range existing {
		if updated, ok := existingByPkg[ext.Pkg]; ok {
			finalList = append(finalList, updated)
			seen[ext.Pkg] = true
		}
	}
	// Append newly added extensions
	for pkg, ext := range existingByPkg {
		if !seen[pkg] {
			finalList = append(finalList, ext)
		}
	}

	// 6. Write to index.json
	out, err := json.MarshalIndent(finalList, "", "  ")
	if err != nil {
		log.Fatalf("Error marshaling JSON: %v", err)
	}

	if err := os.WriteFile(indexPath, out, 0644); err != nil {
		log.Fatalf("Error writing %s: %v", indexPath, err)
	}

	fmt.Printf("Successfully generated %s!\n", indexPath)
	fmt.Printf("Total extensions in catalog: %d (Curated/Existing: %d, Newly Imported from Keiyoushi: %d)\n",
		len(finalList), len(existing), addedCount)
}
