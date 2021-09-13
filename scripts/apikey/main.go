package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"strings"
	"time"

	"google.golang.org/grpc/metadata"

	"golang.org/x/net/context"
	"google.golang.org/grpc"

	"github.com/golang/protobuf/ptypes"
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
	apiClient := flag.String("client", "android", "client")
	apiKey := flag.String("apikey", "", "required api key")
	url := flag.String("url", "localhost:50111", "Url")
	flag.Parse()

	if *apiKey == "" {
		flag.PrintDefaults()
		return
	}

	conn, err := grpc.Dial(*url, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("failed to connect: %s", err)
	}
	defer conn.Close()
	header := metadata.New(map[string]string{"client": *apiClient, "api_key": *apiKey})
	ctx := metadata.NewOutgoingContext(context.Background(), header)
	client := pb.NewLoggyServiceClient(conn)
	app, err := client.GetOrInsertApplication(ctx, &pb.Application{
		Packagename: "sh.loggy",
		Name:        "Loggy",
		Icon:        "loggy.svg",
	})

	if err != nil {
		log.Fatalf("failed to add app: %s", err)
	}

	fmt.Printf("Application ID: %s\n", app.Id)

	device, err := client.GetOrInsertDevice(ctx, &pb.Device{
		Id:      "5b11da9b-35a9-4c87-99b1-def6ca91ace8",
		Appid:   app.Id,
		Details: `{"device_name":"Sample","application_name":"Loggy","application_version":"0.3","android_os_version":"4.14.112+(5891938)","android_api_level":"29","device_type":"generic_x86","device_model":"Android SDK built for x86 sdk_gphone_x86"}`,
	})
	if err != nil {
		log.Fatalf("failed to add device: %s", err)
	}

	fmt.Printf("Device ID: %s\n", device.Id)

	deviceList, err := client.ListDevices(ctx, &pb.ApplicationId{
		Id: app.Id,
	})
	if err != nil {
		log.Fatalf("failed to get device list: %s", err)
	}

	fmt.Printf("Device List: %s\n", deviceList)
	sessionid, err := client.InsertSession(ctx, &pb.Session{
		Deviceid: device.Id,
		Appid:    app.Id,
	})
	if err != nil {
		log.Fatalf("failed to add session: %s", err)
	}

	fmt.Printf("Session ID: %s\n", sessionid)

	streamMessages(client, ctx, sessionid)
}

func streamMessages(client pb.LoggyServiceClient, ctx context.Context, sessionid *pb.SessionId) {
	_, err := client.RegisterSend(ctx, &pb.SessionId{Id: sessionid.Id})
	if err != nil {
		log.Fatalf("failed to register: %s", err)
	}
	stream, _ := client.Send(ctx)
	waitc := make(chan struct{})

	go func() {
		for {
			time.Sleep(time.Second)
			msg := &pb.Message{
				Sessionid: sessionid.Id,
				Msg:       babble(),
				Timestamp: ptypes.TimestampNow(),
			}
			log.Printf("Sesssion - %d: %s\n", msg.Sessionid, msg.Msg)
			stream.Send(msg)
		}
	}()
	<-waitc
	stream.CloseSend()

}
