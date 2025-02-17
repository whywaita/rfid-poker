package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/jinzhu/configor"

	"github.com/whywaita/rfid-poker/pkg/config"
	"github.com/whywaita/rfid-poker/pkg/server"
)

const (
	EnvConfigPath = "RFID_POKER_CONFIG_PATH"
)

func main() {
	if err := run(); err != nil {
		log.Fatalln(err)
	}
}

func run() error {
	ctx := context.Background()

	configFilePath, err := fetchConfigPath()
	if err != nil {
		return fmt.Errorf("fetchConfigPath(): %w", err)
	}

	if err := configor.Load(&config.Conf, configFilePath); err != nil {
		return fmt.Errorf("configor.Load(): %w", err)
	}

	if err := server.Run(ctx); err != nil {
		return fmt.Errorf("server.Run(ctx): %w", err)
	}
	return nil
}

func fetchConfigPath() (string, error) {
	input := os.Getenv(EnvConfigPath)
	if input == "" {
		return "./config.yaml", nil
	}

	_, err := os.Stat(input)
	if err == nil {
		// this is file path
		return input, nil
	}

	u, err := url.Parse(input)
	if err != nil {
		return "", fmt.Errorf("url.Parse(): %w", err)
	}
	switch u.Scheme {
	case "http", "https":
		return fetchHTTPConfigPath(u)
	default:
		return "", fmt.Errorf("unsupported scheme: %s", u.Scheme)
	}
}

func fetchHTTPConfigPath(u *url.URL) (string, error) {
	log.Printf("fetching config from %s", u.String())

	dir := os.TempDir()

	p := strings.Split(u.Path, "/")
	fileName := p[len(p)-1]

	fp := filepath.Join(dir, fileName)
	f, err := os.Create(fp)
	if err != nil {
		return "", fmt.Errorf("failed to create os file: %w", err)
	}
	defer f.Close()

	resp, err := http.Get(u.String())
	if err != nil {
		return "", fmt.Errorf("failed to get config via HTTP(S): %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to get config via HTTP(S): status code is not 200 (status code: %d)", resp.StatusCode)
	}

	if _, err := io.Copy(f, resp.Body); err != nil {
		return "", fmt.Errorf("failed to write file (path: %s): %w", fp, err)
	}

	return fp, nil
}
