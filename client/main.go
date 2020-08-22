package main

import (
	"log"
	"time"

	"golang.org/x/net/context"
	"google.golang.org/grpc"

	pb "github.com/tuxcanfly/loggy/loggy"
)

func main() {
	conn, err := grpc.Dial("localhost:50111", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("failed to connect: %s", err)
	}
	defer conn.Close()

	client := pb.NewLoggyServiceClient(conn)
	appid, err := client.InsertApplication(context.Background(), &pb.Application{
		Id:   "com.swiggy.android",
		Name: "Swiggy",
		Icon: "swiggy.svg",
	})
	if err != nil {
		log.Fatalf("failed to add app: %s", err)
	}

	deviceid, err := client.InsertDevice(context.Background(), &pb.Device{
		Details: map[string]string{"name": "Xiaomi Note 5"},
	})
	if err != nil {
		log.Fatalf("failed to add app: %s", err)
	}

	instanceid, err := client.InsertInstance(context.Background(), &pb.Instance{
		Deviceid: deviceid.Id,
		Appid:    appid.Id,
	})
	if err != nil {
		log.Fatalf("failed to add app: %s", err)
	}

	stream, err := client.Send(context.Background())
	waitc := make(chan struct{})

	go func() {
		for {
			time.Sleep(time.Second)
			msg := &pb.LoggyMessage{Instanceid: instanceid.Id, Msg: time.Now().Format(time.RFC3339Nano)}
			log.Printf("%s: %q\n", msg.Instanceid, msg.Msg)
			stream.Send(msg)
		}
	}()
	<-waitc
	stream.CloseSend()
}
