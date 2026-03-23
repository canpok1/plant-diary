package handler

import (
	"fmt"
	"strings"
	"time"

	"plant-diary/internal/domain"
)

const (
	defaultPrompt = domain.DefaultBookPrompt

	maxPastDiariesInPrompt = 30 // プロンプトに含める過去日記の最大件数
)

// expandPrompt はプロンプトテンプレート内のプレースホルダーを展開する。
// 各プレースホルダーは最初の1箇所のみ展開され、残りは空文字に置換される。
func expandPrompt(promptTemplate string, bookName string, capturedAt time.Time, pastDiaries []domain.Diary) string {
	result := promptTemplate

	result = replaceFirstThenEmpty(result, "{{book_name}}", bookName)
	result = replaceFirstThenEmpty(result, "{{datetime}}", formatDatetime(capturedAt))
	result = replaceFirstThenEmpty(result, "{{past_diaries}}", buildPastDiariesText(pastDiaries))

	return result
}

// replaceFirstThenEmpty は文字列中の最初の old を replacement に置換し、残りを空文字に置換する。
// replacement 自体に old と同じ文字列が含まれていても安全に処理される。
func replaceFirstThenEmpty(s, old, replacement string) string {
	idx := strings.Index(s, old)
	if idx == -1 {
		return s
	}
	return s[:idx] + replacement + strings.ReplaceAll(s[idx+len(old):], old, "")
}

// formatDatetime はJSTの日時を「2006年01月02日 15時04分 (JST)」形式で返す。
func formatDatetime(t time.Time) string {
	return t.In(jst).Format("2006年01月02日 15時04分 (JST)")
}

// buildPastDiariesText は過去日記テキストを構築する。
// 日記が存在しない場合は空文字を返す。
func buildPastDiariesText(pastDiaries []domain.Diary) string {
	if len(pastDiaries) == 0 {
		return ""
	}

	// プロンプトに含める日記を最新のN件に制限（古い順にソートされているため、最後のN件を取得）
	diariesToInclude := pastDiaries
	if len(pastDiaries) > maxPastDiariesInPrompt {
		diariesToInclude = pastDiaries[len(pastDiaries)-maxPastDiariesInPrompt:]
	}

	var builder strings.Builder
	builder.WriteString("参考までに、過去1ヶ月の観察記録を以下に示します：\n\n")

	for _, diary := range diariesToInclude {
		jstTime := diary.CreatedAt.In(jst)
		fmt.Fprintf(&builder, "【%s】\n%s\n\n", jstTime.Format("2006年01月02日"), diary.Content)
	}

	return builder.String()
}
