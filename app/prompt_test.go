package main

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestBuildDiaryPrompt_NoPastDiaries(t *testing.T) {
	pastDiaries := []Diary{}

	prompt := buildDiaryPrompt(pastDiaries)

	// 過去日記がない場合は基本プロンプトのみが返される
	if prompt != basePrompt {
		t.Errorf("expected base prompt only, got '%s'", prompt)
	}
}

func TestBuildDiaryPrompt_WithPastDiaries(t *testing.T) {
	// 過去日記を作成（UTC）
	// UTC時刻なので、JSTに変換すると+9時間される
	time1 := time.Date(2026, 1, 20, 11, 10, 0, 0, time.UTC) // JST: 2026-01-20 20:10
	time2 := time.Date(2026, 2, 1, 10, 30, 0, 0, time.UTC)  // JST: 2026-02-01 19:30

	pastDiaries := []Diary{
		{ID: 1, ImagePath: "/path/1.jpg", Content: "新しい芽が出ました。小さいですが、元気そうです。", CreatedAt: time1},
		{ID: 2, ImagePath: "/path/2.jpg", Content: "葉が大きく成長しています。前回よりも色が濃くなりました。", CreatedAt: time2},
	}

	prompt := buildDiaryPrompt(pastDiaries)

	// 基本プロンプトが含まれることを確認
	if !strings.Contains(prompt, basePrompt) {
		t.Error("expected prompt to contain base prompt")
	}

	// 過去日記の説明が含まれることを確認
	if !strings.Contains(prompt, "過去1ヶ月の観察記録") {
		t.Error("expected prompt to contain past diary description")
	}

	// JST形式の日付が含まれることを確認
	if !strings.Contains(prompt, "2026年01月20日") {
		t.Error("expected prompt to contain JST date for first diary")
	}

	if !strings.Contains(prompt, "2026年02月01日") {
		t.Error("expected prompt to contain JST date for second diary")
	}

	// 過去日記の内容が含まれることを確認
	if !strings.Contains(prompt, "新しい芽が出ました。小さいですが、元気そうです。") {
		t.Error("expected prompt to contain first diary content")
	}

	if !strings.Contains(prompt, "葉が大きく成長しています。前回よりも色が濃くなりました。") {
		t.Error("expected prompt to contain second diary content")
	}

	// 指示文が含まれることを確認
	if !strings.Contains(prompt, "これまでの観察記録を踏まえて") {
		t.Error("expected prompt to contain instruction text")
	}
}

func TestBuildDiaryPrompt_ExceedsMaxEntries(t *testing.T) {
	// 40件の過去日記を作成（maxPastDiariesInPromptを超える）
	pastDiaries := make([]Diary, 40)
	baseTime := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	for i := 0; i < 40; i++ {
		pastDiaries[i] = Diary{
			ID:        i + 1,
			ImagePath: fmt.Sprintf("/path/%d.jpg", i+1),
			Content:   fmt.Sprintf("日記%d番目", i+1),
			CreatedAt: baseTime.AddDate(0, 0, i),
		}
	}

	prompt := buildDiaryPrompt(pastDiaries)

	// 最新の30件のみが含まれることを確認
	// 最も古い10件（日記1番目〜日記10番目）は含まれないはず
	if strings.Contains(prompt, "日記1番目") {
		t.Error("expected oldest entry (日記1番目) to be excluded")
	}
	if strings.Contains(prompt, "日記10番目") {
		t.Error("expected old entry (日記10番目) to be excluded")
	}

	// 最新の30件（日記11番目〜日記40番目）は含まれるはず
	if !strings.Contains(prompt, "日記11番目") {
		t.Error("expected entry (日記11番目) to be included")
	}
	if !strings.Contains(prompt, "日記40番目") {
		t.Error("expected latest entry (日記40番目) to be included")
	}
}
