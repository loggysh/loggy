package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/blevesearch/bleve"
	empty "github.com/golang/protobuf/ptypes/empty"
	uuid "github.com/satori/go.uuid"
	"github.com/tuxcanfly/loggy/loggy"
	pb "github.com/tuxcanfly/loggy/loggy"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
)

var IndexPath = "loggy.index"

// Base contains common columns for all tables.
type Base struct {
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time `sql:"index"`
}

type Application struct {
	Base
	ID   string
	Name string
	Icon string
}

type Device struct {
	Base
	ID      uuid.UUID `gorm:"type:uuid;primary_key;"`
	Details string
}

type Session struct {
	Base
	ID       int32
	AppID    string    `gorm:"type:string;column:application_foreign_key;not null;"`
	DeviceID uuid.UUID `gorm:"type:uuid;column:device_foreign_key;not null;"`
}

type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
	CRASH
)

type Message struct {
	ID int
	Base
	SessionID int32
	Session   Session
	Msg       string
	Timestamp time.Time
	Level     LogLevel
}

func (m *Message) String() string {
	var level string
	switch m.Level {
	case DEBUG:
		level = "DEBUG"
	case INFO:
		level = "INFO"
	case WARN:
		level = "WARN"
	case ERROR:
		level = "ERROR"
	case CRASH:
		level = "CRASH"
	default:
		level = "undefined"
	}
	return fmt.Sprintf("%v :: %d :: <%s> :: %s", m.Timestamp, m.SessionID, level, m.Msg)
}

type loggyServer struct {
	db            *gorm.DB
	indexer       bleve.Index
	notifications chan *pb.Session
	receivers     map[int32]chan *pb.Message
	listeners     map[int32][]int32 // sessionid -> []receivers
}

func (l *loggyServer) GetOrInsertApplication(ctx context.Context, app *pb.Application) (*pb.Application, error) {
	entry := &Application{
		ID:   app.Id,
		Name: app.Name,
		Icon: app.Icon,
	}
	exists := &Application{}
	l.db.Where(entry).FirstOrCreate(&exists)
	return &pb.Application{
		Id:   exists.ID,
		Name: exists.Name,
		Icon: exists.Icon,
	}, nil
}

func (l *loggyServer) ListApplications(ctx context.Context, e *empty.Empty) (*pb.ApplicationList, error) {
	var entries []*Application
	var apps []*pb.Application
	l.db.Find(&entries)
	for _, app := range entries {
		apps = append(apps, &pb.Application{
			Id:   app.ID,
			Name: app.Name,
			Icon: app.Icon,
		})
	}
	return &pb.ApplicationList{Apps: apps}, nil
}

func (l *loggyServer) GetOrInsertDevice(ctx context.Context, device *pb.Device) (*pb.Device, error) {
	deviceid, err := uuid.FromString(device.Id)
	if err != nil {
		return nil, err
	}
	entry := &Device{
		ID:      deviceid,
		Details: device.Details,
	}
	exists := &Device{}
	l.db.Where(entry).FirstOrCreate(&exists)
	return &pb.Device{
		Id:      exists.ID.String(),
		Details: exists.Details,
	}, nil
}

func (l *loggyServer) ListDevices(ctx context.Context, appid *pb.ApplicationId) (*pb.DeviceList, error) {
	var entries []*Device
	var devices []*pb.Device
	var sessions []*Session
	l.db.Where("application_foreign_key = ?", appid.Id).Find(&sessions)
	for _, session := range sessions {
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

func (l *loggyServer) InsertSession(ctx context.Context, session *pb.Session) (*pb.SessionId, error) {
	deviceid, err := uuid.FromString(session.Deviceid)
	if err != nil {
		return nil, err
	}
	exists := &Session{
		AppID:    session.Appid,
		DeviceID: deviceid,
	}
	l.db.Create(&exists)
	return &pb.SessionId{
		Id: exists.ID,
	}, nil
}

func (l *loggyServer) ListSessions(ctx context.Context, query *pb.SessionQuery) (*pb.SessionList, error) {
	var entries []*Session
	var sessions []*pb.Session
	l.db.Where("application_foreign_key = ?", query.Appid).Where("device_foreign_key = ?", query.Deviceid).Find(&entries)
	for _, session := range entries {
		sessions = append(sessions, &pb.Session{
			Id:       session.ID,
			Deviceid: session.DeviceID.String(),
			Appid:    session.AppID,
		})
	}
	return &pb.SessionList{Sessions: sessions}, nil
}

func (l *loggyServer) ListSessionMessages(ctx context.Context, sessionid *pb.SessionId) (*pb.MessageList, error) {
	var entries []*Message
	var messages []*pb.Message
	l.db.Where("session_id = ?", sessionid.Id).Find(&entries)
	for _, message := range entries {
		messages = append(messages, &pb.Message{
			Sessionid: message.SessionID,
			Msg:       message.Msg,
			Timestamp: timestamppb.New(message.Timestamp),
			Level:     loggy.Message_Level(message.Level),
		})
	}
	return &pb.MessageList{Messages: messages}, nil
}

func (l *loggyServer) Notify(e *empty.Empty, stream pb.LoggyService_NotifyServer) error {
	log.Println("Listening")
	for session := range l.notifications {
		log.Println(session)
		stream.Send(session)
	}
	return nil
}

func (l *loggyServer) RegisterSend(ctx context.Context, sessionid *pb.SessionId) (*empty.Empty, error) {
	session := &Session{}
	if l.db.Where("id = ?", sessionid.Id).First(&session).RecordNotFound() {
		return nil, errors.New("session not found")
	}
	l.notifications <- &pb.Session{
		Id:       session.ID,
		Deviceid: session.DeviceID.String(),
		Appid:    session.AppID,
	}
	return &empty.Empty{}, nil
}

func (l *loggyServer) RegisterReceive(ctx context.Context, sessionid *pb.SessionId) (*pb.ReceiverId, error) {
	id := int32(len(l.receivers) + 1)
	l.receivers[id] = make(chan *pb.Message, 100)
	l.listeners[sessionid.Id] = append(l.listeners[sessionid.Id], id)
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
		listeners := l.listeners[in.Sessionid]
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

func (l *loggyServer) Search(ctx context.Context, query *pb.Query) (*pb.MessageList, error) {
	result, err := l.indexer.Search(bleve.NewSearchRequest(bleve.NewFuzzyQuery(query.Query)))
	if err != nil {
		log.Println(err)
	}
	var messages []*pb.Message
	for _, hit := range result.Hits {
		msg := &Message{}
		if l.db.Where("id = ?", hit.ID).First(&msg).RecordNotFound() {
			return nil, errors.New("msg not found")
		}
		messages = append(messages, &pb.Message{
			Sessionid: msg.SessionID,
			Msg:       msg.Msg,
			Timestamp: timestamppb.New(msg.Timestamp),
			Level:     pb.Message_Level(msg.Level),
		})
	}
	return &pb.MessageList{Messages: messages}, nil
}

func main() {
	prefix := flag.String("prefix", "logs", "Prefix for logs. (logs)")
	server := flag.String("server", "localhost", "Server to connect to. (localhost)")
	flag.Parse()

	db, err := gorm.Open("sqlite3", "test.db")
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}
	defer db.Close()

	// Migrate the schema
	db.AutoMigrate(&Application{})
	db.AutoMigrate(&Device{})
	db.AutoMigrate(&Session{})
	db.AutoMigrate(&Message{})

	var indexer bleve.Index
	if _, err := os.Stat(IndexPath); os.IsNotExist(err) {
		indexer, err = bleve.New(IndexPath, bleve.NewIndexMapping())
	} else {
		indexer, err = bleve.Open(IndexPath)
	}
	if err != nil {
		log.Fatalf("failed to create index: %v", err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterLoggyServiceServer(grpcServer, &loggyServer{
		db:            db,
		indexer:       indexer,
		notifications: make(chan *pb.Session),
		receivers:     make(map[int32]chan *pb.Message),
		listeners:     make(map[int32][]int32),
	})

	l, err := net.Listen("tcp", ":50111")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	go logger(prefix, server, indexer, db)

	log.Println("Listening on tcp://localhost:50111")
	grpcServer.Serve(l)
}
