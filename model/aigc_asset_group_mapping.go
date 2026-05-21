package model

import (
	"errors"
	"time"

	"gorm.io/gorm"
)

// AIGCAssetGroupMapping 记录用户与火山引擎 AIGC Asset Group 的绑定关系，实现按用户隔离
type AIGCAssetGroupMapping struct {
	ID              int    `json:"id" gorm:"primaryKey;autoIncrement"`
	UserId          int    `json:"user_id" gorm:"uniqueIndex:idx_aigc_user_group;index"`
	GroupId         string `json:"group_id" gorm:"uniqueIndex:idx_aigc_user_group;type:varchar(128)"`
	ChannelId       int    `json:"channel_id" gorm:"index"`
	VolcProjectName string `json:"volc_project_name" gorm:"type:varchar(64)"`
	Name            string `json:"name" gorm:"type:varchar(128)"`
	CreatedAt       int64  `json:"created_at"`
	UpdatedAt       int64  `json:"updated_at"`
}

func (m *AIGCAssetGroupMapping) BeforeCreate() {
	m.CreatedAt = time.Now().Unix()
	m.UpdatedAt = time.Now().Unix()
}

func (m *AIGCAssetGroupMapping) BeforeUpdate() {
	m.UpdatedAt = time.Now().Unix()
}

// GetUserAIGCAssetGroupMapping 获取用户默认 AIGC Channel 映射
func GetUserAIGCAssetGroupMapping(userId int) (*AIGCAssetGroupMapping, error) {
	mapping := &AIGCAssetGroupMapping{}
	err := DB.Where("user_id = ?", userId).First(mapping).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return mapping, nil
}

// GetAIGCAssetGroupMapping 按 user_id + group_id 查询映射（用于判重）
func GetAIGCAssetGroupMapping(userId int, groupId string) (*AIGCAssetGroupMapping, error) {
	mapping := &AIGCAssetGroupMapping{}
	err := DB.Where("user_id = ? AND group_id = ?", userId, groupId).First(mapping).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return mapping, nil
}

// GetUserAIGCGroupIDs 获取用户拥有的所有 AIGC Group IDs（用于查询过滤）
func GetUserAIGCGroupIDs(userId int) ([]string, error) {
	var groupIDs []string
	err := DB.Model(&AIGCAssetGroupMapping{}).
		Where("user_id = ?", userId).
		Select("group_id").
		Find(&groupIDs).Error
	return groupIDs, err
}

// InsertAIGCAssetGroupMapping 插入用户映射
func InsertAIGCAssetGroupMapping(mapping *AIGCAssetGroupMapping) error {
	return DB.Create(mapping).Error
}

// DeleteAIGCAssetGroupMapping 删除用户与 Asset Group 的映射
func DeleteAIGCAssetGroupMapping(userId int, groupId string) error {
	return DB.Where("user_id = ? AND group_id = ?", userId, groupId).Delete(&AIGCAssetGroupMapping{}).Error
}
