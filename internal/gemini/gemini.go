package gemini

import (
	"context"
	"log"

	"google.golang.org/genai"
)

func SetupClient(project, location string) *genai.Client {
	client, err := genai.NewClient(context.Background(), &genai.ClientConfig{
		Project:  project,
		Location: location,
		Backend:  genai.BackendVertexAI,
	})
	if err != nil {
		log.Fatal("Error creating client:", err)
	}
	return client
}

func GetConfig() *genai.GenerateContentConfig {
	responseSchema := &genai.Schema{
		Type: genai.TypeArray,
		Items: &genai.Schema{
			Type: genai.TypeObject,
			Properties: map[string]*genai.Schema{
				"category": {
					Type:        genai.TypeString,
					Description: "违规类别名称",
				},
				"risk_num": {
					Type:        genai.TypeNumber,
					Description: "风险数值 (0-1)",
				},
				"reason": {
					Type:        genai.TypeString,
					Description: "风险评估理由",
				},
			},
			Required: []string{"category", "risk_num", "reason"},
		},
	}
	return &genai.GenerateContentConfig{
		ResponseMIMEType: "application/json",
		ResponseSchema:   responseSchema,
	}
}

// func GetPrompt() string {
// 	return `你是一个专业的二次元游戏论坛图片内容审核员。你的任务是审核用户上传的图片，判断图片内容是否违反论坛社区规定。请根据以下违规类别，分析图片内容，并给出该图片在每个类别下的风险数值和理由。

// 请特别注意以下几点：

// * **二次元游戏属性**: 审核标准需要结合二次元游戏的特点。例如，适度的角色暴露在某些二次元游戏中可能是被接受的，但需要区分是否过度或带有性暗示。对于游戏内的角色和场景，应适当放宽尺度，但对于真人照片或明显超出游戏范畴的内容，应严格审核。
// * **风险数值**:  违规风险的大小，数值范围为 0 到 1的两位小数。

// **请审核以下违规类别，并为每个类别输出风险数值和理由：**

// Account Trade (账号交易): 图片是否包含任何账号交易的信息，例如买卖账号的截图、联系方式、平台等？
// Scams and Advertisements (诈骗与广告): 图片是否包含任何诈骗信息，例如虚假充值、外挂销售、恶意链接，或者未经允许的广告宣传内容（非官方广告）？
// Information Leakage (信息泄露): 图片是否泄露了用户的个人信息、游戏账号信息、未公开的游戏内部信息（例如测试服截图、未公开的角色/剧情）？
// Verbal Abuse and Threats (言语辱骂与威胁): 图片是否包含针对其他用户的辱骂、诽谤、人身攻击、威胁等内容？
// Sexuality and Nudity (性暗示与裸露): 图片是否包含过度的性暗示、裸露、色情内容？是否描绘性器官、性行为，或者具有强烈的性暗示意味？ **（请结合二次元游戏尺度判断，区分适度福利与过度色情）**
// Minor Safety (未成年人保护): 图片是否包含任何可能危害未成年人身心健康的内容？例如，诱导未成年人进行不良行为、过度暴露未成年人身体、儿童色情内容（即使是二次元也严格禁止）？
// Illegal Activities and Regulated Goods (非法活动与管制物品): 图片是否包含任何非法活动信息，例如毒品、赌博、枪支、非法集会、传销等？是否涉及管制刀具、易燃易爆物品等？
// Religious and Political Content (宗教与政治内容): 图片是否包含任何煽动宗教矛盾、宣扬极端宗教思想、违反国家政策的政治敏感内容？ **（请注意，轻微的游戏内宗教元素一般不视为违规，重点判断是否涉及现实政治和极端宗教）**
// Personal Identifiable Information (个人身份信息): 图片是否包含用户的真实姓名、身份证号、住址、电话号码等个人身份信息？ **（即使是用户自愿公开，也需要评估风险并建议用户撤回）**
// Violent Extremism (暴力极端主义): 图片是否宣扬暴力、恐怖主义、极端主义思想？是否美化暴力行为或恐怖组织？
// Suicide and Self-Harm (自杀与自残): 图片是否包含鼓励、煽动自杀或自残行为的内容？是否详细描述自杀或自残的方式？

// **请分析用户上传的图片，并以 JSON 格式返回每个类别的风险级别和理由。 列表中每个元素对应一个违规类别，包含 "category" (类别名称), "risk_num" (风险数值)和 "reason" (理由) 三个字段。**
// **理由用中文输出。**`
// }

func GetPrompt() string {
	return `你是一个专业的二次元游戏论坛图片内容审核员。你的任务是审核用户上传的图片，判断图片内容是否违反论坛社区规定。

风险等级划分：
0-0.2: 安全，基本无风险
0.21-0.4: 轻微风险，需要关注
0.41-0.6: 中等风险，建议修改
0.61-0.8: 高风险，可能需要删除
0.81-1.0: 严重违规，必须删除

审核时需考虑以下上下文因素：
二次元游戏属性：审核标准需结合二次元游戏特点，适度的角色暴露在某些二次元游戏中可能被接受，但需区分是否过度或带有性暗示。

请审核以下违规类别：
Sexuality and Nudity (性暗示与裸露)
检查重点：
角色服装暴露程度
姿势是否带有性暗示
内容是否涉及性相关元素

Religious and Political Content (宗教与政治内容)
检查重点：
宗教符号与内容
政治相关话题与符号
意识形态倾向

Negative and Exclusive Content (消极与排他内容)
检查重点：
人身攻击与辱骂
歧视性言论
负面情绪引导

Illegal Activities and Regulated Goods (非法活动与管制物品)
检查重点：
非法物品展示
违禁品交易信息
非法活动相关内容

Graphic and Disturbing Content (令人不适的图像内容)
检查重点：
暴力血腥内容
令人不适的场景
自残/伤害相关

以 JSON 格式返回每个类别的风险数值和理由。 列表中每个元素对应一个违规类别，包含 "category" (类别名称), "risk_num" (风险数值)和 "reason" (理由) 三个字段。理由用中文输出。`
}
