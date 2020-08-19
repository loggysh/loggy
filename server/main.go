package main

import (
	"context"
	"io"
	"log"
	"net"

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

type applicationServer struct {
	apps map[string]*pb.Application
}

func (a *applicationServer) Get(ctx context.Context, appid *pb.ApplicationId) (*pb.Application, error) {
	return a.apps[appid.Id], nil
}

func (a *applicationServer) Insert(ctx context.Context, app *pb.Application) (*pb.ApplicationId, error) {
	a.apps[app.Id] = app
	return &pb.ApplicationId{Id: app.Id}, nil
}

type deviceServer struct {
	devices map[int32]*pb.Device
}

func (d *deviceServer) Get(ctx context.Context, deviceid *pb.DeviceId) (*pb.Device, error) {
	return d.devices[deviceid.Id], nil
}

func (d *deviceServer) Insert(ctx context.Context, device *pb.Device) (*pb.DeviceId, error) {
	id := int32(len(d.devices) + 1)
	d.devices[id] = device
	return &pb.DeviceId{Id: id}, nil
}

type instanceServer struct {
	instances map[int32]*pb.Instance
}

func (i *instanceServer) Get(ctx context.Context, instanceid *pb.InstanceId) (*pb.Instance, error) {
	return i.instances[instanceid.Id], nil
}

func (i *instanceServer) Insert(ctx context.Context, instance *pb.Instance) (*pb.InstanceId, error) {
	id := int32(len(i.instances) + 1)
	i.instances[id] = instance
	return &pb.InstanceId{Id: id}, nil
}

type loggyServer struct {
	clients   map[int32]chan *pb.LoggyMessage
	listeners map[int32][]int32 // instanceid -> []clients
}

func (l *loggyServer) RegisterClient(ctx context.Context, instanceid *pb.InstanceId) (*pb.ClientId, error) {
	id := int32(len(l.clients) + 1)
	l.clients[id] = make(chan *pb.LoggyMessage, 100)
	l.listeners[instanceid.Id] = append(l.listeners[instanceid.Id], id)
	return &pb.ClientId{Id: id}, nil
}

func (l *loggyServer) LoggyServer(stream pb.LoggyService_LoggyServerServer) error {
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
		listeners := l.listeners[in.Id]
		for _, clientid := range listeners {
			if client, ok := l.clients[clientid]; ok {
				client <- in
			}
		}
	}
}

func (l *loggyServer) LoggyClient(clientid *pb.ClientId, stream pb.LoggyService_LoggyClientServer) error {
	client := l.clients[clientid.Id]
	for in := range client {
		stream.Send(in)
	}
	return nil
}

func main() {
	grpcServer := grpc.NewServer()
	pb.RegisterLoggyServiceServer(grpcServer, &loggyServer{clients: make(map[int32]chan *pb.LoggyMessage), listeners: make(map[int32][]int32)})
	pb.RegisterApplicationServiceServer(grpcServer, &applicationServer{apps: make(map[string]*pb.Application)})
	pb.RegisterDeviceServiceServer(grpcServer, &deviceServer{devices: make(map[int32]*pb.Device)})
	pb.RegisterInstanceServiceServer(grpcServer, &instanceServer{instances: make(map[int32]*pb.Instance)})

	l, err := net.Listen("tcp", ":50111")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	log.Println("Listening on tcp://localhost:50111")
	grpcServer.Serve(l)
}
