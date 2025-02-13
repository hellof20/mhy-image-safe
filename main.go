package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"

	"github.com/joho/godotenv"
	"golang.org/x/sync/errgroup"
	"google.golang.org/genai"
)

type Result struct {
	Category string `json:"category"`
	Severity string `json:"severity"`
	Reason   string `json:"reason"`
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

			fmt.Println(resp)
			// 写入结果
			//imageName := filepath.Base(imagePath)
			// 创建一个用于写入结果的互斥锁
			// var mu sync.Mutex
			// mu.Lock()
			// _, err = fmt.Fprintf(outputFile, "%s|%s|%s|%s|%s\n",
			// 	time.Now().Format(time.RFC3339),

			// 	imageName,
			// 	resp.Category,
			// 	resp.Severity,
			// 	resp.Reason,
			// )
			// mu.Unlock()

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

func (c *Config) analyzeImage(client *genai.Client, imagePath string) (string, error) {
	imageBytes, err := os.ReadFile(imagePath)
	if err != nil {
		return "", err
	}

	parts := []*genai.Part{
		{Text: getPrompt()},
		{InlineData: &genai.Blob{Data: imageBytes, MIMEType: "image/jpeg"}},
	}

	result, err := client.Models.GenerateContent(context.Background(), c.model, []*genai.Content{{Parts: parts}}, getConfig())
	if err != nil {
		return "", err
	}

	text, err := result.Text()
	if err != nil {
		return "", err
	}

	// var resp Result
	// if err := json.Unmarshal([]byte(text), &resp); err != nil {
	// 	return "", err
	// }

	return text, nil
}

func isImageFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".jpg" || ext == ".jpeg" || ext == ".png" || ext == ".gif" || ext == ".webp"
}

func getPrompt() string {
	return `你是一个专业的二次元游戏论坛图片内容审核员。你的任务是审核用户上传的图片，判断图片内容是否违反论坛社区规定。请根据以下违规类别，分析图片内容，并给出该图片在每个类别下的违规风险级别（低、中、高、无）和理由。

请特别注意以下几点：

* **二次元游戏属性**: 审核标准需要结合二次元游戏的特点。例如，适度的角色暴露在某些二次元游戏中可能是被接受的，但需要区分是否过度或带有性暗示。对于游戏内的角色和场景，应适当放宽尺度，但对于真人照片或明显超出游戏范畴的内容，应严格审核。
* **风险级别**:  违规风险级别分为“低”、“中”、“高”、“无”四个等级。
    * **无**:  图片内容完全不涉及该违规类别。
    * **低**:  图片内容可能存在轻微擦边或暗示，但整体不明显，不构成严重违规，可以接受但需要关注。
    * **中**:  图片内容较为明显地涉及违规类别，需要警告或轻度处理。
    * **高**:  图片内容严重违反规定，必须立即删除并对用户进行处罚。

**请审核以下违规类别，并为每个类别输出风险级别和理由：**

1. **Account Trade (账号交易)**: 图片是否包含任何账号交易的信息，例如买卖账号的截图、联系方式、平台等？
2. **Scams and Advertisements (诈骗与广告)**: 图片是否包含任何诈骗信息，例如虚假充值、外挂销售、恶意链接，或者未经允许的广告宣传内容（非官方广告）？
3. **Information Leakage (信息泄露)**: 图片是否泄露了用户的个人信息、游戏账号信息、未公开的游戏内部信息（例如测试服截图、未公开的角色/剧情）？
4. **Verbal Abuse and Threats (言语辱骂与威胁)**: 图片是否包含针对其他用户的辱骂、诽谤、人身攻击、威胁等内容？
5. **Sexuality and Nudity (性暗示与裸露)**: 图片是否包含过度的性暗示、裸露、色情内容？是否描绘性器官、性行为，或者具有强烈的性暗示意味？ **（请结合二次元游戏尺度判断，区分适度福利与过度色情）**
6. **Minor Safety (未成年人保护)**: 图片是否包含任何可能危害未成年人身心健康的内容？例如，诱导未成年人进行不良行为、过度暴露未成年人身体、儿童色情内容（即使是二次元也严格禁止）？
7. **Illegal Activities and Regulated Goods (非法活动与管制物品)**: 图片是否包含任何非法活动信息，例如毒品、赌博、枪支、非法集会、传销等？是否涉及管制刀具、易燃易爆物品等？
8. **Religious and Political Content (宗教与政治内容)**: 图片是否包含任何煽动宗教矛盾、宣扬极端宗教思想、违反国家政策的政治敏感内容？ **（请注意，轻微的游戏内宗教元素一般不视为违规，重点判断是否涉及现实政治和极端宗教）**
9. **Personal Identifiable Information (个人身份信息)**: 图片是否包含用户的真实姓名、身份证号、住址、电话号码等个人身份信息？ **（即使是用户自愿公开，也需要评估风险并建议用户撤回）**
10. **Violent Extremism (暴力极端主义)**: 图片是否宣扬暴力、恐怖主义、极端主义思想？是否美化暴力行为或恐怖组织？
11. **Suicide and Self-Harm (自杀与自残)**: 图片是否包含鼓励、煽动自杀或自残行为的内容？是否详细描述自杀或自残的方式？

**请分析用户上传的图片，并以 JSON 格式返回每个类别的风险级别和理由。 JSON 格式应包含一个主键 "violations"，其值为一个列表，列表中每个元素对应一个违规类别，包含 "category" (类别名称), "risk_level" (风险级别), 和 "reason" (理由) 三个字段。**`
}

func getConfig() *genai.GenerateContentConfig {
	// responseSchema := &genai.Schema{
	// 	Type: genai.TypeObject,
	// 	Properties: map[string]*genai.Schema{
	// 		"category": {
	// 			Type: genai.TypeString,
	// 			Enum: []string{
	// 				"Account Trade",
	// 				"Scams and Advertisements",
	// 				"Information Leakage",
	// 				"Verbal Abuse and Threats",
	// 				"Sexuality and Nudity",
	// 				"Minor Safety",
	// 				"Illegal Activities and Regulated Goods",
	// 				"Religious and Political Content",
	// 				"Personal Identifiable Information",
	// 				"Violent Extremism",
	// 				"Suicide and Self-Harm",
	// 				"pass",
	// 			},
	// 		},
	// 		"severity": {Type: genai.TypeString, Nullable: false, Enum: []string{"Null", "Low", "Medium", "High"}},
	// 		"reason":   {Type: genai.TypeString},
	// 	},
	// }
	return &genai.GenerateContentConfig{}
}
