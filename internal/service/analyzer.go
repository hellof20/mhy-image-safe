package service

import (
	"context"
	"encoding/json"
	"os"

	"example/mhy-image-safe/internal/gemini"
	"example/mhy-image-safe/internal/model"

	"google.golang.org/genai"
)

type ImageAnalyzer struct {
	client *genai.Client
	model  string
}

func NewImageAnalyzer(client *genai.Client, model string) *ImageAnalyzer {
	return &ImageAnalyzer{
		client: client,
		model:  model,
	}
}

func (a *ImageAnalyzer) AnalyzeImage(imagePath string) ([]model.Violation, error) {
	imageBytes, err := os.ReadFile(imagePath)
	if err != nil {
		return nil, err
	}

	parts := []*genai.Part{
		{Text: gemini.GetPrompt()},
		{InlineData: &genai.Blob{Data: imageBytes, MIMEType: "image/jpeg"}},
	}

	result, err := a.client.Models.GenerateContent(
		context.Background(),
		a.model,
		[]*genai.Content{{Parts: parts}},
		gemini.GetConfig())
	if err != nil {
		return nil, err
	}

	text, err := result.Text()
	if err != nil {
		return nil, err
	}
	var violations []model.Violation
	if err := json.Unmarshal([]byte(text), &violations); err != nil {
		return nil, err
	}

	return violations, nil
}
