package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"google.golang.org/genai"
)

const (
	geminiModel   = "gemini-2.5-flash"
	geminiTimeout = 30 * time.Second
	geminiPrompt  = "この植物の写真を見て、成長の様子や変化を観察してください。親しみやすい口調で、200文字程度の観察日記を書いてください。"
)

// GeminiDiaryGenerator は Gemini API を使って画像から日記を生成する。
type GeminiDiaryGenerator struct {
	apiKey string
}

// NewGeminiDiaryGenerator は環境変数 GEMINI_API_KEY から API キーを取得して
// GeminiDiaryGenerator を生成する。
func NewGeminiDiaryGenerator() (*GeminiDiaryGenerator, error) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("環境変数 GEMINI_API_KEY が設定されていません")
	}
	return &GeminiDiaryGenerator{apiKey: apiKey}, nil
}

// GenerateDiary は画像ファイルを読み込み、Gemini API で観察日記を生成する。
func (g *GeminiDiaryGenerator) GenerateDiary(imagePath string) (string, error) {
	imageBytes, err := os.ReadFile(imagePath)
	if err != nil {
		return "", fmt.Errorf("画像ファイルの読み込みに失敗: %w", err)
	}

	mimeType := http.DetectContentType(imageBytes)

	ctx, cancel := context.WithTimeout(context.Background(), geminiTimeout)
	defer cancel()

	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  g.apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return "", fmt.Errorf("Gemini クライアントの作成に失敗: %w", err)
	}

	parts := []*genai.Part{
		{Text: geminiPrompt},
		{InlineData: &genai.Blob{
			Data:     imageBytes,
			MIMEType: mimeType,
		}},
	}
	contents := []*genai.Content{
		{Parts: parts, Role: "user"},
	}

	resp, err := client.Models.GenerateContent(ctx, geminiModel, contents, nil)
	if err != nil {
		return "", fmt.Errorf("Gemini API の呼び出しに失敗: %w", err)
	}

	text := resp.Text()
	if text == "" {
		return "", fmt.Errorf("Gemini API から空のレスポンスが返されました")
	}

	return text, nil
}
