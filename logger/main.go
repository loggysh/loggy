package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path"

	"golang.org/x/net/context"
	"google.golang.org/grpc"

	"github.com/golang/protobuf/ptypes/empty"
	pb "github.com/tuxcanfly/loggy/loggy"
)

func main() {
	prefix := flag.String("prefix", "logs", "Prefix for logs. (logs)")
	server := flag.String("server", "localhost", "Server to connect to. (localhost)")
	flag.Parse()

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
			log.Fatalf("failed to connect: %s", err)
		}
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

		receiverid, err := client.RegisterReceive(context.Background(), &pb.InstanceId{Id: instance.Id})
		if err != nil {
			log.Fatalf("failed to register: %s", err)
		}

		logfilepath := path.Join(*prefix, app.Id, device.Id, instance.Id)
		err = os.MkdirAll(logfilepath, 0700)
		if err != nil {
			log.Fatalf("failed to mkdir: %s", err)
		}

		go func(instance *pb.Instance, app *pb.Application, device *pb.Device, receiverid *pb.ReceiverId, logfilepath string) {
			stream, err := client.Receive(context.Background(), receiverid)
			if err != nil {
				log.Fatalf("failed to receive: %s", err)
			}

			log.Printf("Started logger for instance: %s\n", instance.Id)

			logfile, err := os.OpenFile(path.Join(logfilepath, "logs.txt"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				log.Fatalf("failed to open file: %s", err)
			}
			defer logfile.Close()

			for {
				in, err := stream.Recv()
				if err == io.EOF {
					break
				}
				if err != nil {
					log.Fatalf("failed to connect: %s", err)
				}
				logline := fmt.Sprintf("%v: level = %v, app = %v; device = %v; msg = %v\n",
					in.Timestamp, in.Level, app, device, in.Msg)
				fmt.Println(logline)
				logfile.WriteString(logline)
			}
			stream.CloseSend()
		}(instance, app, device, receiverid, logfilepath)
		stream.CloseSend()
	}
}
