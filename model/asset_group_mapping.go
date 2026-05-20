package model

import (
	"errors"
	"time"

	"gorm.io/gorm"
)

// AssetGroupMapping 记录用户与火山引擎 Asset Group 的绑定关系，实现按用户隔离
type AssetGroupMapping struct {
	ID              int    `json:"id" gorm:"primaryKey;autoIncrement"`
	UserId          int    `json:"user_id" gorm:"uniqueIndex:idx_user_group;index"`
	GroupId         string `json:"group_id" gorm:"uniqueIndex:idx_user_group;type:varchar(128)"`
	ChannelId       int    `json:"channel_id" gorm:"index"`
	VolcProjectName string `json:"volc_project_name" gorm:"type:varchar(64)"`
	Name            string `json:"name" gorm:"type:varchar(128)"`
	CreatedAt       int64  `json:"created_at"`
	UpdatedAt       int64  `json:"updated_at"`
}

func (m *AssetGroupMapping) BeforeCreate() {
	m.CreatedAt = time.Now().Unix()
	m.UpdatedAt = time.Now().Unix()
}

func (m *AssetGroupMapping) BeforeUpdate() {
	m.UpdatedAt = time.Now().Unix()
}

// GetUserAssetGroupMapping 获取用户绑定的 Asset Group
func GetUserAssetGroupMapping(userId int) (*AssetGroupMapping, error) {
	mapping := &AssetGroupMapping{}
	err := DB.Where("user_id = ?", userId).First(mapping).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return mapping, nil
}

// InsertAssetGroupMapping 插入用户映射
func InsertAssetGroupMapping(mapping *AssetGroupMapping) error {
	return DB.Create(mapping).Error
}

// GetAssetGroupMappingByGroupId 通过 GroupId 查询映射
func GetAssetGroupMappingByGroupId(groupId string) (*AssetGroupMapping, error) {
	mapping := &AssetGroupMapping{}
	err := DB.Where("group_id = ?", groupId).First(mapping).Error
	if err != nil {
		return nil, err
	}
	return mapping, nil
}

// GetUserAssetGroupNames 获取用户所有素材组名称列表
func GetUserAssetGroupNames(userId int) ([]string, error) {
	var names []string
	err := DB.Model(&AssetGroupMapping{}).
		Where("user_id = ?", userId).
		Select("name").
		Find(&names).Error
	return names, err
}
