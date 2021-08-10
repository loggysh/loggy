package main

import (
	"flag"
	"fmt"
	"log"
	"io"

	"google.golang.org/grpc/metadata"
	"golang.org/x/net/context"
	"google.golang.org/grpc"

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
	receiverId, err := client.RegisterReceive(ctx, &pb.SessionId{
		Id: int32(*sessionid),
	})

	receive, err := client.Receive(ctx, receiverId)
	if err != nil {
		log.Fatalf("failed to search: %s", err)
	}

	for {
		in, err := receive.Recv()
		if err == io.EOF {
			fmt.Printf("EOF stream")
			return
		}
		if err != nil {
			log.Fatalf("stream failed: %s", err)
		}
		s := fmt.Sprintf("session id: %s", in.Sessionid)
		m := fmt.Sprintf("msg: %s", in.Msg)

		fmt.Println(s)
		fmt.Println(m)
	}

}
