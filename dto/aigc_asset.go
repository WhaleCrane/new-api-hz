package dto

// AIGC Asset Group DTOs

type CreateAIGCAssetGroupReq struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	ProjectName string `json:"project_name,omitempty"`
}

type CreateAIGCAssetGroupResp struct {
	ID string `json:"id"`
}

type ListAIGCAssetGroupsReq struct {
	Name       string `json:"name,omitempty"`
	GroupType  string `json:"group_type,omitempty"` // AIGC
	PageNumber int    `json:"page_number,omitempty"`
	PageSize   int    `json:"page_size,omitempty"`
}

type GetAIGCAssetGroupReq struct {
	ID string `json:"id"`
}

type UpdateAIGCAssetGroupReq struct {
	ID          string `json:"id"`
	Name        string `json:"name,omitempty"`
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
}

type DeleteAIGCAssetGroupReq struct {
	ID string `json:"id"`
}

// AIGC Asset DTOs (复用 asset.go 中的 AssetDTO, ListAssetsResp)

type CreateAIGCAssetReq struct {
	GroupID   string `json:"group_id"`
	URL       string `json:"url"`
	AssetType string `json:"asset_type"` // Image, Video, Audio
	Name      string `json:"name,omitempty"`
}

type CreateAIGCAssetResp struct {
	ID string `json:"id"`
}

type ListAIGCAssetsReq struct {
	GroupIDs   []string `json:"group_ids,omitempty"`
	GroupType  string   `json:"group_type,omitempty"` // AIGC
	Statuses   []string `json:"statuses,omitempty"`   // Active, Processing, Failed
	Name       string   `json:"name,omitempty"`
	PageNumber int      `json:"page_number,omitempty"`
	PageSize   int      `json:"page_size,omitempty"`
	SortBy     string   `json:"sort_by,omitempty"`
	SortOrder  string   `json:"sort_order,omitempty"`
}

type GetAIGCAssetReq struct {
	ID string `json:"id"`
}

type UpdateAIGCAssetReq struct {
	ID   string `json:"id"`
	Name string `json:"name,omitempty"`
}

type DeleteAIGCAssetReq struct {
	ID string `json:"id"`
}
