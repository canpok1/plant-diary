package main

import (
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
