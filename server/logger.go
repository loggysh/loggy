package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"path"

	"golang.org/x/net/context"
	"google.golang.org/grpc"

	"github.com/blevesearch/bleve"
	"github.com/golang/protobuf/ptypes/empty"
	pb "github.com/tuxcanfly/loggy/loggy"
)

func logger(prefix, server *string, indexer bleve.Index) {
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

		logfilepath := path.Join(*prefix, app.Id, device.Id, instance.Id)
		err = os.MkdirAll(logfilepath, 0777)
		if err != nil {
			log.Printf("failed to mkdir: %s", err)
		}

		log.Println(logfilepath)

		go func(instance *pb.Instance, app *pb.Application, device *pb.Device, receiverid *pb.ReceiverId,
			logfilepath string, indexer bleve.Index) {
			stream, err := client.Receive(context.Background(), receiverid)
			if err != nil {
				log.Printf("failed to receive: %s", err)
			}

			log.Printf("Started logger for instance: %s\n", instance.Id)

			logfile, err := os.OpenFile(path.Join(logfilepath, fmt.Sprintf("%s.txt", instance.Id)), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				log.Printf("failed to open file: %s", err)
			}
			defer logfile.Close()

			var linenumber int
			for {
				in, err := stream.Recv()
				if err == io.EOF {
					break
				}
				if err != nil {
					log.Printf("failed to connect: %s", err)
				}
				linenumber++
				logline := fmt.Sprintf("%v: level = %v, app = %v; device = %v; msg = %v\n",
					in.Timestamp.AsTime(), in.Level, app, device, in.Msg)
				log.Printf(logline)
				indexer.Index(fmt.Sprintf("%s: %d", instance.Id, linenumber), logline)
				logfile.WriteString(logline)
			}
			stream.CloseSend()
		}(instance, app, device, receiverid, logfilepath, indexer)
		stream.CloseSend()
	}
}
