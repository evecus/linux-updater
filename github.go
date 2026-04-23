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
// Keywords are space-separated; scoring is based on how many keyword tokens
// appear in the filename (case-insensitive), weighted by token length.
func BestAsset(assets []GHAsset, keyword string) *GHAsset {
	if len(assets) == 0 {
		return nil
	}
	tokens := strings.Fields(strings.ToLower(keyword))

	bestScore := -1
	var best *GHAsset

	for i := range assets {
		a := &assets[i]
		name := strings.ToLower(a.Name)
		score := 0
		for _, tok := range tokens {
			if strings.Contains(name, tok) {
				score += len(tok) // longer token match = higher weight
			}
		}
		if score > bestScore {
			bestScore = score
			best = a
		}
	}
	return best
}
