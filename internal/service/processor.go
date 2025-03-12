package service

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/sync/errgroup"
)

type ImageProcessor struct {
	analyzer   *ImageAnalyzer
	concurrent int
}

func NewImageProcessor(analyzer *ImageAnalyzer, concurrent int) *ImageProcessor {
	return &ImageProcessor{
		analyzer:   analyzer,
		concurrent: concurrent,
	}
}

func (p *ImageProcessor) ProcessImages(imageDir string) error {
	images, err := p.collectImages(imageDir)
	if err != nil {
		return err
	}

	g := new(errgroup.Group)
	g.SetLimit(p.concurrent)

	var errorCount int32
	var processedCount int32
	totalCount := len(images)

	// 创建一个用于显示进度的 ticker
	ticker := time.NewTicker(4 * time.Second)
	defer ticker.Stop()

	// 在后台显示进度
	go func() {
		for range ticker.C {
			processed := atomic.LoadInt32(&processedCount)
			if processed < int32(totalCount) {
				log.Printf("进度: %d/%d (%.1f%%)", processed, totalCount, float64(processed)/float64(totalCount)*100)
			}
		}
	}()

	for _, imagePath := range images {
		imagePath := imagePath
		g.Go(func() error {
			err := p.processImage(imagePath, &errorCount)
			atomic.AddInt32(&processedCount, 1)
			return err
		})
	}

	if err := g.Wait(); err != nil {
		return err
	}

	log.Printf("处理完成: 总计 %d 张图片, 失败 %d 张", totalCount, errorCount)
	return nil
}

func (p *ImageProcessor) processImage(imagePath string, errorCount *int32) error {
	// log.Printf("Processing %s", imagePath)
	violations, err := p.analyzer.AnalyzeImage(imagePath)
	if err != nil {
		atomic.AddInt32(errorCount, 1)
		log.Printf("Error processing %s: %v", imagePath, err)
		return nil
	}

	// 将结果写入文件
	outputFile, err := os.OpenFile("output.csv", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("opening output file: %w", err)
	}
	defer outputFile.Close()
	imageName := filepath.Base(imagePath)
	//创建一个用于写入结果的互斥锁
	var mu sync.Mutex
	mu.Lock()
	for _, violation := range violations {
		if violation.RiskNum > 0 {
			_, err = fmt.Fprintf(outputFile, "%s|%s|%.2f|%s\n",
				imageName, violation.Category, violation.RiskNum, violation.Reason,
			)
			if err != nil {
				log.Printf("Error writing result for %s: %v", imageName, err)
			}
		}
	}
	mu.Unlock()
	return nil
}

func (p *ImageProcessor) collectImages(dir string) ([]string, error) {
	var images []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() && isImageFile(path) {
			images = append(images, path)
		}
		return nil
	})
	return images, err
}

func isImageFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".jpg" || ext == ".jpeg" || ext == ".png" || ext == ".gif" || ext == ".webp"
}
