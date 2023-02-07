package models

import (
	"fmt"
	"ginchat/utils"

	"gorm.io/gorm"
)

// 人员关系
type Contact struct {
	gorm.Model
	OwnerId  uint //谁得关系信息
	TargetId uint //对应的谁
	Type     int  //对应的类型 1好友  2群  3
	Desc     string
}

func (table *Contact) TableName() string {
	return "contact"
}

func SearchFriends(userId uint) []UserBasic {
	contacts := make([]Contact, 0)
	objIds := make([]uint, 0)
	utils.DB.Where("owner_id = ? and type = 1", userId).Find(&contacts)
	for _, v := range contacts {
		fmt.Println(v)
		objIds = append(objIds, v.TargetId)
	}
	users := make([]UserBasic, 0)
	utils.DB.Where("id in ?", objIds).Find(&users)
	return users
}

func AddFriend(userId uint, targetId uint) (int, string) {
	user := UserBasic{}
	if targetId != 0 {
		user = FindUserByID(targetId)
		if user.Identity != "" {
			if userId == user.ID {
				return -1, "不能添加自己"
			}
			contact0 := Contact{}
			utils.DB.Where("owner_id = ? and target_id = ? and type = 1", userId, targetId).Find(&contact0)
			if contact0.ID != 0 {
				return -1, "不能重复添加"
			}
			//事务插入
			tx := utils.DB.Begin()
			//事务一旦开始，不论遇到什么问题都会Rollback
			defer func() {
				if r := recover(); r != nil {
					tx.Rollback()
				}
			}()
			contact := Contact{}
			contact.OwnerId = userId
			contact.TargetId = targetId
			contact.Type = 1
			if err := utils.DB.Create(&contact).Error; err != nil {
				tx.Rollback()
				return -1, "添加好友失败"
			}
			paircontact := Contact{}
			paircontact.OwnerId = targetId
			paircontact.TargetId = userId
			paircontact.Type = 1
			if err := utils.DB.Create(&paircontact).Error; err != nil {
				tx.Rollback()
				return -1, "添加好友失败"
			}
			tx.Commit()
			return 0, "添加好友成功"
		}
		return -1, "找不到该用户"
	}
	return -1, "好友ID不能为空"
}
