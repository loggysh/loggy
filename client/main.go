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

	appClient := pb.NewApplicationServiceClient(conn)
	appId, err := appClient.Insert(context.Background(), &pb.Application{
		Id:   "com.swiggy.android",
		Name: "Swiggy",
		Icon: "swiggy.svg",
	})
	if err != nil {
		log.Fatalf("failed to add app: %s", err)
	}

	deviceClient := pb.NewDeviceServiceClient(conn)
	deviceId, err := deviceClient.Insert(context.Background(), &pb.Device{
		Details: "Xiaomi Note 5",
	})
	if err != nil {
		log.Fatalf("failed to add app: %s", err)
	}

	instanceClient := pb.NewInstanceServiceClient(conn)
	instanceID, err := instanceClient.Insert(context.Background(), &pb.Instance{
		Deviceid: deviceId.Id,
		Appid:    appId.Id,
	})
	if err != nil {
		log.Fatalf("failed to add app: %s", err)
	}

	client := pb.NewLoggyServiceClient(conn)
	stream, err := client.LoggyServer(context.Background())
	waitc := make(chan struct{})

	go func() {
		for {
			time.Sleep(time.Second)
			msg := &pb.LoggyMessage{Id: instanceID.Id, Msg: time.Now().Format(time.RFC3339Nano)}
			log.Printf("%d: %q\n", msg.Id, msg.Msg)
			stream.Send(msg)
		}
	}()
	<-waitc
	stream.CloseSend()
}
