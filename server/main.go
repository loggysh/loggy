package main

import (
	"context"
	"errors"
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
	PackageName string
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
	AppID    uuid.UUID `gorm:"type:uuid;column:application_foreign_key;not null;"`
	DeviceID uuid.UUID `gorm:"type:uuid;column:device_foreign_key;not null;"`
}

type loggyServer struct {
	db        *gorm.DB
	receivers map[int32]chan *pb.LoggyMessage
	listeners map[string][]int32 // instanceid -> []receivers
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
	dev := &Device{}
	if l.db.Where("id = ?", devid.Id).First(&dev).RecordNotFound() {
		return nil, errors.New("device not found")
	}
	return &pb.Device{
		Details: dev.Details,
	}, nil
}

func (l *loggyServer) InsertDevice(ctx context.Context, dev *pb.Device) (*pb.DeviceId, error) {
	deviceid, err := uuid.FromString(dev.Id)
	if err != nil {
		return nil, err
	}
	entry := Device{
		ID:      deviceid,
		Details: dev.Details,
	}
	if l.db.Create(&entry).Error != nil {
		return nil, errors.New("unable to create device")
	}
	return &pb.DeviceId{Id: entry.ID.String()}, nil
}

func (l *loggyServer) ListDevices(ctx context.Context, appid *pb.ApplicationId) (*pb.DeviceList, error) {
	var entries []*Device
	var devices []*pb.Device
	var instances []*Instance
	l.db.Where("application_foreign_key = ?", appid.Id).Find(&instances)
	for _, instance := range instances {
		l.db.Where("id = ?", instance.DeviceID).Find(&entries)
	}
	for _, device := range entries {
		devices = append(devices, &pb.Device{
			Id:      device.ID.String(),
			Details: device.Details,
		})
	}
	return &pb.DeviceList{Devices: devices}, nil
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

func (l *loggyServer) InsertInstance(ctx context.Context, inst *pb.Instance) (*pb.InstanceId, error) {
	deviceid, err := uuid.FromString(inst.Deviceid)
	if err != nil {
		return nil, err
	}
	appid, err := uuid.FromString(inst.Appid)
	if err != nil {
		return nil, err
	}
	entry := Instance{
		DeviceID: deviceid,
		AppID:    appid,
	}
	if l.db.Create(&entry).Error != nil {
		return nil, errors.New("unable to create instance")
	}
	return &pb.InstanceId{Id: entry.ID.String()}, nil
}

func (l *loggyServer) Register(ctx context.Context, instanceid *pb.InstanceId) (*pb.ReceiverId, error) {
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
		log.Printf("Instance: %s, Session: %s: %s\n", in.Instanceid, in.Sessionid, in.Msg)
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
		db:        db,
		receivers: make(map[int32]chan *pb.LoggyMessage),
		listeners: make(map[string][]int32),
	})

	l, err := net.Listen("tcp", ":50111")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	log.Println("Listening on tcp://localhost:50111")
	grpcServer.Serve(l)
}
