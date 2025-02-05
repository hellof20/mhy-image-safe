package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/joho/godotenv"
	"golang.org/x/sync/errgroup"
	"google.golang.org/genai"
)

type Result struct {
	Type   string `json:"type"`
	Reason string `json:"reason"`
}

type Config struct {
	location    string
	model       string
	project     string
	temperature float64
	target      string
	concurrent  int
	client      *genai.Client
}

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}

	concurrent, _ := strconv.Atoi(os.Getenv("CONCURRENT"))
	temperature, _ := strconv.ParseFloat(os.Getenv("TEMPERATURE"), 64)
	imageDir := os.Getenv("IMAGE_DIR")

	if imageDir == "" {
		log.Fatal("image directory cannot be empty")
	}

	cfg := &Config{
		location:    os.Getenv("LOCATION"),
		model:       os.Getenv("MODEL"),
		project:     os.Getenv("PROJECT"),
		target:      os.Getenv("TARGET"),
		temperature: temperature,
		concurrent:  concurrent,
	}

	// 初始化客户端
	client := cfg.setupClient()
	cfg.client = client

	if err := cfg.processImages(imageDir); err != nil {
		log.Fatal(err)
	}
	log.Println("BatchInvoke completed successfully.")
}

func (c *Config) processImages(imageDir string) error {
	// 打开输出文件
	outputFile, err := os.OpenFile(c.target, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("opening output file: %w", err)
	}
	defer outputFile.Close()

	// 收集所有图片路径
	var images []string
	err = filepath.Walk(imageDir, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() && isImageFile(path) {
			images = append(images, path)
		}
		return nil
	})
	if err != nil {
		return err
	}

	// 创建一个用于写入结果的互斥锁
	var mu sync.Mutex

	// 使用 errgroup 来管理并发和错误处理
	g := new(errgroup.Group)
	g.SetLimit(c.concurrent) // 限制并发数

	// 添加错误计数
	var errorCount int32

	// 并发处理每张图片
	for _, imagePath := range images {
		imagePath := imagePath // 创建副本避免闭包问题
		g.Go(func() error {
			log.Printf("Processing %s", imagePath)
			resp, err := c.analyzeImage(c.client, imagePath)
			if err != nil {
				atomic.AddInt32(&errorCount, 1)
				log.Printf("Error processing %s: %v", imagePath, err)
				return nil
			}

			// 写入结果
			mu.Lock()
			_, err = fmt.Fprintf(outputFile, "%s,%s,%s,%s\n",
				time.Now().Format(time.RFC3339),

				imagePath,
				resp.Type,
				resp.Reason)
			mu.Unlock()

			return err
		})
	}

	if err := g.Wait(); err != nil {
		return err
	}

	// 输出统计信息
	log.Printf("处理完成: 总计 %d 张图片, 失败 %d 张", len(images), errorCount)
	log.Printf("Results written to %s", c.target)
	return nil
}

func (c *Config) setupClient() *genai.Client {
	client, err := genai.NewClient(context.Background(), &genai.ClientConfig{
		Project:  c.project,
		Location: c.location,
		Backend:  genai.BackendVertexAI,
	})
	if err != nil {
		log.Fatal("Error creating client:", err)
	}
	return client
}

func (c *Config) analyzeImage(client *genai.Client, imagePath string) (*Result, error) {
	imageBytes, err := os.ReadFile(imagePath)
	if err != nil {
		return nil, err
	}

	parts := []*genai.Part{
		{Text: getPrompt()},
		{InlineData: &genai.Blob{Data: imageBytes, MIMEType: "image/jpeg"}},
	}

	result, err := client.Models.GenerateContent(context.Background(), c.model, []*genai.Content{{Parts: parts}}, getConfig())
	if err != nil {
		return nil, err
	}

	text, err := result.Text()
	if err != nil {
		return nil, err
	}

	var resp Result
	if err := json.Unmarshal([]byte(text), &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

func isImageFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".jpg" || ext == ".jpeg" || ext == ".png" || ext == ".gif" || ext == ".webp"
}

func getPrompt() string {
	return `<Role>
你是图片内容安全审核专家
</Role>

<Task>
审核图片是否涉及哪个类别，并给出归属到该类别的原因

候选类别:
色情
性暗示
血腥
爆炸
政治
武器
恐怖
广告logo
辱骂
</Task>

<requirement>
输出为中文
不包含上述类别则为安全
</requirement>`
}

func getConfig() *genai.GenerateContentConfig {
	return &genai.GenerateContentConfig{
		ResponseMIMEType: "application/json",
		ResponseSchema: &genai.Schema{
			Type: genai.TypeObject,
			Properties: map[string]*genai.Schema{
				"type":   {Type: genai.TypeString},
				"reason": {Type: genai.TypeString},
			},
		},
	}
}
