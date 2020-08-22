package main

import (
	"fmt"
	"io"
	"log"

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

	instanceid := "1"
	client := pb.NewLoggyServiceClient(conn)

	instance, err := client.GetInstance(context.Background(), &pb.InstanceId{Id: instanceid})
	if err != nil {
		log.Fatalf("failed to connect: %s", err)
	}

	fmt.Println(instance)

	app, err := client.GetApplication(context.Background(), &pb.ApplicationId{Id: instance.Appid})
	if err != nil {
		log.Fatalf("failed to connect: %s", err)
	}

	fmt.Println(app)

	device, err := client.GetDevice(context.Background(), &pb.DeviceId{Id: instance.Deviceid})
	if err != nil {
		log.Fatalf("failed to connect: %s", err)
	}

	fmt.Println(device)

	receiverid, err := client.Register(context.Background(), &pb.InstanceId{Id: instanceid})
	if err != nil {
		log.Fatalf("failed to connect: %s", err)
	}

	stream, err := client.Receive(context.Background(), receiverid)
	if err != nil {
		log.Fatalf("failed to connect: %s", err)
	}

	log.Printf("Started logger for instance: %s\n", instanceid)
	for {
		in, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("failed to connect: %s", err)
		}
		log.Printf("%s: %s\n", in.Instanceid, in.Msg)
	}
	stream.CloseSend()
}
