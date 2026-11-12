package task

import (
	"regexp"
	"strings"
)

var (
	bioTelegramLinkRE = regexp.MustCompile(`(?i)(?:https?://)?(?:www\.)?(?:t\.me|telegram\.me)/([^\s<>()\[\]{}"'，。；、]+)`)
	bioIgnoredPaths = map[string]struct{}{
		"addemoji": {}, "addstickers": {}, "addtheme": {}, "addlist": {},
		"share": {}, "proxy": {}, "socks": {}, "login": {}, "confirmphone": {},
		"setlanguage": {}, "iv": {}, "boost": {}, "giftcode": {},
	}
)

// bioLinkCandidate is a normalized Telegram link found in a user's Bio.
// Public usernames are only candidates: callers MUST resolve them and accept
// only InputPeerChat/InputPeerChannel. A t.me username may identify a person,
// bot, or group, so parsing alone must never write it to the group library.
type bioLinkCandidate struct {
	LinkType string
	LinkHash string
	FullURL  string
}

// extractBioTelegramLinks extracts unique public usernames and private invite
// hashes from Bio text. It intentionally excludes Telegram utility/profile
// paths and internal message links; group-ness is determined after resolution.
func extractBioTelegramLinks(bio string) []bioLinkCandidate {
	matches := bioTelegramLinkRE.FindAllStringSubmatch(bio, -1)
	if len(matches) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(matches))
	result := make([]bioLinkCandidate, 0, len(matches))
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		path := strings.Trim(strings.TrimSpace(match[1]), ".,!?;:，。；、")
		path = strings.Trim(path, "/")
		if path == "" || strings.Contains(path, "/") {
			// Includes c/<channel>/<message>, joinchat is handled below.
			if !strings.HasPrefix(strings.ToLower(path), "joinchat/") {
				continue
			}
		}

		lowerPath := strings.ToLower(path)
		linkType, hash := "public", path
		if strings.HasPrefix(lowerPath, "joinchat/") {
			linkType = "private"
			hash = path[len("joinchat/"):]
		} else if strings.HasPrefix(path, "+") {
			linkType = "private"
			hash = strings.TrimPrefix(path, "+")
		} else {
			hash = strings.TrimPrefix(path, "@")
			if _, ignored := bioIgnoredPaths[strings.ToLower(hash)]; ignored {
				continue
			}
			if len(hash) < 5 || len(hash) > 32 {
				continue
			}
		}
		if hash == "" {
			continue
		}
		key := linkType + ":" + strings.ToLower(hash)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}

		urlPath := path
		if linkType == "private" && strings.HasPrefix(path, "+") {
			urlPath = path
		}
		result = append(result, bioLinkCandidate{
			LinkType: linkType,
			LinkHash: hash,
			FullURL:  "https://t.me/" + urlPath,
		})
	}
	return result
}
