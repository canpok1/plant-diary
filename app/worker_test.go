package main

import (
	"testing"
	"time"
)

func TestParseCreatedAtFromFilename_NewFormat(t *testing.T) {
	// 新形式（YYYYMMDD_HHMM_UTC.jpg）のテスト
	createdAt, err := parseCreatedAtFromFilename("/path/to/20260216_1110_UTC.jpg")
	if err != nil {
		t.Fatalf("parseCreatedAtFromFilename failed: %v", err)
	}

	expected := time.Date(2026, 2, 16, 11, 10, 0, 0, time.UTC)
	if !createdAt.Equal(expected) {
		t.Errorf("expected %v, got %v", expected, createdAt)
	}
}

func TestParseCreatedAtFromFilename_OldFormat(t *testing.T) {
	// 旧形式（YYYYMMDD_HHMM.jpg）のテスト（後方互換性）
	createdAt, err := parseCreatedAtFromFilename("/path/to/20260216_1110.jpg")
	if err != nil {
		t.Fatalf("parseCreatedAtFromFilename failed: %v", err)
	}

	// 旧形式はローカルタイムとして解釈される
	expected := time.Date(2026, 2, 16, 11, 10, 0, 0, time.UTC)
	if !createdAt.Equal(expected) {
		t.Errorf("expected %v, got %v", expected, createdAt)
	}
}

func TestParseCreatedAtFromFilename_InvalidFormat(t *testing.T) {
	// 不正なファイル名形式のテスト
	testCases := []string{
		"/path/to/invalid.jpg",
		"/path/to/20260216.jpg",
		"/path/to/not_a_date.jpg",
		"/path/to/2026_1110.jpg",
	}

	for _, tc := range testCases {
		_, err := parseCreatedAtFromFilename(tc)
		if err == nil {
			t.Errorf("expected error for invalid filename %s, got nil", tc)
		}
	}
}

func TestParseCreatedAtFromFilename_DifferentPaths(t *testing.T) {
	// 異なるパス形式でもベース名から正しくパースできることを確認
	testCases := []struct {
		path     string
		expected time.Time
	}{
		{
			path:     "20260216_1110_UTC.jpg",
			expected: time.Date(2026, 2, 16, 11, 10, 0, 0, time.UTC),
		},
		{
			path:     "/data/photos/20260216_1110_UTC.jpg",
			expected: time.Date(2026, 2, 16, 11, 10, 0, 0, time.UTC),
		},
		{
			path:     "/workspaces/plant-diary/data/photos/20260216_1110_UTC.jpg",
			expected: time.Date(2026, 2, 16, 11, 10, 0, 0, time.UTC),
		},
	}

	for _, tc := range testCases {
		createdAt, err := parseCreatedAtFromFilename(tc.path)
		if err != nil {
			t.Errorf("parseCreatedAtFromFilename(%s) failed: %v", tc.path, err)
			continue
		}

		if !createdAt.Equal(tc.expected) {
			t.Errorf("path %s: expected %v, got %v", tc.path, tc.expected, createdAt)
		}
	}
}
