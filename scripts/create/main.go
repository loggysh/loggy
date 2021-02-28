package main

import (
	"fmt"
	"log"

	"golang.org/x/net/context"
	"google.golang.org/grpc"

	pb "github.com/tuxcanfly/loggy/loggy"
)

func main() {
	conn, err := grpc.Dial("loggy.sh:50111", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("failed to connect: %s", err)
	}
	defer conn.Close()

	client := pb.NewLoggyServiceClient(conn)
	emailID, err := client.InsertWaitListUser(context.Background(), &pb.WaitListUser{
		Email: "feedback@loggy.sh",
	})
	fmt.Println(emailID)
	if err != nil {
		log.Fatalf("failed to add waitlist user: %s", err)
	}

	fmt.Printf("Email ID: %s\n", emailID)
}
