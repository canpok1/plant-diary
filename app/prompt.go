package main

import (
	"fmt"
	"strings"
	"time"
)

const (
	basePrompt             = "この植物の写真を見て、成長の様子や変化を観察してください。親しみやすい口調で、200文字程度の観察日記を書いてください。"
	maxPastDiariesInPrompt = 30 // プロンプトに含める過去日記の最大件数
)

// buildDiaryPrompt は過去日記を含む動的プロンプトを生成する。
func buildDiaryPrompt(pastDiaries []Diary) string {
	if len(pastDiaries) == 0 {
		return basePrompt
	}

	// プロンプトに含める日記を最新のN件に制限（古い順にソートされているため、最後のN件を取得）
	diariesToInclude := pastDiaries
	if len(pastDiaries) > maxPastDiariesInPrompt {
		diariesToInclude = pastDiaries[len(pastDiaries)-maxPastDiariesInPrompt:]
	}

	// JSTに変換するためのヘルパー
	jst := time.FixedZone("Asia/Tokyo", 9*60*60)

	var builder strings.Builder
	builder.WriteString(basePrompt)
	builder.WriteString("\n\n参考までに、過去1ヶ月の観察記録を以下に示します：\n\n")

	for _, diary := range diariesToInclude {
		jstTime := diary.CreatedAt.In(jst)
		fmt.Fprintf(&builder, "【%s】\n%s\n\n", jstTime.Format("2006年01月02日"), diary.Content)
	}

	builder.WriteString("これまでの観察記録を踏まえて、今回の写真から見られる成長の変化や特徴を記述してください。")

	return builder.String()
}
