package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	"golang.org/x/net/context"
	"google.golang.org/grpc"

	pb "github.com/tuxcanfly/loggy/loggy"
)

func main() {
	instanceid := flag.String("instanceid", "", "Instance id to log. (Required)")
	server := flag.String("server", "localhost", "Server to connecto. (localhost)")
	flag.Parse()

	if *instanceid == "" {
		flag.PrintDefaults()
		os.Exit(1)
	}

	conn, err := grpc.Dial(fmt.Sprintf("%s:50111", *server), grpc.WithInsecure())
	if err != nil {
		log.Fatalf("failed to connect: %s", err)
	}
	defer conn.Close()

	client := pb.NewLoggyServiceClient(conn)

	instance, err := client.GetInstance(context.Background(), &pb.InstanceId{Id: *instanceid})
	if err != nil {
		log.Fatalf("failed to get instance: %s", err)
	}

	fmt.Println(instance)

	app, err := client.GetApplication(context.Background(), &pb.ApplicationId{Id: instance.Appid})
	if err != nil {
		log.Fatalf("failed to app: %s", err)
	}

	fmt.Println(app)

	device, err := client.GetDevice(context.Background(), &pb.DeviceId{Id: instance.Deviceid})
	if err != nil {
		log.Fatalf("failed to device: %s", err)
	}

	fmt.Println(device)

	receiverid, err := client.Register(context.Background(), &pb.InstanceId{Id: *instanceid})
	if err != nil {
		log.Fatalf("failed to register: %s", err)
	}

	stream, err := client.Receive(context.Background(), receiverid)
	if err != nil {
		log.Fatalf("failed to receive: %s", err)
	}

	log.Printf("Started logger for instance: %s\n", *instanceid)
	for {
		in, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("failed to connect: %s", err)
		}
		log.Printf("Instance: %s, Session: %s: %s\n", in.Instanceid, in.Sessionid, in.Msg)
	}
	stream.CloseSend()
}
