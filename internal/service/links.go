package service

import (
	"path/filepath"
	"regexp"
	"strings"
)

var wikiLinkRe = regexp.MustCompile(`\[\[([^\[\]|]+)(?:\|[^\[\]]+)?\]\]`)

func parseWikiLinks(text []byte) []string {
	seen := map[string]struct{}{}
	out := []string{}
	for _, m := range wikiLinkRe.FindAllSubmatch(text, -1) {
		name := strings.TrimSpace(string(m[1]))
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		out = append(out, name)
	}
	return out
}

func isLinkSource(name, contentType string) bool {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".md", ".markdown", ".txt", ".text", ".json", ".yaml", ".yml", ".xml", ".csv":
		return true
	}
	ct := strings.ToLower(contentType)
	return strings.HasPrefix(ct, "text/") ||
		ct == "application/json" ||
		ct == "application/markdown" ||
		ct == "application/x-yaml" ||
		ct == "application/xml"
}
