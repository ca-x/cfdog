package ghrelase

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type tag struct {
	ReleaseTag string `json:"tag_name"`
}

// GetLatest Get the latest released version of repo
func GetLatest(repoName string) (string, error) {
	repo := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", repoName)
	resp, err := http.Get(repo)
	defer resp.Body.Close()
	if err != nil {
		return "", err
	}

	all, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	var t tag
	if err = json.Unmarshal(all, &t); err != nil {
		return "", err
	}
	return t.ReleaseTag, nil
}
