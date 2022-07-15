package net_rpc

import (
	"github.com/gogo/protobuf/proto"
	logger "github.com/hezhis/go_log"
	"github.com/hezhis/net_rpc/pb"
	"github.com/hezhis/net_rpc/protocol"
	"log"
	"testing"
)

func test(args *pb.ProtoColorGroup) {
	logger.Info("%v", args)
	logger.Info("%d", args.Id)
}

func testSystem(args *pb.ProtoColorGroup) {
	logger.Info("%v", args)
	logger.Info("%d", args.Id)
}

var (
	s      *Server
	dbChan = make(chan *protocol.Message, 100)
)

func reg() {
	s = NewServer()
	s.registerFunction("actorData", test)
	s.registerFunction("systemData", testSystem)
}

func TestNewServer(t *testing.T) {
	reg()

	callOther("systemData", "testSystem", &pb.ProtoColorGroup{
		Id:     123,
		Name:   "测试",
		Colors: []string{"0xFFFFFF"},
	})

	req2 := <-dbChan
	s.DoCall(req2)
}

func callOther(path, method string, data *pb.ProtoColorGroup) {
	req := protocol.NewMessage()
	req.SetSerializeType(protocol.ProtoBuffer)
	req.ServicePath = path
	req.ServiceMethod = method

	if data, err := proto.Marshal(data); nil == err {
		req.Payload = data
	} else {
		log.Println(err)
	}

	dbChan <- req
}

type Draw struct {
	base int32
}

func (m *Draw) Add(data *pb.ProtoColorGroup) {
	data.Id += m.base
	logger.Debug(data.String())
}

func TestServer_DoCall(t *testing.T) {
	s = NewServer()
	s.Register(&Draw{base: 10})

	req := protocol.NewMessage()
	req.SetSerializeType(protocol.ProtoBuffer)
	req.ServicePath = "Draw"
	req.ServiceMethod = "Add"

	if data, err := proto.Marshal(&pb.ProtoColorGroup{
		Id:     123,
		Name:   "测试",
		Colors: []string{"0xFFFFFF"},
	}); nil == err {
		req.Payload = data
	} else {
		log.Println(err)
	}
	s.DoCall(req)
}
