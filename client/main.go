package main

import (
	"log"
	"time"

	"golang.org/x/net/context"
	"google.golang.org/grpc"

	pb "github.com/tuxcanfly/loggy/simple"
)

func main() {
	conn, err := grpc.Dial("localhost:50111", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("failed to connect: %s", err)
	}
	defer conn.Close()

	client := pb.NewSimpleServiceClient(conn)
	stream, err := client.SimpleRPC(context.Background())
	waitc := make(chan struct{})

	go func() {
		for {
			log.Println("Sleeping...")
			time.Sleep(time.Second)
			msg := &pb.SimpleData{Msg: time.Now().Format(time.RFC3339Nano)}
			log.Printf("msg: %q\n", msg.Msg)
			stream.Send(msg)
		}
	}()
	<-waitc
	stream.CloseSend()
}