package main

import (
	"context"
	"errors"
	"flag"
	"io"
	"log"
	"net"
	"time"

	"google.golang.org/grpc"

	empty "github.com/golang/protobuf/ptypes/empty"
	uuid "github.com/satori/go.uuid"
	pb "github.com/tuxcanfly/loggy/loggy"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
)

// Base contains common columns for all tables.
type Base struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key;"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time `sql:"index"`
}

// BeforeCreate will set a UUID rather than numeric ID.
func (base *Base) BeforeCreate(scope *gorm.Scope) error {
	uuid := uuid.NewV4()
	return scope.SetColumn("ID", uuid)
}

type Application struct {
	Base
	PackageName string `gorm:"unique"`
	Name        string
	Icon        string
}

type Device struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key;"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time `sql:"index"`
	Details   string
}

type Instance struct {
	Base
	AppID    uuid.UUID `gorm:"primary_key;type:uuid;column:application_foreign_key;not null;"`
	DeviceID uuid.UUID `gorm:"primary_key;type:uuid;column:device_foreign_key;not null;"`
}

type loggyServer struct {
	db            *gorm.DB
	notifications chan *pb.Session
	receivers     map[int32]chan *pb.LoggyMessage
	listeners     map[string][]int32 // instanceid -> []receivers
}

func (l *loggyServer) GetApplication(ctx context.Context, appid *pb.ApplicationId) (*pb.Application, error) {
	app := &Application{}
	if l.db.Where("id = ?", appid.Id).First(&app).RecordNotFound() {
		return nil, errors.New("app not found")
	}
	return &pb.Application{
		Id:          app.ID.String(),
		PackageName: app.PackageName,
		Name:        app.Name,
		Icon:        app.Icon,
	}, nil
}

func (l *loggyServer) InsertApplication(ctx context.Context, app *pb.Application) (*pb.ApplicationId, error) {
	entry := Application{
		PackageName: app.PackageName,
		Name:        app.Name,
		Icon:        app.Icon,
	}
	if l.db.Create(&entry).Error != nil {
		return nil, errors.New("unable to create app")
	}
	return &pb.ApplicationId{Id: entry.ID.String()}, nil
}

func (l *loggyServer) ListApplications(ctx context.Context, e *empty.Empty) (*pb.ApplicationList, error) {
	var entries []*Application
	var apps []*pb.Application
	l.db.Find(&entries)
	for _, app := range entries {
		apps = append(apps, &pb.Application{
			Id:          app.ID.String(),
			PackageName: app.PackageName,
			Name:        app.Name,
			Icon:        app.Icon,
		})
	}
	return &pb.ApplicationList{Apps: apps}, nil
}

func (l *loggyServer) GetDevice(ctx context.Context, devid *pb.DeviceId) (*pb.Device, error) {
	device := &Device{}
	if l.db.Where("id = ?", devid.Id).First(&device).RecordNotFound() {
		return nil, errors.New("device not found")
	}
	return &pb.Device{
		Details: device.Details,
	}, nil
}

func (l *loggyServer) InsertDevice(ctx context.Context, device *pb.Device) (*pb.DeviceId, error) {
	deviceid, err := uuid.FromString(device.Id)
	if err != nil {
		return nil, err
	}
	entry := Device{
		ID:      deviceid,
		Details: device.Details,
	}
	if l.db.Create(&entry).Error != nil {
		return nil, errors.New("unable to create device")
	}
	return &pb.DeviceId{Id: entry.ID.String()}, nil
}

func (l *loggyServer) ListDevices(ctx context.Context, appid *pb.ApplicationId) (*pb.DeviceList, error) {
	var entries []*Device
	var devices []*pb.Device
	var sessions []*Session
	l.db.Where("application_foreign_key = ?", appid.Id).Find(&sessions)
	for _, session := range session {
		l.db.Where("id = ?", session.DeviceID).Find(&entries)
	}
	for _, device := range entries {
		devices = append(devices, &pb.Device{
			Id:      device.ID.String(),
			Details: device.Details,
		})
	}
	return &pb.DeviceList{Devices: devices}, nil
}

func (l *loggyServer) GetOrInsertInstance(ctx context.Context, instance *pb.Instance) (*pb.InstanceId, error) {
	deviceid, err := uuid.FromString(instance.Deviceid)
	if err != nil {
		return nil, err
	}
	appid, err := uuid.FromString(instance.Appid)
	if err != nil {
		return nil, err
	}
	entry := &Instance{
		AppID:    appid,
		DeviceID: deviceid,
	}
	exists := &Instance{}
	l.db.Where(entry).FirstOrCreate(&exists)
	return &pb.InstanceId{
		Id: exists.ID.String(),
	}, nil
}

func (l *loggyServer) GetInstance(ctx context.Context, instanceid *pb.InstanceId) (*pb.Instance, error) {
	instance := &Instance{}
	if l.db.Where("id = ?", instanceid.Id).First(&instance).RecordNotFound() {
		return nil, errors.New("instance not found")
	}
	return &pb.Instance{
		Id:       instance.ID.String(),
		Deviceid: instance.DeviceID.String(),
		Appid:    instance.AppID.String(),
	}, nil
}

func (l *loggyServer) ListInstances(ctx context.Context, e *empty.Empty) (*pb.InstanceList, error) {
	var entries []*Instance
	var instances []*pb.Instance
	l.db.Find(&entries)
	for _, instance := range entries {
		instances = append(instances, &pb.Instance{
			Id:       instance.ID.String(),
			Deviceid: instance.DeviceID.String(),
			Appid:    instance.AppID.String(),
		})
	}
	return &pb.InstanceList{Instances: instances}, nil
}

func (l *loggyServer) Notify(e *empty.Empty, stream pb.LoggyService_NotifyServer) error {
	log.Println("Listening")
	for session := range l.notifications {
		log.Println(session)
		stream.Send(session)
	}
	return nil
}

func (l *loggyServer) RegisterSend(ctx context.Context, session *pb.Session) (*empty.Empty, error) {
	instance := &Instance{}
	if l.db.Where("id = ?", session.Instanceid).First(&instance).RecordNotFound() {
		return nil, errors.New("instance not found")
	}
	l.notifications <- &pb.Session{
		Id:       session.ID.String(),
		Deviceid: instance.DeviceID.String(),
		Appid:    instance.AppID.String(),
	}
	return &empty.Empty{}, nil
}

func (l *loggyServer) RegisterReceive(ctx context.Context, instanceid *pb.InstanceId) (*pb.ReceiverId, error) {
	id := int32(len(l.receivers) + 1)
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
	prefix := flag.String("prefix", "logs", "Prefix for logs. (logs)")
	server := flag.String("server", "localhost", "Server to connect to. (localhost)")
	flag.Parse()

	db, err := gorm.Open("sqlite3", "test.db")
	if err != nil {
		panic("failed to connect database")
	}
	defer db.Close()

	// Migrate the schema
	db.AutoMigrate(&Application{})
	db.AutoMigrate(&Device{})
	db.AutoMigrate(&Instance{})

	grpcServer := grpc.NewServer()
	pb.RegisterLoggyServiceServer(grpcServer, &loggyServer{
		db:            db,
		notifications: make(chan *pb.Session),
		receivers:     make(map[int32]chan *pb.LoggyMessage),
		listeners:     make(map[string][]int32),
	})

	l, err := net.Listen("tcp", ":50111")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	log.Println("Listening on tcp://localhost:50111")
	go logger(prefix, server)
	grpcServer.Serve(l)

}
