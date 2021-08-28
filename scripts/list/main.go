package main

import (
	"flag"
	"fmt"
	"log"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	pb "github.com/tuxcanfly/loggy/loggy"
)

func main() {
	sessionid := flag.Int("sessionid", -1, "required Session id")
	userid := flag.String("userid", "", "required User id")
	authorization := flag.String("authorization", "", "required Authorization")
	url := flag.String("url", "localhost:50111", "Url")
	flag.Parse()

	if *sessionid == -1 || *authorization == "" || *userid == "" {
		flag.PrintDefaults()
		return
	}

	conn, err := grpc.Dial(*url, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("failed to connect: %s", err)
	}
	defer conn.Close()
	header := metadata.New(map[string]string{"authorization": *authorization, "user_id": *userid})
	ctx := metadata.NewOutgoingContext(context.Background(), header)
	client := pb.NewLoggyServiceClient(conn)
	messageList, err := client.ListSessionMessages(ctx, &pb.SessionId{
		Id: int32(*sessionid),
	})
	if err != nil {
		log.Fatalf("failed to search: %s", err)
	}

	for _, i := range messageList.Messages {
		fmt.Println(i)
	}
}
