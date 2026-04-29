package service

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
)

const arkBaseURL = "https://ark.cn-beijing.volces.com"
const arkAPIVersion = "2024-01-01"

// CallArkAPI 调用火山引擎 Ark API
// action: 上游 API Action 名称（如 CreateAsset, GetAsset 等）
// reqBody: 请求体（map 结构，将被 JSON 序列化）
func CallArkAPI(ctx context.Context, channel *model.Channel, action string, reqBody map[string]any) (map[string]any, error) {
	accessKey, secretKey, err := ParseVolcengineAssetAuth(channel.Key)
	if err != nil {
		return nil, fmt.Errorf("parse volcengine auth failed: %w", err)
	}

	// 构建请求 URL
	baseURL := arkBaseURL
	if channel.BaseURL != nil && *channel.BaseURL != "" {
		baseURL = *channel.BaseURL
	}

	reqURL := fmt.Sprintf("%s/?Action=%s&Version=%s", baseURL, action, arkAPIVersion)

	jsonBody, err := common.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request body failed: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("create request failed: %w", err)
	}

	// 签名
	if err := SignArkRequest(httpReq, accessKey, secretKey); err != nil {
		return nil, fmt.Errorf("sign request failed: %w", err)
	}

	// 获取 HTTP 客户端（支持 channel 级别的 proxy）
	proxyURL := ""
	if channel.Setting != nil {
		var setting dtoChannelSetting
		if err := common.UnmarshalJsonStr(*channel.Setting, &setting); err == nil {
			proxyURL = setting.Proxy
		}
	}
	client, err := GetHttpClientWithProxy(proxyURL)
	if err != nil {
		return nil, fmt.Errorf("get http client failed: %w", err)
	}

	httpResp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("send request failed: %w", err)
	}
	defer httpResp.Body.Close()

	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body failed: %w", err)
	}

	if httpResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("upstream error: status=%d, body=%s", httpResp.StatusCode, string(respBody))
	}

	var result map[string]any
	if err := common.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response failed: %w", err)
	}

	return result, nil
}

// BuildArkURL 构建火山引擎 Ark API URL
func BuildArkURL(channel *model.Channel, action string) string {
	baseURL := arkBaseURL
	if channel.BaseURL != nil && *channel.BaseURL != "" {
		baseURL = *channel.BaseURL
	}
	return fmt.Sprintf("%s/?Action=%s&Version=%s", baseURL, action, arkAPIVersion)
}

// dtoChannelSetting 用于解析 channel.Setting 中的 proxy 字段
type dtoChannelSetting struct {
	Proxy string `json:"proxy"`
}

// CreateVisualValidateSession 调用 CreateVisualValidateSession API
func CreateVisualValidateSession(ctx context.Context, channel *model.Channel, callbackURL, projectName string) (map[string]any, error) {
	reqBody := map[string]any{
		"CallbackURL": callbackURL,
	}
	if projectName != "" {
		reqBody["ProjectName"] = projectName
	}
	return CallArkAPI(ctx, channel, "CreateVisualValidateSession", reqBody)
}

// GetVisualValidateResult 调用 GetVisualValidateResult API
func GetVisualValidateResult(ctx context.Context, channel *model.Channel, bytedToken, projectName string) (map[string]any, error) {
	reqBody := map[string]any{
		"BytedToken": bytedToken,
	}
	if projectName != "" {
		reqBody["ProjectName"] = projectName
	}
	return CallArkAPI(ctx, channel, "GetVisualValidateResult", reqBody)
}

// CreateAsset 调用 CreateAsset API
func CreateAsset(ctx context.Context, channel *model.Channel, groupID, assetURL, assetType, name, projectName string) (map[string]any, error) {
	reqBody := map[string]any{
		"GroupId":   groupID,
		"URL":       assetURL,
		"AssetType": assetType,
	}
	if name != "" {
		reqBody["Name"] = name
	}
	if projectName != "" {
		reqBody["ProjectName"] = projectName
	}
	return CallArkAPI(ctx, channel, "CreateAsset", reqBody)
}

// ListAssets 调用 ListAssets API
func ListAssets(ctx context.Context, channel *model.Channel, req *ListAssetsInput) (map[string]any, error) {
	reqBody := map[string]any{}

	if len(req.GroupIDs) > 0 {
		reqBody["Filter"] = map[string]any{
			"GroupIds": req.GroupIDs,
		}
		filter := reqBody["Filter"].(map[string]any)
		if req.GroupType != "" {
			filter["GroupType"] = req.GroupType
		}
		if len(req.Statuses) > 0 {
			filter["Statuses"] = req.Statuses
		}
		if req.Name != "" {
			filter["Name"] = req.Name
		}
	}
	if req.PageNumber > 0 {
		reqBody["PageNumber"] = req.PageNumber
	}
	if req.PageSize > 0 {
		reqBody["PageSize"] = req.PageSize
	}
	if req.SortBy != "" {
		reqBody["SortBy"] = req.SortBy
	}
	if req.SortOrder != "" {
		reqBody["SortOrder"] = req.SortOrder
	}
	return CallArkAPI(ctx, channel, "ListAssets", reqBody)
}

type ListAssetsInput struct {
	GroupIDs   []string
	GroupType  string
	Statuses   []string
	Name       string
	PageNumber int
	PageSize   int
	SortBy     string
	SortOrder  string
}

// GetAsset 调用 GetAsset API
func GetAsset(ctx context.Context, channel *model.Channel, id, projectName string) (map[string]any, error) {
	reqBody := map[string]any{
		"Id": id,
	}
	if projectName != "" {
		reqBody["ProjectName"] = projectName
	}
	return CallArkAPI(ctx, channel, "GetAsset", reqBody)
}

// UpdateAsset 调用 UpdateAsset API
func UpdateAsset(ctx context.Context, channel *model.Channel, id, name, projectName string) (map[string]any, error) {
	reqBody := map[string]any{
		"Id": id,
	}
	if name != "" {
		reqBody["Name"] = name
	}
	if projectName != "" {
		reqBody["ProjectName"] = projectName
	}
	return CallArkAPI(ctx, channel, "UpdateAsset", reqBody)
}

// DeleteAsset 调用 DeleteAsset API
func DeleteAsset(ctx context.Context, channel *model.Channel, id, projectName string) (map[string]any, error) {
	reqBody := map[string]any{
		"Id": id,
	}
	if projectName != "" {
		reqBody["ProjectName"] = projectName
	}
	return CallArkAPI(ctx, channel, "DeleteAsset", reqBody)
}

// ListAssetGroups 调用 ListAssetGroups API
func ListAssetGroups(ctx context.Context, channel *model.Channel, req *ListAssetGroupsInput) (map[string]any, error) {
	reqBody := map[string]any{}

	if req.Name != "" || len(req.GroupIDs) > 0 || req.GroupType != "" {
		filter := map[string]any{}
		if req.Name != "" {
			filter["Name"] = req.Name
		}
		if len(req.GroupIDs) > 0 {
			filter["GroupIds"] = req.GroupIDs
		}
		if req.GroupType != "" {
			filter["GroupType"] = req.GroupType
		}
		reqBody["Filter"] = filter
	}
	if req.PageNumber > 0 {
		reqBody["PageNumber"] = req.PageNumber
	}
	if req.PageSize > 0 {
		reqBody["PageSize"] = req.PageSize
	}
	return CallArkAPI(ctx, channel, "ListAssetGroups", reqBody)
}

type ListAssetGroupsInput struct {
	Name       string
	GroupIDs   []string
	GroupType  string
	PageNumber int
	PageSize   int
}

// GetAssetGroup 调用 GetAssetGroup API
func GetAssetGroup(ctx context.Context, channel *model.Channel, id, projectName string) (map[string]any, error) {
	reqBody := map[string]any{
		"Id": id,
	}
	if projectName != "" {
		reqBody["ProjectName"] = projectName
	}
	return CallArkAPI(ctx, channel, "GetAssetGroup", reqBody)
}

// UpdateAssetGroup 调用 UpdateAssetGroup API
func UpdateAssetGroup(ctx context.Context, channel *model.Channel, id, name, title, description, projectName string) (map[string]any, error) {
	reqBody := map[string]any{
		"Id": id,
	}
	if name != "" {
		reqBody["Name"] = name
	}
	if title != "" {
		reqBody["Title"] = title
	}
	if description != "" {
		reqBody["Description"] = description
	}
	if projectName != "" {
		reqBody["ProjectName"] = projectName
	}
	return CallArkAPI(ctx, channel, "UpdateAssetGroup", reqBody)
}

// DeleteAssetGroup 调用 DeleteAssetGroup API
func DeleteAssetGroup(ctx context.Context, channel *model.Channel, id, projectName string) (map[string]any, error) {
	reqBody := map[string]any{
		"Id": id,
	}
	if projectName != "" {
		reqBody["ProjectName"] = projectName
	}
	return CallArkAPI(ctx, channel, "DeleteAssetGroup", reqBody)
}
