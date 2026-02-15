package main

// DiaryGenerator は画像から日記を生成するインターフェース。
type DiaryGenerator interface {
	GenerateDiary(imagePath string) (string, error)
}

// MockDiaryGenerator はテスト用のモック実装。
type MockDiaryGenerator struct{}

func (m *MockDiaryGenerator) GenerateDiary(imagePath string) (string, error) {
	return "この植物は順調に成長しています。葉の色が鮮やかで、新しい芽も見られます。", nil
}
