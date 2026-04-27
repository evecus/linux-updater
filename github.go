package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// GHRelease is the subset of GitHub Release API we need.
type GHRelease struct {
	TagName string    `json:"tag_name"`
	Assets  []GHAsset `json:"assets"`
}

type GHAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
}

var ghClient = &http.Client{Timeout: 30 * time.Second}

// FetchLatestRelease returns the latest release for owner/repo.
func FetchLatestRelease(owner, repo string) (*GHRelease, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", owner, repo)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "linux-updater/1.0")

	resp, err := ghClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("github request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("github API returned %d", resp.StatusCode)
	}
	var rel GHRelease
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return nil, err
	}
	return &rel, nil
}

// ParseOwnerRepo extracts owner and repo from a GitHub URL.
func ParseOwnerRepo(repoURL string) (owner, repo string, err error) {
	// accept https://github.com/owner/repo or github.com/owner/repo
	repoURL = strings.TrimSuffix(repoURL, "/")
	repoURL = strings.TrimSuffix(repoURL, ".git")
	re := regexp.MustCompile(`github\.com[/:]([^/]+)/([^/]+)`)
	m := re.FindStringSubmatch(repoURL)
	if m == nil {
		return "", "", fmt.Errorf("cannot parse github URL: %s", repoURL)
	}
	return m[1], m[2], nil
}

// semver holds a parsed semantic version.
type semver struct {
	major, minor, patch int
	pre                 string // pre-release suffix
	raw                 string
}

var semverRe = regexp.MustCompile(`v?(\d+)\.(\d+)(?:\.(\d+))?(?:[.\-](.+))?`)

func parseSemver(s string) (semver, bool) {
	m := semverRe.FindStringSubmatch(strings.TrimSpace(s))
	if m == nil {
		return semver{raw: s}, false
	}
	major, _ := strconv.Atoi(m[1])
	minor, _ := strconv.Atoi(m[2])
	patch := 0
	if m[3] != "" {
		patch, _ = strconv.Atoi(m[3])
	}
	return semver{major: major, minor: minor, patch: patch, pre: m[4], raw: s}, true
}

// semverCompare returns true if b is newer than a.
func semverCompare(a, b string) (bIsNewer bool, err error) {
	sa, okA := parseSemver(a)
	sb, okB := parseSemver(b)

	if !okA || !okB {
		// fall back to string compare
		return a != b, nil
	}

	if sa.major != sb.major {
		return sb.major > sa.major, nil
	}
	if sa.minor != sb.minor {
		return sb.minor > sa.minor, nil
	}
	if sa.patch != sb.patch {
		return sb.patch > sa.patch, nil
	}
	// pre-release: empty (stable) > non-empty (pre)
	if sa.pre != sb.pre {
		if sa.pre == "" {
			return false, nil // a is stable, b is pre — not newer
		}
		if sb.pre == "" {
			return true, nil // b is stable, a is pre — b is newer
		}
		return sb.pre > sa.pre, nil
	}
	return false, nil
}

// BestAsset picks the release asset whose name best matches the keyword(s).
//
// Matching rules (applied in order):
//  1. Prefer assets that contain ALL keyword tokens (full-match group).
//     If no asset matches all tokens, fall back to partial matches.
//  2. Within the same group, prefer the asset with the SHORTEST filename
//     (fewest extra characters / fields beyond the matched tokens).
//  3. On equal length, prefer the asset whose matched-token character count
//     is highest (more of the filename is explained by the keywords).
func BestAsset(assets []GHAsset, keyword string) *GHAsset {
	if len(assets) == 0 {
		return nil
	}
	tokens := strings.Fields(strings.ToLower(keyword))
	if len(tokens) == 0 {
		return &assets[0]
	}

	type candidate struct {
		asset      *GHAsset
		matchedLen int
		allMatch   bool
	}

	var candidates []candidate
	for i := range assets {
		a := &assets[i]
		name := strings.ToLower(a.Name)
		matched := 0
		all := true
		for _, tok := range tokens {
			if strings.Contains(name, tok) {
				matched += len(tok)
			} else {
				all = false
			}
		}
		if matched > 0 {
			candidates = append(candidates, candidate{a, matched, all})
		}
	}

	if len(candidates) == 0 {
		return nil
	}

	// Determine whether any candidate matches all tokens.
	hasFullMatch := false
	for _, c := range candidates {
		if c.allMatch {
			hasFullMatch = true
			break
		}
	}

	// Keep only the best group (full-match preferred over partial).
	var best *GHAsset
	bestNameLen := -1
	bestMatchedLen := -1

	for _, c := range candidates {
		if hasFullMatch && !c.allMatch {
			continue
		}
		nameLen := len(c.asset.Name)
		// Prefer shorter filename; break ties by more matched characters.
		if best == nil ||
			nameLen < bestNameLen ||
			(nameLen == bestNameLen && c.matchedLen > bestMatchedLen) {
			best = c.asset
			bestNameLen = nameLen
			bestMatchedLen = c.matchedLen
		}
	}
	return best
}
