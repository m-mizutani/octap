package usecase_test

import (
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/m-mizutani/octap/pkg/usecase"
)

func TestParseGitHubURL(t *testing.T) {
	testCases := []struct {
		name      string
		url       string
		wantOwner string
		wantRepo  string
	}{
		{
			name:      "SSH URL",
			url:       "git@github.com:m-mizutani/octap.git",
			wantOwner: "m-mizutani",
			wantRepo:  "octap",
		},
		{
			name:      "HTTPS URL",
			url:       "https://github.com/m-mizutani/octap.git",
			wantOwner: "m-mizutani",
			wantRepo:  "octap",
		},
		{
			name:      "SSH URL with ssh://",
			url:       "ssh://git@github.com/m-mizutani/octap.git",
			wantOwner: "m-mizutani",
			wantRepo:  "octap",
		},
		{
			name:      "Without .git suffix",
			url:       "https://github.com/m-mizutani/octap",
			wantOwner: "m-mizutani",
			wantRepo:  "octap",
		},
		{
			name:      "Invalid URL",
			url:       "https://example.com/something",
			wantOwner: "",
			wantRepo:  "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			owner, repo := usecase.ParseGitHubURL(tc.url)
			gt.Equal(t, owner, tc.wantOwner)
			gt.Equal(t, repo, tc.wantRepo)
		})
	}
}
