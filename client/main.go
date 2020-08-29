package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"strings"
	"time"

	"golang.org/x/net/context"
	"google.golang.org/grpc"

	"github.com/golang/protobuf/ptypes"
	uuid "github.com/satori/go.uuid"
	pb "github.com/tuxcanfly/loggy/loggy"
)

var words []string

func init() {
	rand.Seed(time.Now().UnixNano())
	words = readAvailableDictionary()
}

func readAvailableDictionary() []string {
	file, err := os.Open("/usr/share/dict/words")
	if err != nil {
		log.Fatal(err)
	}

	bytes, err := ioutil.ReadAll(file)
	if err != nil {
		log.Fatal(err)
	}

	return strings.Split(string(bytes), "\n")
}

func babble() string {
	pieces := []string{}
	for i := 0; i < 7; i++ {
		pieces = append(pieces, words[rand.Int()%len(words)])
	}

	return strings.Join(pieces, " ")
}

func main() {
	conn, err := grpc.Dial("localhost:50111", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("failed to connect: %s", err)
	}
	defer conn.Close()

	client := pb.NewLoggyServiceClient(conn)
	appid, err := client.InsertApplication(context.Background(), &pb.Application{
		PackageName: "com.swiggy.android",
		Name:        "Swiggy",
		Icon:        "swiggy.svg",
	})
	if err != nil {
		log.Fatalf("failed to add app: %s", err)
	}

	fmt.Printf("Application ID: %s\n", appid)

	deviceid, err := client.InsertDevice(context.Background(), &pb.Device{
		Id:      uuid.NewV4().String(),
		Details: "{'name': 'Xiaomi Note 5'}",
	})
	if err != nil {
		log.Fatalf("failed to add device: %s", err)
	}

	fmt.Printf("Device ID: %s\n", deviceid)

	instanceid, err := client.GetOrInsertInstance(context.Background(), &pb.Instance{
		Deviceid: deviceid.Id,
		Appid:    appid.Id,
	})
	if err != nil {
		log.Fatalf("failed to add app: %s", err)
	}

	fmt.Printf("Instance ID: %s\n", instanceid)

	_, err = client.RegisterSend(context.Background(), &pb.InstanceId{Id: instanceid.Id})
	if err != nil {
		log.Fatalf("failed to register: %s", err)
	}

	stream, err := client.Send(context.Background())
	waitc := make(chan struct{})

	go func() {
		for {
			time.Sleep(time.Second)
			msg := &pb.LoggyMessage{
				Instanceid: instanceid.Id,
				Sessionid:  uuid.NewV4().String(),
				Msg:        babble(),
				Timestamp:  ptypes.TimestampNow(),
			}
			log.Printf("Instance: %s, Session: %s: %s\n", msg.Instanceid, msg.Sessionid, msg.Msg)
			stream.Send(msg)
		}
	}()
	<-waitc
	stream.CloseSend()
}
