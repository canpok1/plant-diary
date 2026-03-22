package handler

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"plant-diary/internal/domain"
)

func TestExpandPrompt_BookName(t *testing.T) {
	capturedAt := time.Date(2026, 3, 22, 5, 30, 0, 0, time.UTC) // JST: 14:30
	result := expandPrompt("{{book_name}}の日記", "ミニトマト", capturedAt, nil)
	if result != "ミニトマトの日記" {
		t.Errorf("expected 'ミニトマトの日記', got '%s'", result)
	}
}

func TestExpandPrompt_Datetime(t *testing.T) {
	capturedAt := time.Date(2026, 3, 22, 5, 30, 0, 0, time.UTC) // JST: 14:30
	result := expandPrompt("現在は{{datetime}}です。", "トマト", capturedAt, nil)
	if !strings.Contains(result, "2026年03月22日 14時30分 (JST)") {
		t.Errorf("expected datetime placeholder to be expanded, got '%s'", result)
	}
}

func TestExpandPrompt_PastDiariesWithDiaries(t *testing.T) {
	capturedAt := time.Date(2026, 3, 22, 5, 30, 0, 0, time.UTC)
	time1 := time.Date(2026, 1, 20, 11, 10, 0, 0, time.UTC) // JST: 2026-01-20 20:10
	time2 := time.Date(2026, 2, 1, 10, 30, 0, 0, time.UTC)  // JST: 2026-02-01 19:30
	pastDiaries := []domain.Diary{
		{ID: 1, Content: "新しい芽が出ました。", CreatedAt: time1},
		{ID: 2, Content: "葉が大きく成長しています。", CreatedAt: time2},
	}

	result := expandPrompt("{{past_diaries}}", "トマト", capturedAt, pastDiaries)

	if !strings.Contains(result, "過去1ヶ月の観察記録") {
		t.Error("expected past diaries header")
	}
	if !strings.Contains(result, "2026年01月20日") {
		t.Error("expected JST date for first diary")
	}
	if !strings.Contains(result, "新しい芽が出ました。") {
		t.Error("expected first diary content")
	}
	if !strings.Contains(result, "葉が大きく成長しています。") {
		t.Error("expected second diary content")
	}
}

func TestExpandPrompt_PastDiariesEmpty(t *testing.T) {
	capturedAt := time.Date(2026, 3, 22, 5, 30, 0, 0, time.UTC)
	result := expandPrompt("before{{past_diaries}}after", "トマト", capturedAt, []domain.Diary{})
	if result != "beforeafter" {
		t.Errorf("expected empty past_diaries to be replaced with empty string, got '%s'", result)
	}
}

func TestExpandPrompt_MultipleOccurrencesOnlyFirstExpanded(t *testing.T) {
	capturedAt := time.Date(2026, 3, 22, 5, 30, 0, 0, time.UTC)
	result := expandPrompt("{{book_name}}と{{book_name}}", "ミニトマト", capturedAt, nil)
	// 最初の1つのみ展開、2つ目は空文字
	if result != "ミニトマトと" {
		t.Errorf("expected only first occurrence expanded, got '%s'", result)
	}
}

func TestExpandPrompt_PlaceholderAbsent(t *testing.T) {
	capturedAt := time.Date(2026, 3, 22, 5, 30, 0, 0, time.UTC)
	pastDiaries := []domain.Diary{
		{ID: 1, Content: "芽が出た。", CreatedAt: capturedAt.AddDate(0, 0, -1)},
	}
	// {{past_diaries}} がない場合、過去日記は埋め込まれない
	result := expandPrompt("シンプルなプロンプト", "トマト", capturedAt, pastDiaries)
	if strings.Contains(result, "芽が出た。") {
		t.Error("expected past diaries not to be embedded when placeholder is absent")
	}
	if result != "シンプルなプロンプト" {
		t.Errorf("expected unchanged template, got '%s'", result)
	}
}

func TestExpandPrompt_ExceedsMaxEntries(t *testing.T) {
	capturedAt := time.Date(2026, 3, 22, 5, 30, 0, 0, time.UTC)
	pastDiaries := make([]domain.Diary, 40)
	baseTime := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	for i := 0; i < 40; i++ {
		pastDiaries[i] = domain.Diary{
			ID:        i + 1,
			Content:   fmt.Sprintf("日記%d番目", i+1),
			CreatedAt: baseTime.AddDate(0, 0, i),
		}
	}

	result := expandPrompt("{{past_diaries}}", "トマト", capturedAt, pastDiaries)

	// 最も古い10件は含まれない
	if strings.Contains(result, "日記1番目") {
		t.Error("expected oldest entry to be excluded")
	}
	if strings.Contains(result, "日記10番目") {
		t.Error("expected old entry to be excluded")
	}
	// 最新の30件は含まれる
	if !strings.Contains(result, "日記11番目") {
		t.Error("expected entry to be included")
	}
	if !strings.Contains(result, "日記40番目") {
		t.Error("expected latest entry to be included")
	}
}

func TestExpandPrompt_ReplacementContainsToken(t *testing.T) {
	capturedAt := time.Date(2026, 3, 22, 5, 30, 0, 0, time.UTC)
	// book_name に {{book_name}} というトークンが含まれる場合でも二重展開されない
	result := expandPrompt("{{book_name}}の記録", "{{book_name}}", capturedAt, nil)
	// 最初の {{book_name}} が "{{book_name}}" に置換され、後続の {{book_name}} は空になる
	if result != "{{book_name}}の記録" {
		t.Errorf("expected replacement token not to be re-expanded, got '%s'", result)
	}
}

func TestDefaultPrompt(t *testing.T) {
	// デフォルトプロンプトに必要なプレースホルダーが含まれているか確認
	if !strings.Contains(defaultPrompt, "{{book_name}}") {
		t.Error("defaultPrompt should contain {{book_name}}")
	}
	if !strings.Contains(defaultPrompt, "{{datetime}}") {
		t.Error("defaultPrompt should contain {{datetime}}")
	}
	if !strings.Contains(defaultPrompt, "{{past_diaries}}") {
		t.Error("defaultPrompt should contain {{past_diaries}}")
	}
}
