package main

import (
	"bytes"
	"compress/bzip2"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"archive/tar"
	"archive/zip"
)

var ghProxies = []string{
	"",                      // direct first
	"https://ghfast.top/",
	"https://gh-proxy.com/",
	"https://gh.ddlc.top/",
	"https://ghproxy.it/",
}

var dlClient = &http.Client{Timeout: 120 * time.Second}

// DownloadFile tries direct download then proxy mirrors.
// Returns the local temp file path. The temp file preserves the original
// filename's extension so that format-detection by suffix works correctly.
func DownloadFile(rawURL string, logger func(string)) (string, error) {
	// Extract the original filename from the URL to keep its extension.
	origName := filepath.Base(strings.Split(rawURL, "?")[0])
	ext := ""
	// Handle compound extensions like .tar.gz / .pkg.tar.zst
	for _, compound := range []string{
		".tar.gz", ".tar.xz", ".tar.bz2", ".tar.zst",
		".tar.lz4", ".tar.lz", ".pkg.tar.zst", ".pkg.tar.xz",
	} {
		if strings.HasSuffix(strings.ToLower(origName), compound) {
			ext = compound
			break
		}
	}
	if ext == "" {
		ext = filepath.Ext(origName) // e.g. ".deb", ".rpm", ".zip"
	}

	tmp, err := os.CreateTemp("", "updater-dl-*"+ext)
	if err != nil {
		return "", err
	}
	tmp.Close()

	for _, proxy := range ghProxies {
		url := proxy + rawURL
		if proxy == "" {
			logger(fmt.Sprintf("⬇ downloading (direct): %s", rawURL))
		} else {
			logger(fmt.Sprintf("⬇ retrying via proxy %s", proxy))
		}
		err := downloadTo(url, tmp.Name())
		if err == nil {
			logger("✓ download complete")
			return tmp.Name(), nil
		}
		logger(fmt.Sprintf("✗ failed: %v", err))
	}
	os.Remove(tmp.Name())
	return "", fmt.Errorf("all download attempts failed for %s", rawURL)
}

func downloadTo(url, dest string) error {
	resp, err := dlClient.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	f, err := os.OpenFile(dest, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, resp.Body)
	return err
}

// magicBytes reads the first 8 bytes of a file.
func magicBytes(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	buf := make([]byte, 8)
	n, _ := f.Read(buf)
	return buf[:n], nil
}

func hasPrefix(b []byte, hex string) bool {
	needle := make([]byte, len(hex)/2)
	for i := range needle {
		fmt.Sscanf(hex[i*2:i*2+2], "%02x", &needle[i])
	}
	return len(b) >= len(needle) && bytes.Equal(b[:len(needle)], needle)
}

// archiveExts lists all file extensions that should be treated as archives
// and recursively extracted. Order matters for suffix matching (longest first).
var archiveExts = []string{
	".tar.gz", ".tar.xz", ".tar.bz2", ".tar.zst", ".tar.lz4", ".tar.lz",
	".tgz", ".txz", ".tbz2",
	".tar",
	".zip",
	".gz",  // plain gzip (not tar)
	".xz",  // plain xz (not tar)
	".bz2", // plain bzip2 (not tar)
}

// isArchiveName returns true if the filename looks like a known archive.
func isArchiveName(name string) bool {
	low := strings.ToLower(name)
	for _, ext := range archiveExts {
		if strings.HasSuffix(low, ext) {
			return true
		}
	}
	return false
}

// hasNoExtension returns true for files with no dot in the base name,
// e.g. "sing-box", "frpc", "mihomo" — typical Linux binary names.
func hasNoExtension(name string) bool {
	base := filepath.Base(name)
	return !strings.Contains(base, ".")
}

// ExtractAndFindBinary extracts the downloaded file (recursively if needed)
// and returns the path of the ELF binary inside it.
// repoName / binaryKeyword are used only as tiebreakers when multiple ELFs exist.
func ExtractAndFindBinary(filePath, repoName, binaryKeyword string, logger func(string)) (string, error) {
	extractDir, err := os.MkdirTemp("", "updater-extract-*")
	if err != nil {
		return "", err
	}

	// extractOne extracts a single file into dir; returns a list of paths produced.
	var extractRecursive func(src, dir string, depth int) error

	extractRecursive = func(src, dir string, depth int) error {
		if depth > 5 {
			return fmt.Errorf("too many nested archive layers")
		}

		magic, err := magicBytes(src)
		if err != nil {
			return err
		}

		srcName := strings.ToLower(filepath.Base(src))

		switch {
		case hasPrefix(magic, "504b0304"): // ZIP
			logger(fmt.Sprintf("📦 [layer %d] zip → extracting %s", depth, filepath.Base(src)))
			if err := extractZip(src, dir); err != nil {
				return fmt.Errorf("unzip: %w", err)
			}

		case hasPrefix(magic, "1f8b"): // gzip
			logger(fmt.Sprintf("📦 [layer %d] gzip → extracting %s", depth, filepath.Base(src)))
			// Try tar.gz first; fall back to plain gz
			if err := extractTarGz(src, dir); err != nil {
				out := filepath.Join(dir, strings.TrimSuffix(filepath.Base(src), ".gz"))
				if err2 := extractGzip(src, out); err2 != nil {
					return fmt.Errorf("gzip extract: %w", err)
				}
			}

		case hasPrefix(magic, "fd377a585a00"): // xz
			logger(fmt.Sprintf("📦 [layer %d] xz → extracting %s", depth, filepath.Base(src)))
			// Try tar.xz first; fall back to plain xz
			if err := extractTarXz(src, dir); err != nil {
				out := filepath.Join(dir, strings.TrimSuffix(filepath.Base(src), ".xz"))
				f, err2 := os.Open(src)
				if err2 != nil {
					return fmt.Errorf("xz open: %w", err2)
				}
				xzr, err2 := newXzReader(f)
				f.Close()
				if err2 != nil {
					return fmt.Errorf("xz reader: %w", err2)
				}
				outf, err2 := os.Create(out)
				if err2 != nil {
					xzr.Close()
					return err2
				}
				_, err2 = io.Copy(outf, xzr)
				outf.Close()
				xzr.Close()
				if err2 != nil {
					return fmt.Errorf("xz decompress: %w", err2)
				}
			}

		case hasPrefix(magic, "425a68"): // bzip2
			logger(fmt.Sprintf("📦 [layer %d] bzip2 → extracting %s", depth, filepath.Base(src)))
			if err := extractTarBz2(src, dir); err != nil {
				return fmt.Errorf("bz2 extract: %w", err)
			}

		case hasPrefix(magic, "7f454c46"): // ELF — already a binary
			logger(fmt.Sprintf("🔧 [layer %d] ELF binary: %s", depth, filepath.Base(src)))
			dest := filepath.Join(dir, filepath.Base(src))
			return copyFile(src, dest)

		default:
			// Magic bytes unknown — try to guess from extension
			switch {
			case strings.HasSuffix(srcName, ".tar"):
				logger(fmt.Sprintf("📦 [layer %d] bare tar → %s", depth, filepath.Base(src)))
				if err := extractTarAuto(src, dir); err != nil {
					return fmt.Errorf("tar extract: %w", err)
				}
			default:
				logger(fmt.Sprintf("⚠ [layer %d] unknown format, copying as-is: %s", depth, filepath.Base(src)))
				dest := filepath.Join(dir, filepath.Base(src))
				return copyFile(src, dest)
			}
		}

		// After extraction, recurse into any archive files that appeared
		return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() || path == src {
				return nil
			}
			if isArchiveName(info.Name()) {
				subDir := filepath.Join(dir, "__sub_"+info.Name())
				if err := os.MkdirAll(subDir, 0755); err != nil {
					return err
				}
				if err := extractRecursive(path, subDir, depth+1); err != nil {
					return err
				}
				os.Remove(path) // replace archive with its contents
			}
			return nil
		})
	}

	if err := extractRecursive(filePath, extractDir, 1); err != nil {
		return "", err
	}

	return findBinary(extractDir, repoName, binaryKeyword, logger)
}

// findBinary walks dir and picks the ELF binary using this priority:
//  1. Files with NO extension (classic Linux binary naming) that are ELF
//  2. Any ELF file (confirmed by magic bytes 7F 45 4C 46)
//  3. Tiebreak by binaryKeyword match, then repoName match, then largest size
func findBinary(dir, hint, binaryKeyword string, logger func(string)) (string, error) {
	type candidate struct {
		path      string
		size      int64
		noExt     bool // true = no file extension
		kwMatch   bool // matches binaryKeyword
		hintMatch bool // matches repoName
	}

	var elfs []candidate // only confirmed ELF files

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		magic, _ := magicBytes(path)
		if !hasPrefix(magic, "7f454c46") {
			return nil // not ELF, skip
		}
		name := strings.ToLower(filepath.Base(path))
		c := candidate{
			path:  path,
			size:  info.Size(),
			noExt: hasNoExtension(path),
		}
		if binaryKeyword != "" {
			c.kwMatch = strings.Contains(name, strings.ToLower(binaryKeyword))
		}
		if hint != "" {
			c.hintMatch = strings.Contains(name, strings.ToLower(hint))
		}
		elfs = append(elfs, c)
		return nil
	})
	if err != nil {
		return "", err
	}
	if len(elfs) == 0 {
		return "", fmt.Errorf("no ELF binary found in extracted archive")
	}

	// Sort: noExt > kwMatch > hintMatch > size
	sort.Slice(elfs, func(i, j int) bool {
		a, b := elfs[i], elfs[j]
		if a.noExt != b.noExt {
			return a.noExt // no-extension wins
		}
		if a.kwMatch != b.kwMatch {
			return a.kwMatch
		}
		if a.hintMatch != b.hintMatch {
			return a.hintMatch
		}
		return a.size > b.size
	})

	chosen := elfs[0]
	reason := ""
	switch {
	case chosen.kwMatch:
		reason = fmt.Sprintf("keyword=%q", binaryKeyword)
	case chosen.hintMatch:
		reason = fmt.Sprintf("repo name=%q", hint)
	case chosen.noExt:
		reason = "no file extension"
	default:
		reason = "largest ELF"
	}
	logger(fmt.Sprintf("🔍 selected binary: %s  (reason: %s)", filepath.Base(chosen.path), reason))
	return chosen.path, nil
}

// ---- archive helpers ----

func extractZip(src, dst string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()
	for _, f := range r.File {
		outPath := filepath.Join(dst, filepath.Clean(f.Name))
		if f.FileInfo().IsDir() {
			os.MkdirAll(outPath, 0755)
			continue
		}
		os.MkdirAll(filepath.Dir(outPath), 0755)
		rc, err := f.Open()
		if err != nil {
			return err
		}
		out, err := os.OpenFile(outPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, f.Mode())
		if err != nil {
			rc.Close()
			return err
		}
		io.Copy(out, rc)
		rc.Close()
		out.Close()
	}
	return nil
}

func extractTarGz(src, dst string) error {
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gz.Close()
	return extractTar(tar.NewReader(gz), dst)
}

func extractGzip(src, dst string) error {
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gz.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, gz)
	return err
}

func extractTarBz2(src, dst string) error {
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer f.Close()
	return extractTar(tar.NewReader(bzip2.NewReader(f)), dst)
}

func extractTarXz(src, dst string) error {
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer f.Close()
	xzr, err := newXzReader(f)
	if err != nil {
		return err
	}
	defer xzr.Close()
	return extractTar(tar.NewReader(xzr), dst)
}

func extractTarAuto(src, dst string) error {
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer f.Close()
	return extractTar(tar.NewReader(f), dst)
}

func extractTar(tr *tar.Reader, dst string) error {
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		outPath := filepath.Join(dst, filepath.Clean(hdr.Name))
		switch hdr.Typeflag {
		case tar.TypeDir:
			os.MkdirAll(outPath, 0755)
		case tar.TypeReg, tar.TypeRegA:
			os.MkdirAll(filepath.Dir(outPath), 0755)
			out, err := os.OpenFile(outPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, hdr.FileInfo().Mode())
			if err != nil {
				return err
			}
			io.Copy(out, tr)
			out.Close()
		}
	}
	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	info, _ := in.Stat()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode())
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}
