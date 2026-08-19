package models

import (
	"fmt"

	"ginchat/utils"

	"gorm.io/gorm"
)

type Community struct {
	gorm.Model
	Name    string
	OwnerId uint
	Img     string
	Desc    string
}

func (table *Community) TableName() string {
	return "community"
}

// CreateCommunity 建群,并把创建者写入群关系
func CreateCommunity(community Community) (int, string) {
	if len(community.Name) == 0 {
		return -1, "群名不能为空"
	}
	if community.OwnerId == 0 {
		return -1, "请先登录"
	}
	user := UserBasic{}
	utils.DB.Where("id = ?", community.OwnerId).First(&user)
	if user.ID == 0 {
		return -1, "用户不存在"
	}
	if err := utils.DB.Create(&community).Error; err != nil {
		fmt.Println(err)
		return -1, "建群失败"
	}
	//创建者默认入群
	if err := addGroupRelation(community.OwnerId, community.ID); err != nil {
		fmt.Println(err)
	}
	return 0, "建群成功"
}

// JoinCommunity 加群
func JoinCommunity(userId uint, comId uint) (int, string) {
	if comId == 0 {
		return -1, "群ID不能为空"
	}
	com := Community{}
	utils.DB.Where("id = ?", comId).First(&com)
	if com.ID == 0 {
		return -1, "该群不存在"
	}
	contact := Contact{}
	utils.DB.Where("owner_id = ? and target_id = ? and type = 2", userId, comId).Find(&contact)
	if contact.ID != 0 {
		return -1, "已加入该群,请不要重复添加"
	}
	if err := addGroupRelation(userId, comId); err != nil {
		fmt.Println(err)
		return -1, "加群失败"
	}
	return 0, "加群成功"
}

// LoadCommunity 查询用户加入的所有群
func LoadCommunity(ownerId uint) ([]*Community, string) {
	//先查用户的群关系,再批量取群信息
	objIds := SearchCommunityIds(ownerId)
	data := make([]*Community, 0)
	if len(objIds) > 0 {
		utils.DB.Where("id in ?", objIds).Find(&data)
	}
	return data, "查询列表成功"
}

// SearchCommunityIds 查询用户加入的所有群ID
func SearchCommunityIds(userId uint) []uint {
	contacts := make([]Contact, 0)
	objIds := make([]uint, 0)
	utils.DB.Where("owner_id = ? and type = 2", userId).Find(&contacts)
	for _, v := range contacts {
		objIds = append(objIds, v.TargetId)
	}
	return objIds
}

// addGroupRelation 建立用户与群的关系,用户在线时同步更新其群组集合
func addGroupRelation(userId uint, comId uint) error {
	contact := Contact{}
	contact.OwnerId = userId
	contact.TargetId = comId
	contact.Type = 2
	if err := utils.DB.Create(&contact).Error; err != nil {
		return err
	}
	//用户在线则实时更新群组集合,保证新群的群消息能即时送达
	rwLocker.RLock()
	node, ok := clientMap[int64(userId)]
	rwLocker.RUnlock()
	if ok {
		node.GroupSets.Add(comId)
	}
	return nil
}
