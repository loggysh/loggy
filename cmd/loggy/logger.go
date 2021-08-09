package main

import (
	"fmt"
	"io"
	"log"

	"github.com/tuxcanfly/loggy/service"

	"golang.org/x/net/context"
	"google.golang.org/grpc"

	"github.com/blevesearch/bleve"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/jinzhu/gorm"
	pb "github.com/tuxcanfly/loggy/loggy"
)

func logger(prefix, server *string, indexer bleve.Index, db *gorm.DB) {
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
		session, err := stream.Recv()
		print(session)
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Printf("failed to receive session: %s", err)
		}
		log.Println(session)

		receiverid, err := client.RegisterReceive(context.Background(), &pb.SessionId{Id: session.Id})
		if err != nil {
			log.Printf("failed to register receive: %s", err)
		}

		log.Println(receiverid)

		go func(session *pb.Session, receiverid *pb.ReceiverId, indexer bleve.Index, db *gorm.DB) {
			stream, err := client.Receive(context.Background(), receiverid)
			if err != nil {
				log.Printf("failed to receive: %s", err)
			}

			log.Printf("Started logger for session: %d\n", session.Id)

			for {
				in, err := stream.Recv()
				if err == io.EOF {
					break
				}
				if err != nil {
					log.Printf("failed to connect: %s", err)
				}
				msg := service.Message{
					SessionID: in.Sessionid,
					Msg:       in.Msg,
					Timestamp: in.Timestamp.AsTime(),
					Level:     service.LogLevel(in.Level),
				}
				if db.Create(&msg).Error != nil {
					log.Println("unable to create message")
					continue
				}
				log.Println(msg.String())
				indexer.Index(fmt.Sprintf("%d", msg.ID), in)
			}
			stream.CloseSend()
		}(session, receiverid, indexer, db)
		stream.CloseSend()
	}
}
