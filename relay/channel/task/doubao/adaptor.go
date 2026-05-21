package doubao

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relay/channel"
	"github.com/QuantumNous/new-api/relay/channel/task/taskcommon"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/service"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"github.com/samber/lo"
)

// ============================
// Request / Response structures
// ============================

type ContentItem struct {
	Type     string    `json:"type,omitempty"`
	Text     string    `json:"text,omitempty"`
	ImageURL *MediaURL `json:"image_url,omitempty"`
	VideoURL *MediaURL `json:"video_url,omitempty"`
	AudioURL *MediaURL `json:"audio_url,omitempty"`
	Role     string    `json:"role,omitempty"`
}

type MediaURL struct {
	URL string `json:"url,omitempty"`
}

type requestPayload struct {
	Model                 string         `json:"model"`
	Content               []ContentItem  `json:"content,omitempty"`
	CallbackURL           string         `json:"callback_url,omitempty"`
	ReturnLastFrame       *dto.BoolValue `json:"return_last_frame,omitempty"`
	ServiceTier           string         `json:"service_tier,omitempty"`
	ExecutionExpiresAfter *dto.IntValue  `json:"execution_expires_after,omitempty"`
	GenerateAudio         *dto.BoolValue `json:"generate_audio,omitempty"`
	Draft                 *dto.BoolValue `json:"draft,omitempty"`
	Tools                 []struct {
		Type string `json:"type,omitempty"`
	} `json:"tools,omitempty"`
	Resolution  string         `json:"resolution,omitempty"`
	Ratio       string         `json:"ratio,omitempty"`
	Duration    *dto.IntValue  `json:"duration,omitempty"`
	Frames      *dto.IntValue  `json:"frames,omitempty"`
	Seed        *dto.IntValue  `json:"seed,omitempty"`
	CameraFixed *dto.BoolValue `json:"camera_fixed,omitempty"`
	Watermark   *dto.BoolValue `json:"watermark,omitempty"`
}

type responsePayload struct {
	ID string `json:"id"` // task_id
}

type responseTask struct {
	ID      string `json:"id"`
	Model   string `json:"model"`
	Status  string `json:"status"`
	Content struct {
		VideoURL string `json:"video_url"`
	} `json:"content"`
	Seed            int    `json:"seed"`
	Resolution      string `json:"resolution"`
	Duration        int    `json:"duration"`
	Ratio           string `json:"ratio"`
	FramesPerSecond int    `json:"framespersecond"`
	ServiceTier     string `json:"service_tier"`
	Tools           []struct {
		Type string `json:"type"`
	} `json:"tools"`
	Usage struct {
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
		ToolUsage        struct {
			WebSearch int `json:"web_search"`
		} `json:"tool_usage"`
	} `json:"usage"`
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
	CreatedAt int64 `json:"created_at"`
	UpdatedAt int64 `json:"updated_at"`
}

// ============================
// Adaptor implementation
// ============================

type TaskAdaptor struct {
	taskcommon.BaseBilling
	ChannelType int
	apiKey      string
	baseURL     string
}

func (a *TaskAdaptor) Init(info *relaycommon.RelayInfo) {
	a.ChannelType = info.ChannelType
	a.baseURL = info.ChannelBaseUrl
	a.apiKey = info.ApiKey
}

// ValidateRequestAndSetAction parses body, validates fields and sets default action.
func (a *TaskAdaptor) ValidateRequestAndSetAction(c *gin.Context, info *relaycommon.RelayInfo) (taskErr *dto.TaskError) {
	if info.RelayMode == relayconstant.RelayModeSeedanceSubmit || isSeedance2Model(info.OriginModelName) {
		return relaycommon.ValidateSeedanceTaskRequest(c, info)
	}
	// Accept only POST /v1/video/generations as "generate" action.
	return relaycommon.ValidateBasicTaskRequest(c, info, constant.TaskActionGenerate)
}

// BuildRequestURL constructs the upstream URL.
func (a *TaskAdaptor) BuildRequestURL(_ *relaycommon.RelayInfo) (string, error) {
	return fmt.Sprintf("%s/api/v3/contents/generations/tasks", a.baseURL), nil
}

// BuildRequestHeader sets required headers.
func (a *TaskAdaptor) BuildRequestHeader(_ *gin.Context, req *http.Request, _ *relaycommon.RelayInfo) error {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+a.apiKey)
	return nil
}

// isSeedance2Model 判断是否为 doubao-seedance-2 系列模型。
func isSeedance2Model(modelName string) bool {
	return strings.HasPrefix(modelName, "doubao-seedance-2")
}

// EstimateBilling 检测视频输入和分辨率，返回预扣倍率。
// Seedance 2 系列模型走 token 预估计费，其余 doubao 视频走四档分辨率倍率。
func (a *TaskAdaptor) EstimateBilling(c *gin.Context, info *relaycommon.RelayInfo) map[string]float64 {
	if info.RelayMode == relayconstant.RelayModeSeedanceSubmit || isSeedance2Model(info.OriginModelName) {
		return estimateSeedanceBilling(c, info)
	}
	// 标准格式路径：仅检测 metadata 中的 video_url（不含分辨率信息）
	req, err := relaycommon.GetTaskRequest(c)
	if err != nil {
		return nil
	}
	if hasVideoInMetadata(req.Metadata) {
		ratioMap, ok := GetResolutionVideoRatio(info.OriginModelName)
		if !ok {
			return nil
		}
		if ratio, ok := ratioMap["low_video"]; ok {
			return map[string]float64{"video_resolution_input": ratio}
		}
	}
	return nil
}

// estimateSeedanceBilling 预扣费，直接使用官方分档价格（¥/百万tokens）：
//
//	quota = tokens / 1,000,000 × tierPrice × QuotaPerUnit × groupRatio
//
// tierPrice 来自 seedanceTierPriceMap（如 480p/720p 无视频 = 46 元/百万tokens）。
// 覆盖 info.PriceData.ModelPrice 为 tierPrice，供 BillingContext 记录并在结算时使用。
//
// 不再返回 OtherRatios，不经过 step 6 乘算，直接设置 info.PriceData.Quota。
func estimateSeedanceBilling(c *gin.Context, info *relaycommon.RelayInfo) map[string]float64 {
	req, err := relaycommon.GetSeedanceTaskRequest(c)
	if err != nil {
		return nil
	}

	// ---- 1) 计算预估 token ----
	resolution := req.Resolution
	if resolution == "" {
		resolution = "720p"
	}
	aspectRatio := req.Ratio
	if aspectRatio == "" {
		aspectRatio = "16:9"
	}
	duration := 4
	if req.Duration != nil && int(*req.Duration) > 0 {
		duration = int(*req.Duration)
	}
	fps := 24
	if req.Frames != nil && int(*req.Frames) > 0 {
		fps = int(*req.Frames)
	}

	width, height := resolveSeedanceDimensions(resolution, aspectRatio)
	estTokens := (width * height * fps * duration) / 1024
	if estTokens <= 0 {
		return nil
	}

	// ---- 2) 直接获取分档价格（¥/百万tokens） ----
	hasVideo := req.HasVideo()
	tier := resolveResolutionTier(resolution)
	key := tierKey(hasVideo, tier)

	tierPrice := 0.0
	if priceMap, ok := GetSeedanceTierPrice(info.OriginModelName); ok {
		if p, ok := priceMap[key]; ok {
			tierPrice = p
		}
	}
	if tierPrice <= 0 {
		// fallback: 使用 basePrice
		tierPrice = info.PriceData.ModelPrice
		if tierPrice <= 0 {
			tierPrice = info.PriceData.ModelRatio * 2
		}
	}
	if tierPrice <= 0 {
		return nil
	}

	// ---- 3) 官方计费公式 ----
	quota := int(float64(estTokens) / 1_000_000.0 * tierPrice * common.QuotaPerUnit *
		info.PriceData.GroupRatioInfo.GroupRatio)
	if quota <= 0 {
		return nil
	}

	info.PriceData.Quota = quota
	// 覆盖 ModelPrice 为分档价格，BillingContext 将携带此值供结算时使用
	info.PriceData.ModelPrice = tierPrice

	return nil
}

// -- Seedance 分辨率 / 比例 精确映射 --

// seedanceDimensions 表：Seedance 官方精确分辨率映射。
// 注意 3:4 和 9:16 分别是 4:3 和 16:9 的竖屏版本（宽高互换）。
var seedanceDimensions = map[string]map[string][2]int{
	"480p": {
		"21:9": {992, 432},
		"16:9": {864, 496},
		"4:3":  {752, 560},
		"1:1":  {640, 640},
		"3:4":  {560, 752},
		"9:16": {496, 864},
	},
	"720p": {
		"21:9": {1470, 630},
		"16:9": {1280, 720},
		"4:3":  {1112, 834},
		"1:1":  {960, 960},
		"3:4":  {834, 1112},
		"9:16": {720, 1280},
	},
	"1080p": {
		"21:9": {2206, 946},
		"16:9": {1920, 1080},
		"4:3":  {1664, 1248},
		"1:1":  {1440, 1440},
		"3:4":  {1248, 1664},
		"9:16": {1080, 1920},
	},
}

// resolveSeedanceDimensions 根据 resolution 和 aspectRatio 查表获取确切宽高。
// 不存在的组合返回 720p 16:9 默认值。
func resolveSeedanceDimensions(resolution, aspectRatio string) (width, height int) {
	res := strings.ToLower(strings.TrimSpace(resolution))
	ratio := aspectRatio
	if ratio == "" {
		ratio = "16:9"
	}
	if resMap, ok := seedanceDimensions[res]; ok {
		if dim, ok := resMap[ratio]; ok {
			return dim[0], dim[1]
		}
	}
	// fallback 默认 720p 16:9
	return 1280, 720
}

// resolutionTier 分辨率档位。
type resolutionTier int

const (
	tierLow  resolutionTier = iota // 480p / 720p（默认）
	tierHigh                       // 1080p
)

// resolveResolutionTier 根据 resolution 字段判断档位。
func resolveResolutionTier(resolution string) resolutionTier {
	r := strings.ToLower(strings.TrimSpace(resolution))
	if r == "1080p" {
		return tierHigh
	}
	return tierLow
}

// tierKey 组合 hasVideo 和 resolutionTier 生成比率映射 key。
func tierKey(hasVideo bool, tier resolutionTier) string {
	if hasVideo {
		if tier == tierHigh {
			return "high_video"
		}
		return "low_video"
	}
	if tier == tierHigh {
		return "high"
	}
	return "low"
}

// [已弃用] EstimateBilling 旧版实现（仅检测 video_url 存在性，保留用于参考）。
// func (a *TaskAdaptor) EstimateBilling(c *gin.Context, info *relaycommon.RelayInfo) map[string]float64 {
// 	if info.RelayMode == relayconstant.RelayModeSeedanceSubmit {
// 		req, err := relaycommon.GetSeedanceTaskRequest(c)
// 		if err != nil {
// 			return nil
// 		}
// 		if req.HasVideo() {
// 			if ratio, ok := GetVideoInputRatio(info.OriginModelName); ok {
// 				return map[string]float64{"video_input": ratio}
// 			}
// 		}
// 		return nil
// 	}
// 	req, err := relaycommon.GetTaskRequest(c)
// 	if err != nil {
// 		return nil
// 	}
// 	if hasVideoInMetadata(req.Metadata) {
// 		if ratio, ok := GetVideoInputRatio(info.OriginModelName); ok {
// 			return map[string]float64{"video_input": ratio}
// 		}
// 	}
// 	return nil
// }

// hasVideoInMetadata 直接检查 metadata 的 content 数组是否包含 video_url 条目，
// 避免构建完整的上游 requestPayload。
func hasVideoInMetadata(metadata map[string]interface{}) bool {
	if metadata == nil {
		return false
	}
	contentRaw, ok := metadata["content"]
	if !ok {
		return false
	}
	contentSlice, ok := contentRaw.([]interface{})
	if !ok {
		return false
	}
	for _, item := range contentSlice {
		itemMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		if itemMap["type"] == "video_url" {
			return true
		}
		if _, has := itemMap["video_url"]; has {
			return true
		}
	}
	return false
}

// BuildRequestBody converts request into Doubao specific format.
func (a *TaskAdaptor) BuildRequestBody(c *gin.Context, info *relaycommon.RelayInfo) (io.Reader, error) {
	if info.RelayMode == relayconstant.RelayModeSeedanceSubmit || isSeedance2Model(info.OriginModelName) {
		return a.buildSeedanceRequestBody(c, info)
	}
	return a.buildStandardRequestBody(c, info)
}

// buildSeedanceRequestBody passes the Seedance 2.0 official format directly to upstream without conversion.
func (a *TaskAdaptor) buildSeedanceRequestBody(c *gin.Context, info *relaycommon.RelayInfo) (io.Reader, error) {
	req, err := relaycommon.GetSeedanceTaskRequest(c)
	if err != nil {
		return nil, err
	}

	if info.IsModelMapped {
		req.Model = info.UpstreamModelName
	} else {
		info.UpstreamModelName = req.Model
	}

	data, err := common.Marshal(req)
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(data), nil
}

// buildStandardRequestBody converts TaskSubmitReq into the Doubao requestPayload format.
func (a *TaskAdaptor) buildStandardRequestBody(c *gin.Context, info *relaycommon.RelayInfo) (io.Reader, error) {
	req, err := relaycommon.GetTaskRequest(c)
	if err != nil {
		return nil, err
	}

	body, err := a.convertToRequestPayload(&req)
	if err != nil {
		return nil, errors.Wrap(err, "convert request payload failed")
	}
	if info.IsModelMapped {
		body.Model = info.UpstreamModelName
	} else {
		info.UpstreamModelName = body.Model
	}
	data, err := common.Marshal(body)
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(data), nil
}

// DoRequest delegates to common helper.
func (a *TaskAdaptor) DoRequest(c *gin.Context, info *relaycommon.RelayInfo, requestBody io.Reader) (*http.Response, error) {
	return channel.DoTaskApiRequest(a, c, info, requestBody)
}

// DoResponse handles upstream response, returns taskID etc.
func (a *TaskAdaptor) DoResponse(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (taskID string, taskData []byte, taskErr *dto.TaskError) {
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		taskErr = service.TaskErrorWrapper(err, "read_response_body_failed", http.StatusInternalServerError)
		return
	}
	_ = resp.Body.Close()

	// Parse Doubao response
	var dResp responsePayload
	if err := common.Unmarshal(responseBody, &dResp); err != nil {
		taskErr = service.TaskErrorWrapper(errors.Wrapf(err, "body: %s", responseBody), "unmarshal_response_body_failed", http.StatusInternalServerError)
		return
	}

	if dResp.ID == "" {
		taskErr = service.TaskErrorWrapper(fmt.Errorf("task_id is empty"), "invalid_response", http.StatusInternalServerError)
		return
	}

	ov := dto.NewOpenAIVideo()
	ov.ID = info.PublicTaskID
	ov.TaskID = info.PublicTaskID
	ov.CreatedAt = time.Now().Unix()
	ov.Model = info.OriginModelName

	c.JSON(http.StatusOK, ov)
	return dResp.ID, responseBody, nil
}

// FetchTask fetch task status
func (a *TaskAdaptor) FetchTask(baseUrl, key string, body map[string]any, proxy string) (*http.Response, error) {
	taskID, ok := body["task_id"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid task_id")
	}

	uri := fmt.Sprintf("%s/api/v3/contents/generations/tasks/%s", baseUrl, taskID)

	req, err := http.NewRequest(http.MethodGet, uri, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+key)

	client, err := service.GetHttpClientWithProxy(proxy)
	if err != nil {
		return nil, fmt.Errorf("new proxy http client failed: %w", err)
	}
	return client.Do(req)
}

func (a *TaskAdaptor) GetModelList() []string {
	return ModelList
}

func (a *TaskAdaptor) GetChannelName() string {
	return ChannelName
}

func (a *TaskAdaptor) convertToRequestPayload(req *relaycommon.TaskSubmitReq) (*requestPayload, error) {
	r := requestPayload{
		Model:   req.Model,
		Content: []ContentItem{},
	}

	// Add images if present
	if req.HasImage() {
		for _, imgURL := range req.Images {
			r.Content = append(r.Content, ContentItem{
				Type: "image_url",
				ImageURL: &MediaURL{
					URL: imgURL,
				},
			})
		}
	}

	metadata := req.Metadata
	if err := taskcommon.UnmarshalMetadata(metadata, &r); err != nil {
		return nil, errors.Wrap(err, "unmarshal metadata failed")
	}

	if sec, _ := strconv.Atoi(req.Seconds); sec > 0 {
		r.Duration = lo.ToPtr(dto.IntValue(sec))
	}

	r.Content = lo.Reject(r.Content, func(c ContentItem, _ int) bool { return c.Type == "text" })
	r.Content = append(r.Content, ContentItem{
		Type: "text",
		Text: req.Prompt,
	})

	return &r, nil
}

func (a *TaskAdaptor) ParseTaskResult(respBody []byte) (*relaycommon.TaskInfo, error) {
	resTask := responseTask{}
	if err := common.Unmarshal(respBody, &resTask); err != nil {
		return nil, errors.Wrap(err, "unmarshal task result failed")
	}

	taskResult := relaycommon.TaskInfo{
		Code: 0,
	}

	// Map Doubao status to internal status
	switch resTask.Status {
	case "pending", "queued":
		taskResult.Status = model.TaskStatusQueued
		taskResult.Progress = "10%"
	case "processing", "running":
		taskResult.Status = model.TaskStatusInProgress
		taskResult.Progress = "50%"
	case "succeeded":
		taskResult.Status = model.TaskStatusSuccess
		taskResult.Progress = "100%"
		taskResult.Url = resTask.Content.VideoURL
		// 解析 usage 信息用于按倍率计费
		taskResult.CompletionTokens = resTask.Usage.CompletionTokens
		taskResult.TotalTokens = resTask.Usage.TotalTokens
	case "failed":
		taskResult.Status = model.TaskStatusFailure
		taskResult.Progress = "100%"
		taskResult.Reason = resTask.Error.Message
	default:
		// Unknown status, treat as processing
		taskResult.Status = model.TaskStatusInProgress
		taskResult.Progress = "30%"
	}

	return &taskResult, nil
}

// AdjustBillingOnComplete 根据上游实际 total_tokens 多退少补。
// 公式与预扣一致：total_tokens / 1,000,000 × bc.ModelPrice × QuotaPerUnit × bc.GroupRatio
//
// bc.ModelPrice 由 estimateSeedanceBilling 设为分档价格（如 28、46、51、31）。
func (a *TaskAdaptor) AdjustBillingOnComplete(task *model.Task, taskResult *relaycommon.TaskInfo) int {
	if taskResult.TotalTokens <= 0 {
		return 0
	}
	bc := task.PrivateData.BillingContext
	if bc == nil || !isSeedance2Model(bc.OriginModelName) {
		return 0
	}
	tierPrice := bc.ModelPrice
	if tierPrice <= 0 {
		return 0
	}
	groupRatio := bc.GroupRatio
	if groupRatio <= 0 {
		groupRatio = 1.0
	}
	quota := int(float64(taskResult.TotalTokens) / 1_000_000.0 * tierPrice * common.QuotaPerUnit * groupRatio)
	if quota <= 0 {
		return 0
	}
	return quota
}

func (a *TaskAdaptor) ConvertToOpenAIVideo(originTask *model.Task) ([]byte, error) {
	var dResp responseTask
	if err := common.Unmarshal(originTask.Data, &dResp); err != nil {
		return nil, errors.Wrap(err, "unmarshal doubao task data failed")
	}

	openAIVideo := dto.NewOpenAIVideo()
	openAIVideo.ID = originTask.TaskID
	openAIVideo.TaskID = originTask.TaskID
	openAIVideo.Status = originTask.Status.ToVideoStatus()
	openAIVideo.SetProgressStr(originTask.Progress)
	openAIVideo.SetMetadata("url", dResp.Content.VideoURL)
	openAIVideo.CreatedAt = originTask.CreatedAt
	openAIVideo.CompletedAt = originTask.UpdatedAt
	openAIVideo.Model = originTask.Properties.OriginModelName

	if dResp.Status == "failed" {
		openAIVideo.Error = &dto.OpenAIVideoError{
			Message: dResp.Error.Message,
			Code:    dResp.Error.Code,
		}
	}

	return common.Marshal(openAIVideo)
}
