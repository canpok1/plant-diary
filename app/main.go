package main

import (
	"fmt"
	"log"
)

func main() {
	// インターフェースの統合確認サンプル
	fmt.Println("=== 植物観察日記システム - インターフェース統合確認 ===")

	// モック実装のインスタンス化
	repo := NewMockDiaryRepository()
	generator := &MockDiaryGenerator{}

	// サンプルシナリオ: 画像から日記を生成してDBに保存
	imagePath := "data/photos/20260215_1200.jpg"

	// 1. 画像が既に処理済みか確認
	processed, err := repo.IsImageProcessed(imagePath)
	if err != nil {
		log.Fatalf("IsImageProcessed error: %v", err)
	}
	fmt.Printf("1. 画像処理確認: %s -> 処理済み=%v\n", imagePath, processed)

	// 2. 日記を生成
	content, err := generator.GenerateDiary(imagePath)
	if err != nil {
		log.Fatalf("GenerateDiary error: %v", err)
	}
	fmt.Printf("2. 日記生成: %s\n", content)

	// 3. 日記をDBに保存
	err = repo.CreateDiary(imagePath, content)
	if err != nil {
		log.Fatalf("CreateDiary error: %v", err)
	}
	fmt.Println("3. 日記保存: 成功")

	// 4. 再度処理済みか確認（保存後は true になるはず）
	processed, err = repo.IsImageProcessed(imagePath)
	if err != nil {
		log.Fatalf("IsImageProcessed error: %v", err)
	}
	fmt.Printf("4. 画像処理確認（保存後）: %s -> 処理済み=%v\n", imagePath, processed)

	// 5. 全日記を取得
	diaries, err := repo.GetAllDiaries()
	if err != nil {
		log.Fatalf("GetAllDiaries error: %v", err)
	}
	fmt.Printf("5. 全日記取得: %d件\n", len(diaries))
	for _, d := range diaries {
		fmt.Printf("   - ID=%d, ImagePath=%s, CreatedAt=%s\n", d.ID, d.ImagePath, d.CreatedAt.Format("2006-01-02 15:04:05"))
	}

	// 6. IDで日記を取得
	if len(diaries) > 0 {
		diary, err := repo.GetDiaryByID(diaries[0].ID)
		if err != nil {
			log.Fatalf("GetDiaryByID error: %v", err)
		}
		if diary != nil {
			fmt.Printf("6. ID指定取得: ID=%d, Content=%s\n", diary.ID, diary.Content)
		}
	}

	fmt.Println("\n=== 統合確認完了 ===")
	fmt.Println("✓ DiaryRepository インターフェース: 正常動作")
	fmt.Println("✓ DiaryGenerator インターフェース: 正常動作")
	fmt.Println("✓ モジュール間連携: 問題なし")
}
