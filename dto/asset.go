package dto

type CreateVisualValidateSessionReq struct {
	CallbackURL string `json:"callback_url"`
}

type CreateVisualValidateSessionResp struct {
	BytedToken  string `json:"byted_token"`
	H5Link      string `json:"h5_link"`
	CallbackURL string `json:"callback_url"`
}

type GetVisualValidateResultReq struct {
	BytedToken string `json:"byted_token"`
}

type GetVisualValidateResultResp struct {
	GroupID string `json:"group_id"`
}

type CreateAssetReq struct {
	GroupID   string `json:"group_id"`
	URL       string `json:"url"`
	AssetType string `json:"asset_type"` // Image, Video, Audio
	Name      string `json:"name,omitempty"`
}

type CreateAssetResp struct {
	ID string `json:"id"`
}

type ListAssetsReq struct {
	GroupIDs  []string `json:"group_ids,omitempty"`
	GroupType string   `json:"group_type,omitempty"` // LivenessFace
	Statuses  []string `json:"statuses,omitempty"`   // Active, Processing, Failed
	Name      string   `json:"name,omitempty"`
	PageNumber int    `json:"page_number,omitempty"`
	PageSize   int    `json:"page_size,omitempty"`
	SortBy     string `json:"sort_by,omitempty"`
	SortOrder  string `json:"sort_order,omitempty"`
}

type AssetDTO struct {
	ID          string `json:"id"`
	GroupID     string `json:"group_id"`
	Status      string `json:"status"`
	AssetType   string `json:"asset_type"`
	URL         string `json:"url"`
	Name        string `json:"name"`
	ProjectName string `json:"project_name"`
	CreateTime  string `json:"create_time"`
	UpdateTime  string `json:"update_time"`
}

type ListAssetsResp struct {
	Items      []AssetDTO `json:"items"`
	TotalCount int        `json:"total_count"`
	PageNumber int        `json:"page_number"`
	PageSize   int        `json:"page_size"`
}

type GetAssetReq struct {
	ID string `json:"id"`
}

type UpdateAssetReq struct {
	ID   string `json:"id"`
	Name string `json:"name,omitempty"`
}

type DeleteAssetReq struct {
	ID string `json:"id"`
}

type ListAssetGroupsReq struct {
	Name       string   `json:"name,omitempty"`
	GroupIDs   []string `json:"group_ids,omitempty"`
	GroupType  string   `json:"group_type,omitempty"` // LivenessFace
	PageNumber int      `json:"page_number,omitempty"`
	PageSize   int      `json:"page_size,omitempty"`
	SortBy     string   `json:"sort_by,omitempty"`
	SortOrder  string   `json:"sort_order,omitempty"`
}

type AssetGroupDTO struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Title       string `json:"title"`
	Description string `json:"description"`
	GroupType   string `json:"group_type"`
	ProjectName string `json:"project_name"`
	CreateTime  string `json:"create_time"`
	UpdateTime  string `json:"update_time"`
}

type ListAssetGroupsResp struct {
	Items      []AssetGroupDTO `json:"items"`
	TotalCount int             `json:"total_count"`
	PageNumber int             `json:"page_number"`
	PageSize   int             `json:"page_size"`
}

type GetAssetGroupReq struct {
	ID string `json:"id"`
}

type UpdateAssetGroupReq struct {
	ID          string `json:"id"`
	Name        string `json:"name,omitempty"`
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
}

type DeleteAssetGroupReq struct {
	ID string `json:"id"`
}
