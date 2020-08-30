package main

import (
	"flag"
	"fmt"
	"log"

	"golang.org/x/net/context"
	"google.golang.org/grpc"

	pb "github.com/tuxcanfly/loggy/loggy"
)

func main() {
	sessionid := flag.Int("sessionid", -1, "Session id")
	flag.Parse()

	if *sessionid == -1 {
		flag.PrintDefaults()
		return
	}

	conn, err := grpc.Dial("localhost:50111", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("failed to connect: %s", err)
	}
	defer conn.Close()

	client := pb.NewLoggyServiceClient(conn)
	messages, err := client.ListSessionMessages(context.Background(), &pb.SessionId{
		Id: int32(*sessionid),
	})
	if err != nil {
		log.Fatalf("failed to search: %s", err)
	}

	fmt.Printf("Search Results: %s\n", messages)
}
