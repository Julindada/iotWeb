package mqtt

import (
	"encoding/json"
	"iotWeb/model"
	"log"
	"math/rand"
	"strings"
	"time"

	MQTT "github.com/eclipse/paho.mqtt.golang"
)

type Message struct {
	ID   uint
	Data float64
}

type OnLineNode struct {
	model.Node
	FreshData *FreshData
}

type FreshData struct {
	Data    float64
	IsFresh bool
	NodeID  uint
	timer   *time.Timer
}

var client MQTT.Client
var ch chan int
var OnLineNodeMap map[uint]*OnLineNode
var NodeFreshData map[uint]*FreshData

func messageHandler(client MQTT.Client, msg MQTT.Message) {
	log.Println("TOPIC:", msg.Topic())
	log.Println("MSG:", string(msg.Payload()))

	mp := msg.Payload()
	for i, b := range mp {
		if b == '#' {
			mp[i] = ','
		}
	}

	log.Println(string(mp))

	var m Message
	dec := json.NewDecoder(strings.NewReader(string(mp)))
	if err := dec.Decode(&m); err != nil {
		log.Println(err)
		return
	}
	switch msg.Topic() {
	// case "register":
	//     if OnLineNodeMap[m.ID] == nil {
	//         OnLineNodeMap[m.ID] = model.GetNodeByID(m.ID)
	//         NodeFreshData[m.ID] = NewFreshData()
	//     }
	//     // model.AddNode()
	case "message":
		if OnLineNodeMap[m.ID] == nil {
			if n := model.GetNodeByID(m.ID); n != nil {
				OnLineNodeMap[m.ID] = NewOnLineNode(n)
			} else {
				return
			}
		}
		OnLineNodeMap[m.ID].InsertData(m.Data)
		OnLineNodeMap[m.ID].FreshData.Updata(m.Data)
		log.Println(m.ID, m.Data)
	case "":
	}
}

const (
	mqttUrl = "tcp://120.77.86.243:1883"
)

func init() {

	ch = make(chan int)
	OnLineNodeMap = make(map[uint]*OnLineNode)

	opts := MQTT.NewClientOptions().AddBroker(mqttUrl)
	opts.SetClientID("server")
	opts.SetDefaultPublishHandler(messageHandler)

	client = MQTT.NewClient(opts)

	go func() {
		if token := client.Connect(); token.Wait() && token.Error() != nil {
			panic(token.Error())
			return

		}

		if token := client.Subscribe("message", 0, nil); token.Wait() && token.Error() != nil {
			log.Println(token.Error())
			return

		}
	}()

	go func() {
		for {
			<-ch
		}
	}()

	for i := 0; i < 100; i++ {
		go test(uint(i))
	}
}
func test(id uint) {
	if n := model.GetNodeByID(id); n != nil {
		OnLineNodeMap[id] = NewOnLineNode(n)
	} else {
		return
	}
	for {
		OnLineNodeMap[id].FreshData.Updata((10 + float64(rand.Float64())))
		time.Sleep(time.Second * 1)
	}
}

//获得在线与离线节点
func GetNodes(p *model.Park) ([]*model.Node, []*model.Node) {
	nodes := p.GetNodes()
	offlinenodes := make([]*model.Node, 0)
	onlinenodes := make([]*model.Node, 0)
	for key, n := range p.Nodes {
		//说明不在线
		if OnLineNodeMap[n.ID] == nil {
			offlinenodes = append(offlinenodes, &nodes[key])
		} else {
			onlinenodes = append(onlinenodes, &nodes[key])
		}
	}
	return onlinenodes, offlinenodes
}

func NewOnLineNode(n *model.Node) *OnLineNode {
	oln := &OnLineNode{FreshData: NewFreshData(n.ID)}
	oln.Node = *n
	return oln
}

func NewFreshData(id uint) *FreshData {
	f := &FreshData{timer: time.NewTimer(time.Second * 5), NodeID: id}
	f.IsFresh = true
	go func() {
		for {
			<-f.timer.C
			f.IsFresh = false
			delete(OnLineNodeMap, f.NodeID)
			return
		}
	}()
	return f
}

func (n *OnLineNode) InsertData(v float64) {
	n.Node.InsertData(v)
}

func (n *OnLineNode) UpdateNode() {
	n.Node = *model.GetNodeByID(n.ID)
}

func (f *FreshData) Updata(v float64) {
	f.Data = v
	f.timer.Reset(time.Second * 5)
	f.IsFresh = true
}

func Subscribe(topic string, qos byte) {
	if token := client.Subscribe(topic, qos, nil); token.Wait() && token.Error() != nil {
		log.Println(token.Error())
	}
}

func Unsubscribe(topic string) {
	if token := client.Unsubscribe(topic); token.Wait() && token.Error() != nil {
		log.Println(token.Error())
	}
}

func SendOn() {
	if token := client.Publish("com", 1, false, "on"); token.Wait() && token.Error() != nil {
		log.Println(token.Error())
	}
}

func SendOff() {
	if token := client.Publish("com", 1, false, "off"); token.Wait() && token.Error() != nil {
		log.Println(token.Error())
	}
}

func Close() {
	ch <- 0
}
