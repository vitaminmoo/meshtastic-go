package main

import (
	"context"
	"github.com/charmbracelet/log"
	pb "github.com/meshnet-gophers/meshtastic-go/meshtastic"
	"github.com/meshnet-gophers/meshtastic-go/transport"
	"github.com/meshnet-gophers/meshtastic-go/transport/serial"
	"google.golang.org/protobuf/proto"
	"os"
	"os/signal"
	"time"
)

var port string

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	log.SetLevel(log.DebugLevel)

	if len(os.Args) > 1 {
		port = os.Args[1]
	} else {
		port = serial.GetPorts()[0]
	}
	serialConn, err := serial.Connect(port)
	if err != nil {
		panic(err)
	}
	streamConn, err := transport.NewClientStreamConn(serialConn)
	if err != nil {
		panic(err)
	}
	defer func() {
		if err := streamConn.Close(); err != nil {
			panic(err)
		}
	}()

	client := transport.NewClient(streamConn, false)
	client.Handle(new(pb.MeshPacket), func(msg proto.Message) {
		pkt := msg.(*pb.MeshPacket)
		data := pkt.GetDecoded()
		log.Info("Received message from radio", "msg", processMessage(data), "from", pkt.From, "portnum", data.Portnum.String())
	})
	ctxTimeout, cancelTimeout := context.WithTimeout(ctx, 10*time.Second)
	defer cancelTimeout()
	if client.Connect(ctxTimeout) != nil {
		panic("Failed to connect to the radio")
	}

	log.Info("Waiting for interrupt signal")
	<-ctx.Done()
}

func processMessage(message *pb.Data) string {
	if message.Portnum == pb.PortNum_NODEINFO_APP {
		var user = pb.User{}
		proto.Unmarshal(message.Payload, &user)
		return user.String()
	}
	if message.Portnum == pb.PortNum_POSITION_APP {
		var pos = pb.Position{}
		proto.Unmarshal(message.Payload, &pos)
		return pos.String()
	}
	if message.Portnum == pb.PortNum_TELEMETRY_APP {
		var t = pb.Telemetry{}
		proto.Unmarshal(message.Payload, &t)
		return t.String()
	}
	if message.Portnum == pb.PortNum_NEIGHBORINFO_APP {
		var n = pb.NeighborInfo{}
		proto.Unmarshal(message.Payload, &n)
		return n.String()
	}
	if message.Portnum == pb.PortNum_STORE_FORWARD_APP {
		var s = pb.StoreAndForward{}
		proto.Unmarshal(message.Payload, &s)
		return s.String()
	}

	return "unknown message type"
}
