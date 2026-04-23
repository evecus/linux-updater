package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
)

type xzReadCloser struct {
	cmd    *exec.Cmd
	stdout io.ReadCloser
	tmpIn  string
}

func (x *xzReadCloser) Read(p []byte) (int, error) {
	return x.stdout.Read(p)
}

func (x *xzReadCloser) Close() error {
	x.stdout.Close()
	x.cmd.Wait()
	if x.tmpIn != "" {
		os.Remove(x.tmpIn)
	}
	return nil
}

// newXzReader decompresses xz data from r using the system xz or xzdec binary.
func newXzReader(r io.Reader) (*xzReadCloser, error) {
	bin, err := findXzBin()
	if err != nil {
		return nil, err
	}

	tmp, err := os.CreateTemp("", "updater-xz-*.xz")
	if err != nil {
		return nil, err
	}
	if _, err := io.Copy(tmp, r); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return nil, err
	}
	tmp.Close()

	cmd := exec.Command(bin, "-d", "-c", tmp.Name())
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		os.Remove(tmp.Name())
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		os.Remove(tmp.Name())
		return nil, fmt.Errorf("xz start failed: %w", err)
	}
	return &xzReadCloser{cmd: cmd, stdout: stdout, tmpIn: tmp.Name()}, nil
}

func findXzBin() (string, error) {
	for _, name := range []string{"xz", "xzdec"} {
		if path, err := exec.LookPath(name); err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("xz/xzdec not found; install xz-utils to handle .tar.xz files")
}
