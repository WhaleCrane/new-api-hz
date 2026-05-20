package controller

import (
	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/service"

	"github.com/gin-gonic/gin"
)

// CreateAIGCAssetGroup 创建素材资产组合
func CreateAIGCAssetGroup(c *gin.Context) {
	var req dto.CreateAIGCAssetGroupReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}
	userId := c.GetInt("id")
	resp, err := service.CreateAIGCAssetGroup(c.Request.Context(), userId, req.Name, req.Description, req.ProjectName)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, resp)
}

// ListAIGCAssetGroups 查询素材资产组合列表
func ListAIGCAssetGroups(c *gin.Context) {
	var req dto.ListAIGCAssetGroupsReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}
	userId := c.GetInt("id")
	input := &service.ListAIGCAssetGroupsInput{
		Name:       req.Name,
		GroupType:  req.GroupType,
		PageNumber: req.PageNumber,
		PageSize:   req.PageSize,
	}
	resp, err := service.ListAIGCAssetGroups(c.Request.Context(), userId, input)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, resp)
}

// GetAIGCAssetGroup 获取单个素材资产组合信息
func GetAIGCAssetGroup(c *gin.Context) {
	var req dto.GetAIGCAssetGroupReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}
	userId := c.GetInt("id")
	resp, err := service.GetAIGCAssetGroup(c.Request.Context(), userId, req.ID)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, resp)
}

// UpdateAIGCAssetGroup 更新素材资产组合信息
func UpdateAIGCAssetGroup(c *gin.Context) {
	var req dto.UpdateAIGCAssetGroupReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}
	userId := c.GetInt("id")
	resp, err := service.UpdateAIGCAssetGroup(c.Request.Context(), userId, req.ID, req.Name, req.Title, req.Description)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, resp)
}

// DeleteAIGCAssetGroup 删除素材资产组合
func DeleteAIGCAssetGroup(c *gin.Context) {
	var req dto.DeleteAIGCAssetGroupReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}
	userId := c.GetInt("id")
	resp, err := service.DeleteAIGCAssetGroup(c.Request.Context(), userId, req.ID)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, resp)
}

// CreateAIGCAsset 创建素材资产
func CreateAIGCAsset(c *gin.Context) {
	var req dto.CreateAIGCAssetReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}
	userId := c.GetInt("id")
	resp, err := service.CreateAIGCAsset(c.Request.Context(), userId, req.GroupID, req.URL, req.AssetType, req.Name)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, resp)
}

// ListAIGCAssets 查询素材资产列表
func ListAIGCAssets(c *gin.Context) {
	var req dto.ListAIGCAssetsReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}
	userId := c.GetInt("id")
	input := &service.ListAIGCAssetsInput{
		GroupIDs:   req.GroupIDs,
		GroupType:  req.GroupType,
		Statuses:   req.Statuses,
		Name:       req.Name,
		PageNumber: req.PageNumber,
		PageSize:   req.PageSize,
		SortBy:     req.SortBy,
		SortOrder:  req.SortOrder,
	}
	resp, err := service.ListAIGCAssets(c.Request.Context(), userId, input)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, resp)
}

// GetAIGCAsset 获取单个素材资产信息
func GetAIGCAsset(c *gin.Context) {
	var req dto.GetAIGCAssetReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}
	userId := c.GetInt("id")
	resp, err := service.GetAIGCAsset(c.Request.Context(), userId, req.ID)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, resp)
}

// UpdateAIGCAsset 更新素材资产信息
func UpdateAIGCAsset(c *gin.Context) {
	var req dto.UpdateAIGCAssetReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}
	userId := c.GetInt("id")
	resp, err := service.UpdateAIGCAsset(c.Request.Context(), userId, req.ID, req.Name)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, resp)
}

// DeleteAIGCAsset 删除素材资产
func DeleteAIGCAsset(c *gin.Context) {
	var req dto.DeleteAIGCAssetReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}
	userId := c.GetInt("id")
	resp, err := service.DeleteAIGCAsset(c.Request.Context(), userId, req.ID)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, resp)
}
