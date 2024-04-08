package main

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"github.com/charmbracelet/log"
	pb "github.com/meshnet-gophers/meshtastic-go/meshtastic"
	"github.com/meshnet-gophers/meshtastic-go/mqtt"
	"github.com/meshnet-gophers/meshtastic-go/radio"
	"google.golang.org/protobuf/proto"
	"strings"
)

func main() {
	client := mqtt.NewClient("tcp://mqtt.meshtastic.org:1883", "meshdev", "large4cats", "msh")
	err := client.Connect()
	if err != nil {
		log.Fatal(err)
	}
	client.Handle("LongFast", channelHandler("LongFast"))
	log.Info("Started")
	select {}
}

func channelHandler(channel string) mqtt.HandlerFunc {
	return func(m mqtt.Message) {
		var env pb.ServiceEnvelope
		err := proto.Unmarshal(m.Payload, &env)
		if err != nil {
			log.Fatal("failed unmarshalling to service envelope", "err", err, "payload", hex.EncodeToString(m.Payload))
			return
		}

		key, err := generateKey("1PG7OiApB1nwvP+rz05pAQ==")
		if err != nil {
			log.Fatal(err)
		}

		decodedMessage, err := radio.XOR(env.Packet.GetEncrypted(), key, env.Packet.Id, env.Packet.From)
		if err != nil {
			log.Error(err)
		}
		var message pb.Data
		err = proto.Unmarshal(decodedMessage, &message)

		log.Info(processMessage(message), "topic", m.Topic, "channel", channel, "portnum", message.Portnum.String())
	}
}

func processMessage(message pb.Data) string {
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

	return fmt.Sprintf("unknown message type")
}

func generateKey(key string) ([]byte, error) {
	// Pad the key with '=' characters to ensure it's a valid base64 string
	padding := (4 - len(key)%4) % 4
	paddedKey := key + strings.Repeat("=", padding)

	// Replace '-' with '+' and '_' with '/'
	replacedKey := strings.ReplaceAll(paddedKey, "-", "+")
	replacedKey = strings.ReplaceAll(replacedKey, "_", "/")

	// Decode the base64-encoded key
	return base64.StdEncoding.DecodeString(replacedKey)
}
