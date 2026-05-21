package router

import (
	"github.com/QuantumNous/new-api/controller"
	"github.com/QuantumNous/new-api/middleware"

	"github.com/gin-gonic/gin"
)

func SetAssetRouter(router *gin.Engine) {
	assetGroup := router.Group("/s1/asset")
	assetGroup.Use(middleware.RouteTag("relay"))
	assetGroup.Use(middleware.TokenAuth(), middleware.Distribute())
	{
		// 真人认证
		assetGroup.POST("/visual-validate/session", controller.CreateVisualValidateSession)
		assetGroup.POST("/visual-validate/result", controller.GetVisualValidateResult)

		// 素材资产 CRUD
		assetGroup.POST("/assets/create", controller.CreateAsset)
		assetGroup.POST("/assets/list", controller.ListAssets)
		assetGroup.POST("/assets/get", controller.GetAsset)
		assetGroup.POST("/assets/update", controller.UpdateAsset)
		assetGroup.POST("/assets/delete", controller.DeleteAsset)

		// 素材资产组 CRUD
		assetGroup.POST("/asset-groups/list", controller.ListAssetGroups)
		assetGroup.POST("/asset-groups/get", controller.GetAssetGroup)
		assetGroup.POST("/asset-groups/update", controller.UpdateAssetGroup)
		assetGroup.POST("/asset-groups/delete", controller.DeleteAssetGroup)
	}
}
