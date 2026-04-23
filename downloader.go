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
// Returns the local temp file path.
func DownloadFile(rawURL string, logger func(string)) (string, error) {
	tmp, err := os.CreateTemp("", "updater-dl-*")
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

// ExtractAndFindBinary extracts the downloaded file and returns the path to
// the best candidate executable inside it.
// repoName is used as a hint when multiple executables are present.
func ExtractAndFindBinary(filePath, repoName string, logger func(string)) (string, error) {
	magic, err := magicBytes(filePath)
	if err != nil {
		return "", err
	}

	extractDir, err := os.MkdirTemp("", "updater-extract-*")
	if err != nil {
		return "", err
	}

	switch {
	case hasPrefix(magic, "504b0304"): // ZIP
		logger("📦 detected zip archive, extracting...")
		if err := extractZip(filePath, extractDir); err != nil {
			return "", fmt.Errorf("unzip: %w", err)
		}

	case hasPrefix(magic, "1f8b"): // gzip / tar.gz
		logger("📦 detected gzip, extracting...")
		if err := extractTarGz(filePath, extractDir); err != nil {
			// maybe plain gzip, not tar
			out := filepath.Join(extractDir, "binary_file")
			if err2 := extractGzip(filePath, out); err2 != nil {
				return "", fmt.Errorf("gzip extract: %w", err)
			}
		}

	case hasPrefix(magic, "fd377a585a00"): // xz / tar.xz
		logger("📦 detected xz archive, extracting...")
		if err := extractTarXz(filePath, extractDir); err != nil {
			return "", fmt.Errorf("xz extract: %w", err)
		}

	case hasPrefix(magic, "425a68"): // bzip2 / tar.bz2
		logger("📦 detected bzip2 archive, extracting...")
		if err := extractTarBz2(filePath, extractDir); err != nil {
			return "", fmt.Errorf("bz2 extract: %w", err)
		}

	case hasPrefix(magic, "7f454c46"): // ELF
		logger("🔧 detected ELF binary directly")
		dest := filepath.Join(extractDir, "binary_file")
		if err := copyFile(filePath, dest); err != nil {
			return "", err
		}

	default:
		logger("⚠ unknown format, trying tar then raw copy...")
		if err := extractTarAuto(filePath, extractDir); err != nil {
			dest := filepath.Join(extractDir, "binary_file")
			if err2 := copyFile(filePath, dest); err2 != nil {
				return "", fmt.Errorf("cannot handle file format")
			}
		}
	}

	return findBinary(extractDir, repoName, logger)
}

// findBinary walks extractDir and picks the best executable candidate.
func findBinary(dir, hint string, logger func(string)) (string, error) {
	type candidate struct {
		path string
		size int64
		prio int // higher = better
	}
	var candidates []candidate

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		magic, _ := magicBytes(path)
		isELF := hasPrefix(magic, "7f454c46")
		isExec := info.Mode()&0111 != 0

		prio := 0
		if isELF {
			prio += 100
		}
		if isExec {
			prio += 50
		}
		name := strings.ToLower(filepath.Base(path))
		hintLow := strings.ToLower(hint)
		if strings.Contains(name, hintLow) {
			prio += 30
		}
		// penalise docs / metadata
		for _, ext := range []string{".txt", ".md", ".json", ".yaml", ".yml", ".toml", ".conf", ".sh"} {
			if strings.HasSuffix(name, ext) {
				prio -= 200
			}
		}

		if prio > -100 {
			candidates = append(candidates, candidate{path, info.Size(), prio})
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	if len(candidates) == 0 {
		return "", fmt.Errorf("no files found in extracted archive")
	}

	// sort: higher prio first, then larger size
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].prio != candidates[j].prio {
			return candidates[i].prio > candidates[j].prio
		}
		return candidates[i].size > candidates[j].size
	})

	chosen := candidates[0].path
	logger(fmt.Sprintf("🔍 selected binary: %s", filepath.Base(chosen)))
	return chosen, nil
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
