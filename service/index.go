package service

import (
	"fmt"
	"ginchat/models"
	"html/template"
	"strconv"

	"github.com/gin-gonic/gin"
)

// GetIndex
// @Tags 首页
// @Success 200 {string} welcome
// @Router /index [get]
func GetIndex(c *gin.Context) {
	ind, err := template.ParseFiles("index.html", "views/chat/head.html")
	if err != nil {
		panic(err)
	}
	ind.Execute(c.Writer, "index")
	/* c.JSON(200, gin.H{
		"message": "welcome",
	}) */
}

func ToRegister(c *gin.Context) {
	ind, err := template.ParseFiles("views/user/register.html")
	fmt.Println("进入到register页面", ind)
	if err != nil {
		panic(err)
	}
	ind.Execute(c.Writer, "register")
	/* c.JSON(200, gin.H{
		"message": "welcome",
	}) */
}

func ToChat(c *gin.Context) {
	fmt.Println("进入到ToChat")
	ind, err := template.ParseFiles("views/chat/index.html",
		"views/chat/head.html",
		"views/chat/foot.html",
		"views/chat/tabmenu.html",
		"views/chat/concat.html",
		"views/chat/group.html",
		"views/chat/profile.html",
		"views/chat/main.html",
		"views/chat/createcom.html")
	if err != nil {
		panic(err)
	}
	userId, _ := strconv.Atoi(c.Query("userId"))
	token := c.Query("token")
	user := models.UserBasic{}
	user.ID = uint(userId)
	user.Identity = token
	fmt.Println("ToChat>>>>>>>>", user)
	ind.Execute(c.Writer, user)
}

func Chat(c *gin.Context) {
	models.Chat(c.Writer, c.Request)
}
