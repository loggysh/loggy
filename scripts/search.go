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
	query := flag.String("query", "", "Search query")
	flag.Parse()

	if *query == "" {
		flag.PrintDefaults()
		return
	}

	conn, err := grpc.Dial("localhost:50111", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("failed to connect: %s", err)
	}
	defer conn.Close()

	client := pb.NewLoggyServiceClient(conn)
	results, err := client.Search(context.Background(), &pb.Query{
		Query: *query,
	})
	if err != nil {
		log.Fatalf("failed to search: %s", err)
	}

	fmt.Printf("Search Results: %s\n", results)
}
