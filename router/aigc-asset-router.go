package router

import (
	"github.com/QuantumNous/new-api/controller"
	"github.com/QuantumNous/new-api/middleware"

	"github.com/gin-gonic/gin"
)

func SetAIGCAssetRouter(router *gin.Engine) {
	aigcAssetGroup := router.Group("/s2/aigc-asset")
	aigcAssetGroup.Use(middleware.RouteTag("relay"))
	aigcAssetGroup.Use(middleware.TokenAuth(), middleware.Distribute())
	{
		// 素材资产组合 CRUD
		aigcAssetGroup.POST("/asset-groups/create", controller.CreateAIGCAssetGroup)
		aigcAssetGroup.POST("/asset-groups/list", controller.ListAIGCAssetGroups)
		aigcAssetGroup.POST("/asset-groups/get", controller.GetAIGCAssetGroup)
		aigcAssetGroup.POST("/asset-groups/update", controller.UpdateAIGCAssetGroup)
		aigcAssetGroup.POST("/asset-groups/delete", controller.DeleteAIGCAssetGroup)

		// 素材资产 CRUD
		aigcAssetGroup.POST("/assets/create", controller.CreateAIGCAsset)
		aigcAssetGroup.POST("/assets/list", controller.ListAIGCAssets)
		aigcAssetGroup.POST("/assets/get", controller.GetAIGCAsset)
		aigcAssetGroup.POST("/assets/update", controller.UpdateAIGCAsset)
		aigcAssetGroup.POST("/assets/delete", controller.DeleteAIGCAsset)
	}
}
