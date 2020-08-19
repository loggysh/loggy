package main

import (
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

	instanceID := int32(1)
	client := pb.NewLoggyServiceClient(conn)
	clientid, err := client.RegisterClient(context.Background(), &pb.InstanceId{Id: instanceID})
	if err != nil {
		log.Fatalf("failed to connect: %s", err)
	}

	stream, err := client.LoggyClient(context.Background(), clientid)
	if err != nil {
		log.Fatalf("failed to connect: %s", err)
	}

	log.Printf("Started logger for instance: %d\n", instanceID)
	for {
		in, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("failed to connect: %s", err)
		}
		log.Printf("%d: %s\n", in.Id, in.Msg)
	}
	stream.CloseSend()
}
