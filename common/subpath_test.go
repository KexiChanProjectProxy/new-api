package common

import (
	"testing"
)

func TestParseRelaySubpaths(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		raw     string
		want    []string
		wantErr bool
	}{
		{
			name:    "empty string returns empty slice",
			raw:     "",
			want:    []string{},
			wantErr: false,
		},
		{
			name:    "whitespace only returns empty slice",
			raw:     "   ",
			want:    []string{},
			wantErr: false,
		},
		{
			name:    "single valid subpath",
			raw:     "/a/b",
			want:    []string{"/a/b"},
			wantErr: false,
		},
		{
			name:    "UUID subpath",
			raw:     "/123e4567-e89b-12d3-a456-426614174000/123e4567-e89b-12d3-a456-426614174111",
			want:    []string{"/123e4567-e89b-12d3-a456-426614174000/123e4567-e89b-12d3-a456-426614174111"},
			wantErr: false,
		},
		{
			name:    "multiple valid subpaths",
			raw:     "/a/b,/c/d",
			want:    []string{"/a/b", "/c/d"},
			wantErr: false,
		},
		{
			name:    "multiple subpaths with whitespace",
			raw:     "/a/b , /c/d ",
			want:    []string{"/a/b", "/c/d"},
			wantErr: false,
		},
		{
			name:    "empty string is invalid",
			raw:     "",
			want:    []string{},
			wantErr: false,
		},
		{
			name:    "double slash is invalid",
			raw:     "//",
			want:    nil,
			wantErr: true,
		},
		{
			name:    "no leading slash is invalid",
			raw:     "a/b",
			want:    nil,
			wantErr: true,
		},
		{
			name:    "double slash in middle is invalid",
			raw:     "/a//b",
			want:    nil,
			wantErr: true,
		},
		{
			name:    "trailing slash creates empty segment",
			raw:     "/a/b/",
			want:    nil,
			wantErr: true,
		},
		{
			name:    "just slash alone is invalid",
			raw:     "/",
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := ParseRelaySubpaths(tt.raw)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseRelaySubpaths() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && len(got) != len(tt.want) {
				t.Errorf("ParseRelaySubpaths() got %v, want %v", got, tt.want)
				return
			}
			if !tt.wantErr {
				for i, v := range got {
					if v != tt.want[i] {
						t.Errorf("ParseRelaySubpaths()[%d] = %v, want %v", i, v, tt.want[i])
					}
				}
			}
		})
	}
}

func TestNormalizeSubpath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "trims whitespace",
			input: "  /a/b  ",
			want:  "/a/b",
		},
		{
			name:  "removes trailing slash",
			input: "/a/b/",
			want:  "/a/b",
		},
		{
			name:  "collapses double slashes",
			input: "/a//b",
			want:  "/a/b",
		},
		{
			name:  "adds leading slash if missing",
			input: "a/b",
			want:  "/a/b",
		},
		{
			name:  "removes leading multiple slashes",
			input: "///a/b",
			want:  "/a/b",
		},
		{
			name:  "trims multiple slashes",
			input: "///a/b",
			want:  "/a/b",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := normalizeSubpath(tt.input)
			if got != tt.want {
				t.Errorf("normalizeSubpath(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestValidateSubpath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:    "valid simple path",
			input:   "/a/b",
			want:    "/a/b",
			wantErr: false,
		},
		{
			name:    "valid UUID path",
			input:   "/123e4567-e89b-12d3-a456-426614174000/123e4567-e89b-12d3-a456-426614174111",
			want:    "/123e4567-e89b-12d3-a456-426614174000/123e4567-e89b-12d3-a456-426614174111",
			wantErr: false,
		},
		{
			name:    "empty string is invalid",
			input:   "",
			want:    "",
			wantErr: true,
		},
		{
			name:    "no leading slash is invalid",
			input:   "a/b",
			want:    "",
			wantErr: true,
		},
		{
			name:    "double slash is invalid",
			input:   "//",
			want:    "",
			wantErr: true,
		},
		{
			name:    "trailing slash is invalid",
			input:   "/a/b/",
			want:    "",
			wantErr: true,
		},
		{
			name:    "just slash alone is invalid",
			input:   "/",
			want:    "",
			wantErr: true,
		},
		{
			name:    "double slash in middle is invalid",
			input:   "/a//b",
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := validateSubpath(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateSubpath(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("validateSubpath(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}