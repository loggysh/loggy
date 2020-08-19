package main

import (
	"io"
	"log"
	"net"

	"google.golang.org/grpc"

	pb "github.com/tuxcanfly/loggy/simple"
)

type simpleServer struct {
}

func (s *simpleServer) SimpleRPC(stream pb.SimpleService_SimpleRPCServer) error {
	log.Println("Started stream")
	for {
		in, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		log.Printf("%d: %s\n", in.Id, in.Msg)
	}
}

func main() {
	grpcServer := grpc.NewServer()
	pb.RegisterSimpleServiceServer(grpcServer, &simpleServer{})

	l, err := net.Listen("tcp", ":50111")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	log.Println("Listening on tcp://localhost:50111")
	grpcServer.Serve(l)
}
