package main

// DiaryGenerator は画像から日記を生成するインターフェース。
type DiaryGenerator interface {
	GenerateDiary(imagePath string) (string, error)
}

// DiaryGeneratorWithPrompt は動的プロンプトをサポートするインターフェース。
type DiaryGeneratorWithPrompt interface {
	DiaryGenerator
	GenerateDiaryWithPrompt(imagePath string, prompt string) (string, error)
}

// MockDiaryGenerator はテスト用のモック実装。
type MockDiaryGenerator struct{}

func (m *MockDiaryGenerator) GenerateDiary(imagePath string) (string, error) {
	return "この植物は順調に成長しています。葉の色が鮮やかで、新しい芽も見られます。", nil
}

func (m *MockDiaryGenerator) GenerateDiaryWithPrompt(imagePath string, prompt string) (string, error) {
	// プロンプトは無視して固定文字列を返す
	return "この植物は順調に成長しています。葉の色が鮮やかで、新しい芽も見られます。", nil
}
