package service

import (
	"context"
	"fmt"

	"github.com/QuantumNous/new-api/model"
)

// resolveAIGCChannelForUser 获取平台默认 VolcEngineAIGC channel 和用户的 project name
func resolveAIGCChannelForUser(userId int) (*model.Channel, string, error) {
	mapping, err := model.GetUserAIGCAssetGroupMapping(userId)
	if err != nil {
		return nil, "", fmt.Errorf("query user mapping failed: %w", err)
	}
	if mapping == nil || mapping.ChannelId == 0 {
		// 用户尚未绑定，尝试获取平台第一个可用的 VolcEngineAIGC channel
		var channels []*model.Channel
		err := model.DB.Where("type = ? and status = ? limit 1", 45, 1).Find(&channels).Error
		if err != nil || len(channels) == 0 {
			return nil, "", fmt.Errorf("no volcengine AIGC channel configured, please contact admin")
		}
		return channels[0], "default", nil
	}

	channel, err := model.GetChannelById(mapping.ChannelId, true)
	if err != nil {
		return nil, "", fmt.Errorf("get channel failed: %w", err)
	}
	projectName := mapping.VolcProjectName
	if projectName == "" {
		projectName = "default"
	}
	return channel, projectName, nil
}

// ensureAIGCUserMapping 确保用户有映射，无则自动分配 channel
func ensureAIGCUserMapping(ctx context.Context, userId int) (*model.Channel, string, error) {
	channel, projectName, err := resolveAIGCChannelForUser(userId)
	if err != nil {
		return nil, "", err
	}
	return channel, projectName, nil
}

// CreateAIGCAssetGroup 创建素材资产组合
func CreateAIGCAssetGroup(ctx context.Context, userId int, name, description, projectName string) (map[string]any, error) {
	channel, defaultProject, err := ensureAIGCUserMapping(ctx, userId)
	if err != nil {
		return nil, err
	}
	if projectName == "" {
		projectName = defaultProject
	}

	reqBody := map[string]any{
		"Name":        name,
		"Description": description,
		"GroupType":   "AIGC",
		"ProjectName": projectName,
	}
	resp, err := CallArkAPI(ctx, channel, "CreateAssetGroup", reqBody)
	if err != nil {
		return nil, err
	}

	// 从响应结果中提取 GroupId，创建映射
	if result, ok := resp["Result"].(map[string]any); ok {
		if groupId, ok := result["Id"].(string); groupId != "" && ok {
			_, _ = CreateAIGCAssetMapping(userId, groupId, channel.Id, projectName)
		}
	}

	return resp, nil
}

// CreateAIGCAssetMapping 在用户创建素材组后创建映射
func CreateAIGCAssetMapping(userId int, groupId string, channelId int, projectName string) (*model.AIGCAssetGroupMapping, error) {
	existing, err := model.GetAIGCAssetGroupMapping(userId, groupId)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return existing, nil // 该 group_id 已存在，直接返回
	}

	mapping := &model.AIGCAssetGroupMapping{
		UserId:          userId,
		GroupId:         groupId,
		ChannelId:       channelId,
		VolcProjectName: projectName,
	}
	if err := model.InsertAIGCAssetGroupMapping(mapping); err != nil {
		return nil, fmt.Errorf("insert AIGC asset group mapping failed: %w", err)
	}
	return mapping, nil
}

// CreateAIGCAsset 创建素材
func CreateAIGCAsset(ctx context.Context, userId int, groupID, assetURL, assetType, name string) (map[string]any, error) {
	// 校验用户是否已创建素材组
	userGroupIDs, err := model.GetUserAIGCGroupIDs(userId)
	if err != nil {
		return nil, fmt.Errorf("failed to query user AIGC groups: %w", err)
	}
	if len(userGroupIDs) == 0 {
		return nil, fmt.Errorf("user must create an AIGC asset group before creating assets")
	}

	// 校验 group_id 是否属于当前用户
	found := false
	for _, gid := range userGroupIDs {
		if gid == groupID {
			found = true
			break
		}
	}
	if !found {
		return nil, fmt.Errorf("group_id does not belong to current user")
	}

	channel, projectName, err := resolveAIGCChannelForUser(userId)
	if err != nil {
		return nil, err
	}

	reqBody := map[string]any{
		"GroupId":     groupID,
		"URL":         assetURL,
		"AssetType":   assetType,
		"ProjectName": projectName,
	}
	if name != "" {
		reqBody["Name"] = name
	}
	return CallArkAPI(ctx, channel, "CreateAsset", reqBody)
}

type ListAIGCAssetsInput struct {
	GroupIDs   []string
	GroupType  string
	Statuses   []string
	Name       string
	PageNumber int
	PageSize   int
	SortBy     string
	SortOrder  string
}

// ListAIGCAssets 查询素材列表（自动按用户隔离）
func ListAIGCAssets(ctx context.Context, userId int, req *ListAIGCAssetsInput) (map[string]any, error) {
	channel, projectName, err := resolveAIGCChannelForUser(userId)
	if err != nil {
		return nil, err
	}

	// 自动注入用户拥有的 group IDs
	userGroupIDs, err := model.GetUserAIGCGroupIDs(userId)
	if err != nil {
		return nil, err
	}

	reqBody := map[string]any{
		"Filter": map[string]any{
			"GroupIds": userGroupIDs,
		},
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
	if projectName != "" {
		filter["ProjectName"] = projectName
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

// GetAIGCAsset 获取单个素材
func GetAIGCAsset(ctx context.Context, userId int, id string) (map[string]any, error) {
	channel, projectName, err := resolveAIGCChannelForUser(userId)
	if err != nil {
		return nil, err
	}

	reqBody := map[string]any{
		"Id": id,
	}
	if projectName != "" {
		reqBody["ProjectName"] = projectName
	}
	return CallArkAPI(ctx, channel, "GetAsset", reqBody)
}

// UpdateAIGCAsset 更新素材
func UpdateAIGCAsset(ctx context.Context, userId int, id, name string) (map[string]any, error) {
	channel, projectName, err := resolveAIGCChannelForUser(userId)
	if err != nil {
		return nil, err
	}

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

// DeleteAIGCAsset 删除素材
func DeleteAIGCAsset(ctx context.Context, userId int, id string) (map[string]any, error) {
	channel, projectName, err := resolveAIGCChannelForUser(userId)
	if err != nil {
		return nil, err
	}

	reqBody := map[string]any{
		"Id": id,
	}
	if projectName != "" {
		reqBody["ProjectName"] = projectName
	}
	return CallArkAPI(ctx, channel, "DeleteAsset", reqBody)
}

type ListAIGCAssetGroupsInput struct {
	Name       string
	GroupIDs   []string
	GroupType  string
	PageNumber int
	PageSize   int
}

// ListAIGCAssetGroups 查询素材组列表（自动按用户隔离）
func ListAIGCAssetGroups(ctx context.Context, userId int, req *ListAIGCAssetGroupsInput) (map[string]any, error) {
	channel, projectName, err := resolveAIGCChannelForUser(userId)
	if err != nil {
		return nil, err
	}

	// 自动注入用户拥有的 group IDs
	userGroupIDs, err := model.GetUserAIGCGroupIDs(userId)
	if err != nil {
		return nil, err
	}

	reqBody := map[string]any{
		"Filter": map[string]any{
			"GroupIds": userGroupIDs,
		},
	}
	filter := reqBody["Filter"].(map[string]any)
	if req.Name != "" {
		filter["Name"] = req.Name
	}
	if req.GroupType != "" {
		filter["GroupType"] = req.GroupType
	}
	if projectName != "" {
		filter["ProjectName"] = projectName
	}
	if req.PageNumber > 0 {
		reqBody["PageNumber"] = req.PageNumber
	}
	if req.PageSize > 0 {
		reqBody["PageSize"] = req.PageSize
	}
	return CallArkAPI(ctx, channel, "ListAssetGroups", reqBody)
}

// GetAIGCAssetGroup 获取素材组
func GetAIGCAssetGroup(ctx context.Context, userId int, id string) (map[string]any, error) {
	channel, projectName, err := resolveAIGCChannelForUser(userId)
	if err != nil {
		return nil, err
	}

	reqBody := map[string]any{
		"Id": id,
	}
	if projectName != "" {
		reqBody["ProjectName"] = projectName
	}
	return CallArkAPI(ctx, channel, "GetAssetGroup", reqBody)
}

// UpdateAIGCAssetGroup 更新素材组
func UpdateAIGCAssetGroup(ctx context.Context, userId int, id, name, title, description string) (map[string]any, error) {
	channel, projectName, err := resolveAIGCChannelForUser(userId)
	if err != nil {
		return nil, err
	}

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

// DeleteAIGCAssetGroup 删除素材组
func DeleteAIGCAssetGroup(ctx context.Context, userId int, id string) (map[string]any, error) {
	channel, projectName, err := resolveAIGCChannelForUser(userId)
	if err != nil {
		return nil, err
	}

	reqBody := map[string]any{
		"Id": id,
	}
	if projectName != "" {
		reqBody["ProjectName"] = projectName
	}
	resp, err := CallArkAPI(ctx, channel, "DeleteAssetGroup", reqBody)
	if err != nil {
		return nil, err
	}

	// 删除成功后，清理本地映射
	if err := model.DeleteAIGCAssetGroupMapping(userId, id); err != nil {
		return nil, fmt.Errorf("failed to delete asset group mapping: %w", err)
	}

	return resp, nil
}
