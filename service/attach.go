package service

import (
	"fmt"
	"ginchat/utils"
	"io"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func Upload(c *gin.Context) {
	writer := c.Writer
	request := c.Request
	srcFile, head, err := request.FormFile("file")
	if err != nil {
		utils.RespFail(writer, err.Error())
	}
	suffix := ".png"
	ofilName := head.Filename
	tem := strings.Split(ofilName, ".")
	if len(tem) > 1 {
		suffix = "." + tem[len(tem)-1]
	}
	fileName := fmt.Sprintf("%d%04d%s", time.Now().Unix(), rand.Int31(), suffix)
	dstFile, err := os.Create("./asset/upload" + fileName)
	if err != nil {
		utils.RespFail(writer, err.Error())
	}
	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		utils.RespFail(writer, err.Error())
	}
	url := "./asset/upload" + fileName
	utils.RespOk(writer, url, "发送图片成功")
}
