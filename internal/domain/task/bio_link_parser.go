package task

import (
	"context"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	bioTelegramLinkRE = regexp.MustCompile(`(?i)(?:https?://)?(?:www\.)?(?:t\.me|telegram\.me|telegram\.dog)/([^\s<>()\[\]{}"'，。；、]+)`)
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
		path = strings.SplitN(path, "?", 2)[0]
		path = strings.SplitN(path, "#", 2)[0]
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


// waitTelegramFloodWait honors an explicit Telegram FLOOD_WAIT_N response.
// The caller controls the overall context/deadline; no retry is attempted if
// the requested wait exceeds that deadline.
func waitTelegramFloodWait(ctx context.Context, err error) bool {
	if err == nil { return false }
	message := strings.ToUpper(err.Error())
	const marker = "FLOOD_WAIT_"
	index := strings.Index(message, marker)
	if index < 0 { return false }
	start := index + len(marker)
	end := start
	for end < len(message) && message[end] >= '0' && message[end] <= '9' { end++ }
	if end == start { return false }
	seconds, parseErr := strconv.Atoi(message[start:end])
	if parseErr != nil || seconds <= 0 { return false }
	timer := time.NewTimer(time.Duration(seconds) * time.Second)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}
