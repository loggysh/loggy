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
	appid, err := client.InsertApplication(context.Background(), &pb.Application{
		PackageName: "sh.loggy.android",
		Name:        "Loggy",
		Icon:        "loggy.svg",
	})
	if err != nil {
		log.Fatalf("failed to add app: %s", err)
	}

	fmt.Printf("Application ID: %s\n", appid)
}
