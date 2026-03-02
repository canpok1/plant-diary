package usecase

// DiaryGenerator は画像から日記を生成するインターフェース。
type DiaryGenerator interface {
	GenerateDiary(imagePath string) (string, error)
}

// DiaryGeneratorWithPrompt は動的プロンプトをサポートするインターフェース。
type DiaryGeneratorWithPrompt interface {
	DiaryGenerator
	GenerateDiaryWithPrompt(imagePath string, prompt string) (string, error)
}
