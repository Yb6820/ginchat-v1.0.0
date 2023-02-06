package main

//项目入口
import (
	"ginchat/router"
	"ginchat/utils"
)

func main() {
	utils.InitConfig()
	utils.InitMySQL()
	utils.InitRedis()
	r := router.Router()
	r.Run() // 监听并在 0.0.0.0:8080 上启动服务
}
