package main

import (
	"fmt"
	"io"
	"log"

	"golang.org/x/net/context"
	"google.golang.org/grpc"

	"github.com/go-ego/riot"
	"github.com/go-ego/riot/types"
	"github.com/golang/protobuf/ptypes/empty"
	pb "github.com/tuxcanfly/loggy/loggy"
)

func indexer(searcher *riot.Engine, server *string) {
	conn, err := grpc.Dial(fmt.Sprintf("%s:50111", *server), grpc.WithInsecure())
	if err != nil {
		log.Fatalf("failed to connect: %s", err)
	}
	defer conn.Close()

	client := pb.NewLoggyServiceClient(conn)

	stream, err := client.Notify(context.Background(), &empty.Empty{})
	if err != nil {
		log.Fatalf("failed to listen: %s", err)
	}

	for {
		instance, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Printf("failed to receive instance: %s", err)
		}
		log.Println(instance)

		app, err := client.GetApplication(context.Background(), &pb.ApplicationId{Id: instance.Appid})
		if err != nil {
			log.Printf("failed to get app: %s", err)
		}

		log.Println(app)

		device, err := client.GetDevice(context.Background(), &pb.DeviceId{Id: instance.Deviceid})
		if err != nil {
			log.Printf("failed to get device: %s", err)
		}

		log.Println(device)

		receiverid, err := client.RegisterReceive(context.Background(), &pb.InstanceId{Id: instance.Id})
		if err != nil {
			log.Printf("failed to register receive: %s", err)
		}

		log.Println(receiverid)

		go func(instance *pb.Instance, app *pb.Application, device *pb.Device, receiverid *pb.ReceiverId, searcher *riot.Engine) {
			stream, err := client.Receive(context.Background(), receiverid)
			if err != nil {
				log.Printf("failed to receive: %s", err)
			}

			log.Printf("Started searcher for instance: %s", instance.Id)

			var i int
			for {
				in, err := stream.Recv()
				if err == io.EOF {
					break
				}
				if err != nil {
					log.Printf("failed to connect: %s", err)
				}
				searcher.Index(instance.Id, types.DocData{Content: in.Msg})
				i++
			}
			stream.CloseSend()
		}(instance, app, device, receiverid, searcher)
		stream.CloseSend()
	}
}
