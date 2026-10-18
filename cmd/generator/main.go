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
	Lang    string `json:"Lang"`
	Folder  string `json:"Folder"`
	Name    string `json:"Name"`
	Theme   string `json:"Theme"`
	BaseURL string `json:"BaseURL"`
}

func isRelevantLang(lang string) bool {
	l := strings.ToLower(strings.TrimSpace(lang))
	return strings.HasPrefix(l, "es") || strings.HasPrefix(l, "en") || l == "all"
}

func normalizeEngine(theme string) string {
	t := strings.ToLower(strings.TrimSpace(theme))
	switch t {
	case "madara", "yiimadara", "madaranoajax":
		return "madara"
	case "mangathemesia":
		return "mangathemesia"
	default:
		return ""
	}
}

func main() {
	fmt.Println("=== Tsuruyomi Extension Catalog Generator (ES / EN / ALL) ===")

	// 1. Load existing curated index.json
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

	// 2. Load theme mappings (es, en, all)
	themeMap := make(map[string]ThemeMapping)
	if data, err := os.ReadFile("all_themes_map.json"); err == nil {
		var list []ThemeMapping
		if err := json.Unmarshal(data, &list); err == nil {
			for _, m := range list {
				// Key by exact package name suffix: lang.folder
				pkgKey := fmt.Sprintf("%s.%s", strings.ToLower(m.Lang), strings.ToLower(m.Folder))
				themeMap[pkgKey] = m
				themeMap[strings.ToLower(m.Folder)] = m
				themeMap[strings.ToLower(strings.ReplaceAll(m.Name, " ", ""))] = m
			}
			fmt.Printf("Loaded %d theme mappings from all_themes_map.json\n", len(list))
		}
	}

	// Also load keiyoushi_es_themes.json for any extra fallback
	if data, err := os.ReadFile("keiyoushi_es_themes.json"); err == nil {
		var list []ThemeMapping
		if err := json.Unmarshal(data, &list); err == nil {
			for _, m := range list {
				pkgKey := fmt.Sprintf("es.%s", strings.ToLower(m.Folder))
				if _, exists := themeMap[pkgKey]; !exists {
					themeMap[pkgKey] = m
				}
				if _, exists := themeMap[strings.ToLower(m.Folder)]; !exists {
					themeMap[strings.ToLower(m.Folder)] = m
				}
			}
		}
	}

	// 3. Fetch Keiyoushi index.json
	fmt.Println("Fetching Keiyoushi index.json...")
	client := &http.Client{Timeout: 35 * time.Second}
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
	skippedNoTheme := 0
	skippedLang := 0

	for _, kExt := range keiRepo.ExtensionList.Extensions {
		pkg := kExt.PackageName
		if pkg == "" {
			continue
		}

		// Check if already in our curated repository
		if cur, ok := existingByPkg[pkg]; ok {
			cur.Name = kExt.Name
			if kExt.VersionName != "" {
				cur.Version = kExt.VersionName
			}
			existingByPkg[pkg] = cur
			updatedCount++
			continue
		}

		// Check language relevance
		// Check if pkg contains .es., .es-419., .en., .all.
		parts := strings.Split(pkg, ".")
		langSegment := ""
		if len(parts) >= 5 {
			langSegment = parts[len(parts)-2]
		}

		isRelevant := isRelevantLang(langSegment)
		if !isRelevant {
			for _, s := range kExt.Sources {
				if isRelevantLang(s.Language) {
					isRelevant = true
					if langSegment == "" {
						langSegment = s.Language
					}
					break
				}
			}
		}

		if !isRelevant {
			skippedLang++
			continue
		}

		// Look up theme
		shortName := strings.ToLower(parts[len(parts)-1])
		pkgKey := fmt.Sprintf("%s.%s", strings.ToLower(langSegment), shortName)

		mapping, hasTheme := themeMap[pkgKey]
		if !hasTheme {
			mapping, hasTheme = themeMap[shortName]
		}
		if !hasTheme {
			mapping, hasTheme = themeMap[strings.ToLower(strings.ReplaceAll(kExt.Name, " ", ""))]
		}

		rawTheme := ""
		if hasTheme {
			rawTheme = mapping.Theme
		}

		engine := normalizeEngine(rawTheme)
		if engine == "" {
			skippedNoTheme++
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
			if base == "" {
				continue
			}

			srcLang := s.Language
			if srcLang == "" {
				srcLang = langSegment
			}

			tSources = append(tSources, TsuruSource{
				Name:    s.Name,
				ID:      idStr,
				BaseURL: base,
				Lang:    srcLang,
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

		mainLang := langSegment
		if mainLang == "" && len(tSources) > 0 {
			mainLang = tSources[0].Lang
		}

		newExt := TsuruExtension{
			Name:    kExt.Name,
			Pkg:     pkg,
			Apk:     shortName + ".json",
			Lang:    mainLang,
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
	seen := make(map[string]bool)
	for _, ext := range existing {
		if updated, ok := existingByPkg[ext.Pkg]; ok {
			finalList = append(finalList, updated)
			seen[ext.Pkg] = true
		}
	}
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
	fmt.Printf("Total extensions in catalog: %d (Curated: %d, Newly Imported: %d)\n",
		len(finalList), len(existing), addedCount)
	fmt.Printf("Skipped non-target languages: %d | Skipped unmapped themes: %d\n", skippedLang, skippedNoTheme)
}
