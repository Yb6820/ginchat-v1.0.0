package models

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"sync"

	"github.com/gorilla/websocket"
	"gopkg.in/fatih/set.v0"
	"gorm.io/gorm"

	"ginchat/utils"
)

// 消息
type Message struct {
	gorm.Model
	FromId   uint   //发送者
	TargetId uint   //接收者
	Type     int    //发送类型  1私聊 2群聊 3广播
	Media    int    //消息类型  文字，图片 ，音频
	Context  string `json:"Content"` //消息内容(前端字段名为Content,加tag对齐)
	Pic      string
	Url      string
	Desc     string
	Amount   int  //其他数字统计
	UserId   uint `json:"userId" gorm:"-"` //前端透传的发送者ID,用于群聊分发,不持久化
}

func (table *Message) TableName() string {
	return "message"
}

type Node struct {
	Conn      *websocket.Conn
	DataQueue chan []byte
	GroupSets set.Interface
}

// 映射关系
var clientMap map[int64]*Node = make(map[int64]*Node, 0)

// 读写锁
var rwLocker sync.RWMutex

func Chat(writer http.ResponseWriter, request *http.Request) {
	//1、获取参数并检验token   合法性
	query := request.URL.Query()
	Id := query.Get("userId")
	userId, _ := strconv.ParseInt(Id, 10, 64)
	//token:=query.Get("token")
	//targetId := query.Get("targetId")
	//context := query.Get("context")
	//msgtype := query.Get("type")
	isvalida := true //checkToke()  待......
	conn, err := (&websocket.Upgrader{
		//token权限校验
		CheckOrigin: func(r *http.Request) bool {
			return isvalida
		},
	}).Upgrade(writer, request, nil)
	if err != nil {
		fmt.Println(err)
		return
	}
	//2、获取链接
	node := &Node{
		Conn:      conn,
		DataQueue: make(chan []byte, 50),
		GroupSets: set.New(set.ThreadSafe),
	}
	//3、用户关系(加载用户加入的群,用于群聊消息分发)
	comIds := SearchCommunityIds(uint(userId))
	for _, v := range comIds {
		node.GroupSets.Add(v)
	}

	//4、userid和node绑定 并加锁
	rwLocker.Lock()
	clientMap[userId] = node
	rwLocker.Unlock()
	//5、完成发送逻辑
	go sendProc(node)
	//6、完成接收逻辑
	go resvProc(node)
	sendMsg(uint(userId), []byte("欢迎进入聊天室"))
}

func sendProc(node *Node) {
	for {
		select {
		case data := <-node.DataQueue:
			fmt.Println("[ws]sendProc >>>> msg:", string(data))
			err := node.Conn.WriteMessage(websocket.TextMessage, data)
			if err != nil {
				fmt.Println(err)
				return
			}
		}
	}
}

func resvProc(node *Node) {
	for {
		_, data, err := node.Conn.ReadMessage()
		if err != nil {
			fmt.Println(err)
			return
		}
		broadMsg(data)
		fmt.Println("[ws]recvProc data<<<<<<<<", string(data))
	}
}

var udpsendChan chan []byte = make(chan []byte, 1024)

func broadMsg(data []byte) {
	udpsendChan <- data
}
func init() {
	fmt.Println("init goroutine")
	go udpSendProc()
	go udpRecvProc()
	fmt.Println("init goroutine")
}

// 完成udp数据发送携程
func udpSendProc() {
	con, err := net.DialUDP("udp", nil, &net.UDPAddr{
		IP:   net.IPv4(192, 168, 0, 255),
		Port: 3000,
	})
	if err != nil {
		fmt.Println(err)
		return
	}
	defer con.Close()
	for {
		select {
		case data := <-udpsendChan:
			fmt.Println("udpSendProc data", string(data))
			_, err := con.Write(data)
			if err != nil {
				fmt.Println(err)
				return
			}
		}
	}
}

// 完成udp数据接收携程
func udpRecvProc() {
	con, err := net.ListenUDP("udp", &net.UDPAddr{
		IP:   net.IPv4zero,
		Port: 3000,
	})
	if err != nil {
		fmt.Println(err)
		return
	}
	defer con.Close()
	for {
		var buf [512]byte
		n, err := con.Read(buf[0:])
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Println("udpRecvProc data:", string(buf[0:n]))
		dispatch(buf[0:n])
	}
}

// 后端调度逻辑
func dispatch(data []byte) {
	msg := Message{}
	err := json.Unmarshal(data, &msg)
	if err != nil {
		fmt.Println(err)
		return
	}
	switch msg.Type {
	case 1: //发送私信
		fmt.Println("dispatch data:", string(data))
		sendMsg(msg.TargetId, data)
		saveMsg(msg)
	case 2: //发送群聊
		fmt.Println("dispatch group data:", string(data))
		sendGroupMsg(msg.UserId, msg.TargetId, data)
		saveMsg(msg)
		/* case 3:
			sendAllMsg()
		case 4: */

	}
}

func sendMsg(userId uint, msg []byte) {
	fmt.Println("sendMsg >>> userID", userId, "  msg:", string(msg))
	rwLocker.RLock()
	node, ok := clientMap[int64(userId)]
	rwLocker.RUnlock()
	if ok {
		node.DataQueue <- msg
	}
}

// sendGroupMsg 群聊消息分发给群内除发送者以外的所有在线成员
func sendGroupMsg(fromId uint, groupId uint, msg []byte) {
	fmt.Println("sendGroupMsg >>> groupID", groupId, "  msg:", string(msg))
	//先筛选出群内在线成员的节点,避免持锁投递
	nodes := make([]*Node, 0)
	rwLocker.RLock()
	for userId, node := range clientMap {
		//自己发的消息前端已本地渲染,无需回发
		if userId == int64(fromId) {
			continue
		}
		//仅发给加入了该群的成员
		if node.GroupSets.Has(groupId) {
			nodes = append(nodes, node)
		}
	}
	rwLocker.RUnlock()
	for _, node := range nodes {
		node.DataQueue <- msg
	}
}

// saveMsg 消息落库,用于刷新后加载历史记录(私聊/群聊通用)
func saveMsg(msg Message) {
	m := Message{
		FromId:   msg.UserId, //前端透传的发送者ID
		TargetId: msg.TargetId,
		Type:     msg.Type,
		Media:    msg.Media,
		Context:  msg.Context,
		Url:      msg.Url,
		Amount:   msg.Amount,
	}
	if err := utils.DB.Create(&m).Error; err != nil {
		fmt.Println("saveMsg err:", err)
	}
}

// LoadGroupMessages 查询群聊历史消息(最近limit条,按时间升序)
func LoadGroupMessages(groupId uint, limit int) []Message {
	msgs := make([]Message, 0)
	utils.DB.Where("target_id = ? and type = 2", groupId).
		Order("id desc").Limit(limit).Find(&msgs)
	// 反转为时间升序,便于前端按序渲染
	for i, j := 0, len(msgs)-1; i < j; i, j = i+1, j-1 {
		msgs[i], msgs[j] = msgs[j], msgs[i]
	}
	// 补充前端渲染所需的发送者userId字段
	for i := range msgs {
		msgs[i].UserId = msgs[i].FromId
	}
	return msgs
}

// LoadPrivateMessages 查询两人私聊历史消息(双向合并,最近limit条,按时间升序)
func LoadPrivateMessages(userId uint, targetId uint, limit int) []Message {
	msgs := make([]Message, 0)
	utils.DB.Where("type = 1 and ((from_id = ? and target_id = ?) or (from_id = ? and target_id = ?))",
		userId, targetId, targetId, userId).
		Order("id desc").Limit(limit).Find(&msgs)
	// 反转为时间升序,便于前端按序渲染
	for i, j := 0, len(msgs)-1; i < j; i, j = i+1, j-1 {
		msgs[i], msgs[j] = msgs[j], msgs[i]
	}
	// 补充前端渲染所需的发送者userId字段
	for i := range msgs {
		msgs[i].UserId = msgs[i].FromId
	}
	return msgs
}
