package main

import (
	"context"
	"io"
	"log"
	"net"
	"strconv"

	"google.golang.org/grpc"

	pb "github.com/tuxcanfly/loggy/loggy"
)

type application struct {
	id   string
	name string
	icon string
}

type device struct {
	id      int32
	details string
}

type instance struct {
	id       int32
	appid    string
	deviceid int32
}

type loggyServer struct {
	apps      map[string]*pb.Application
	devices   map[string]*pb.Device
	instances map[string]*pb.Instance
	receivers map[string]chan *pb.LoggyMessage
	listeners map[string][]string // instanceid -> []receivers
}

func (l *loggyServer) GetApplication(ctx context.Context, appid *pb.ApplicationId) (*pb.Application, error) {
	if app, ok := l.apps[appid.Id]; ok {
		return app, nil
	}
	return nil, nil
}

func (l *loggyServer) InsertApplication(ctx context.Context, app *pb.Application) (*pb.ApplicationId, error) {
	id := strconv.Itoa(len(l.apps) + 1)
	l.apps[id] = app
	return &pb.ApplicationId{Id: id}, nil
}

func (l *loggyServer) GetDevice(ctx context.Context, deviceid *pb.DeviceId) (*pb.Device, error) {
	if device, ok := l.devices[deviceid.Id]; ok {
		return device, nil
	}
	return nil, nil
}

func (l *loggyServer) InsertDevice(ctx context.Context, device *pb.Device) (*pb.DeviceId, error) {
	id := strconv.Itoa(len(l.devices) + 1)
	l.devices[id] = device
	return &pb.DeviceId{Id: id}, nil
}

func (l *loggyServer) GetInstance(ctx context.Context, instanceid *pb.InstanceId) (*pb.Instance, error) {
	if instance, ok := l.instances[instanceid.Id]; ok {
		return instance, nil
	}
	return nil, nil
}

func (l *loggyServer) InsertInstance(ctx context.Context, instance *pb.Instance) (*pb.InstanceId, error) {
	id := strconv.Itoa(len(l.instances) + 1)
	l.instances[id] = instance
	return &pb.InstanceId{Id: id}, nil
}

func (l *loggyServer) Register(ctx context.Context, instanceid *pb.InstanceId) (*pb.ReceiverId, error) {
	id := string(len(l.receivers) + 1)
	l.receivers[id] = make(chan *pb.LoggyMessage, 100)
	l.listeners[instanceid.Id] = append(l.listeners[instanceid.Id], id)
	return &pb.ReceiverId{Id: id}, nil
}

func (l *loggyServer) Send(stream pb.LoggyService_SendServer) error {
	log.Println("Started stream")
	for {
		in, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		log.Printf("%s: %s\n", in.Instanceid, in.Msg)
		listeners := l.listeners[in.Instanceid]
		for _, receiverid := range listeners {
			if client, ok := l.receivers[receiverid]; ok {
				client <- in
			}
		}
	}
}

func (l *loggyServer) Receive(receiverid *pb.ReceiverId, stream pb.LoggyService_ReceiveServer) error {
	client := l.receivers[receiverid.Id]
	for in := range client {
		stream.Send(in)
	}
	return nil
}

func main() {
	grpcServer := grpc.NewServer()
	pb.RegisterLoggyServiceServer(grpcServer, &loggyServer{
		apps:      make(map[string]*pb.Application),
		devices:   make(map[string]*pb.Device),
		instances: make(map[string]*pb.Instance),
		receivers: make(map[string]chan *pb.LoggyMessage),
		listeners: make(map[string][]string),
	})

	l, err := net.Listen("tcp", ":50111")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	log.Println("Listening on tcp://localhost:50111")
	grpcServer.Serve(l)
}
