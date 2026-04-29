package controller

import (
	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"

	"github.com/gin-gonic/gin"
)

func getChannelForAsset(c *gin.Context, channelID int) (*model.Channel, error) {
	channel, err := model.GetChannelById(channelID, true)
	if err != nil {
		common.ApiError(c, err)
		return nil, err
	}
	return channel, nil
}

// CreateVisualValidateSession 创建真人认证 H5 会话
func CreateVisualValidateSession(c *gin.Context) {
	var req dto.CreateVisualValidateSessionReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}
	channel, err := getChannelForAsset(c, req.ChannelID)
	if err != nil {
		return
	}
	resp, err := service.CreateVisualValidateSession(c.Request.Context(), channel, req.CallbackURL, req.ProjectName)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, resp)
}

// GetVisualValidateResult 获取真人认证结果和 Group ID
func GetVisualValidateResult(c *gin.Context) {
	var req dto.GetVisualValidateResultReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}
	channel, err := getChannelForAsset(c, req.ChannelID)
	if err != nil {
		return
	}
	resp, err := service.GetVisualValidateResult(c.Request.Context(), channel, req.BytedToken, req.ProjectName)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, resp)
}

// CreateAsset 创建素材资产
func CreateAsset(c *gin.Context) {
	var req dto.CreateAssetReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}
	channel, err := getChannelForAsset(c, req.ChannelID)
	if err != nil {
		return
	}
	resp, err := service.CreateAsset(c.Request.Context(), channel, req.GroupID, req.URL, req.AssetType, req.Name, req.ProjectName)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, resp)
}

// ListAssets 查询素材资产列表
func ListAssets(c *gin.Context) {
	var req dto.ListAssetsReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}
	channel, err := getChannelForAsset(c, req.ChannelID)
	if err != nil {
		return
	}
	input := &service.ListAssetsInput{
		GroupIDs:   req.GroupIDs,
		GroupType:  req.GroupType,
		Statuses:   req.Statuses,
		Name:       req.Name,
		PageNumber: req.PageNumber,
		PageSize:   req.PageSize,
		SortBy:     req.SortBy,
		SortOrder:  req.SortOrder,
	}
	resp, err := service.ListAssets(c.Request.Context(), channel, input)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, resp)
}

// GetAsset 获取单个素材资产信息
func GetAsset(c *gin.Context) {
	var req dto.GetAssetReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}
	channel, err := getChannelForAsset(c, req.ChannelID)
	if err != nil {
		return
	}
	resp, err := service.GetAsset(c.Request.Context(), channel, req.ID, req.ProjectName)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, resp)
}

// UpdateAsset 更新素材资产信息
func UpdateAsset(c *gin.Context) {
	var req dto.UpdateAssetReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}
	channel, err := getChannelForAsset(c, req.ChannelID)
	if err != nil {
		return
	}
	resp, err := service.UpdateAsset(c.Request.Context(), channel, req.ID, req.Name, req.ProjectName)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, resp)
}

// DeleteAsset 删除素材资产
func DeleteAsset(c *gin.Context) {
	var req dto.DeleteAssetReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}
	channel, err := getChannelForAsset(c, req.ChannelID)
	if err != nil {
		return
	}
	resp, err := service.DeleteAsset(c.Request.Context(), channel, req.ID, req.ProjectName)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, resp)
}

// ListAssetGroups 查询素材资产组列表
func ListAssetGroups(c *gin.Context) {
	var req dto.ListAssetGroupsReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}
	channel, err := getChannelForAsset(c, req.ChannelID)
	if err != nil {
		return
	}
	input := &service.ListAssetGroupsInput{
		Name:       req.Name,
		GroupIDs:   req.GroupIDs,
		GroupType:  req.GroupType,
		PageNumber: req.PageNumber,
		PageSize:   req.PageSize,
	}
	resp, err := service.ListAssetGroups(c.Request.Context(), channel, input)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, resp)
}

// GetAssetGroup 获取单个素材资产组信息
func GetAssetGroup(c *gin.Context) {
	var req dto.GetAssetGroupReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}
	channel, err := getChannelForAsset(c, req.ChannelID)
	if err != nil {
		return
	}
	resp, err := service.GetAssetGroup(c.Request.Context(), channel, req.ID, req.ProjectName)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, resp)
}

// UpdateAssetGroup 更新素材资产组信息
func UpdateAssetGroup(c *gin.Context) {
	var req dto.UpdateAssetGroupReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}
	channel, err := getChannelForAsset(c, req.ChannelID)
	if err != nil {
		return
	}
	resp, err := service.UpdateAssetGroup(c.Request.Context(), channel, req.ID, req.Name, req.Title, req.Description, req.ProjectName)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, resp)
}

// DeleteAssetGroup 删除素材资产组
func DeleteAssetGroup(c *gin.Context) {
	var req dto.DeleteAssetGroupReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}
	channel, err := getChannelForAsset(c, req.ChannelID)
	if err != nil {
		return
	}
	resp, err := service.DeleteAssetGroup(c.Request.Context(), channel, req.ID, req.ProjectName)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, resp)
}
