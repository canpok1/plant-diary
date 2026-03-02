package gemini

// MockDiaryGenerator はテスト用のDiaryGenerator実装
type MockDiaryGenerator struct{}

// GenerateDiary は固定の日記テキストを返す
func (m *MockDiaryGenerator) GenerateDiary(imagePath string) (string, error) {
	return "この植物は順調に成長しています。葉の色が鮮やかで、新しい芽も見られます。", nil
}
