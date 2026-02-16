package main

import (
	"testing"
)

func TestMockDiaryGenerator_GenerateDiary(t *testing.T) {
	generator := &MockDiaryGenerator{}

	content, err := generator.GenerateDiary("/path/to/image.jpg")
	if err != nil {
		t.Fatalf("GenerateDiary failed: %v", err)
	}

	if content == "" {
		t.Error("expected non-empty content, got empty string")
	}

	// モック実装は固定の文字列を返すことを確認
	expected := "この植物は順調に成長しています。葉の色が鮮やかで、新しい芽も見られます。"
	if content != expected {
		t.Errorf("expected content '%s', got '%s'", expected, content)
	}
}
