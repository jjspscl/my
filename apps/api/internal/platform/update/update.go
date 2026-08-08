package update

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const latestReleaseURL = "https://api.github.com/repos/jjspscl/my/releases/latest"

type releaseResponse struct {
	TagName string `json:"tag_name"`
}

type Result struct {
	Current string
	Latest  string
}

func Check(ctx context.Context, current string) (Result, error) {
	latest, err := latestVersion(ctx, latestReleaseURL)
	if err != nil {
		return Result{}, err
	}
	return Result{Current: current, Latest: latest}, nil
}

func (r Result) HasUpdate() bool {
	return Compare(r.Latest, r.Current) > 0
}

func (r Result) Message() string {
	if r.Current == "dev" {
		return "development build; latest release is " + r.Latest
	}
	if r.HasUpdate() {
		return fmt.Sprintf("update available: current %s, latest %s\ninstall: curl -fsSL https://raw.githubusercontent.com/jjspscl/my/%s/scripts/install-mcp.sh | sh", r.Current, r.Latest, r.Latest)
	}
	return fmt.Sprintf("up to date: %s", r.Current)
}

func Compare(left, right string) int {
	leftParts := versionParts(left)
	rightParts := versionParts(right)
	for i := 0; i < 3; i++ {
		if leftParts[i] < rightParts[i] {
			return -1
		}
		if leftParts[i] > rightParts[i] {
			return 1
		}
	}
	return 0
}

func versionParts(value string) [3]int {
	var parts [3]int
	value = strings.TrimPrefix(strings.TrimSpace(value), "v")
	for i, part := range strings.Split(value, ".") {
		if i >= len(parts) {
			break
		}
		for _, digit := range part {
			if digit < '0' || digit > '9' {
				break
			}
			parts[i] = parts[i]*10 + int(digit-'0')
		}
	}
	return parts
}

func latestVersion(ctx context.Context, endpoint string) (string, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", err
	}
	request.Header.Set("Accept", "application/vnd.github+json")
	request.Header.Set("User-Agent", "my-mcp-update-check")
	client := &http.Client{Timeout: 10 * time.Second}
	response, err := client.Do(request)
	if err != nil {
		return "", fmt.Errorf("check latest release: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("check latest release: unexpected status %s", response.Status)
	}
	var release releaseResponse
	if err := json.NewDecoder(response.Body).Decode(&release); err != nil {
		return "", fmt.Errorf("decode latest release: %w", err)
	}
	if release.TagName == "" {
		return "", fmt.Errorf("latest release has empty tag")
	}
	return release.TagName, nil
}
