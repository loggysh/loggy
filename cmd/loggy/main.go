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
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/blevesearch/bleve"
	empty "github.com/golang/protobuf/ptypes/empty"
	uuid "github.com/satori/go.uuid"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/tuxcanfly/loggy/loggy"
	pb "github.com/tuxcanfly/loggy/loggy"
	"github.com/tuxcanfly/loggy/service"
)

var IndexPath = "loggy.index"

type loggyServer struct {
	lock          sync.RWMutex
	db            *gorm.DB
	indexer       bleve.Index
	notifications chan *pb.Session
	receivers     map[int32]chan *pb.Message
	listeners     map[int32][]int32 // sessionid -> []receivers

	loggy.UnimplementedLoggyServiceServer
}

func getUserIdFromMetaData(ctx context.Context) (string, error) {
	md, _ := metadata.FromIncomingContext(ctx)
	if len(md["user_id"]) == 0 {
		return "", fmt.Errorf("no user id in metadata")
	}
	userID := md["user_id"][0]
	fmt.Println(userID)
	return userID, nil
}

func (l *loggyServer) InsertWaitListUser(ctx context.Context, app *pb.WaitListUser) (*empty.Empty, error) {
	entry := &service.WaitlistUser{
		Email: app.Email,
	}
	l.db.Where(entry).FirstOrCreate(&entry)
	return &empty.Empty{}, nil
}

func (l *loggyServer) GetOrInsertApplication(ctx context.Context, app *pb.Application) (*pb.Application, error) {
	userID, err := getUserIdFromMetaData(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to add application. no user id")
	}

	//append userID to App ID
	appID := userID + "/" + app.Packagename

	entry := &service.Application{
		ID:          appID,
		UserID:      userID,
		PackageName: app.Packagename,
		Name:        app.Name,
		Icon:        app.Icon,
	}
	exists := &service.Application{}
	l.db.Where(entry).FirstOrCreate(&exists)
	return &pb.Application{
		Id:          exists.ID,
		Packagename: exists.PackageName,
		Name:        exists.Name,
		Icon:        exists.Icon,
	}, nil
}

func (l *loggyServer) ListApplications(ctx context.Context, userid *pb.UserId) (*pb.ApplicationList, error) {
	var entries []*service.Application
	var apps []*pb.Application
	l.db.Where("user_id = ?", userid.Id).Find(&entries)
	for _, app := range entries {
		apps = append(apps, &pb.Application{
			Id:          app.ID,
			Packagename: app.PackageName,
			Name:        app.Name,
			Icon:        app.Icon,
		})
	}
	return &pb.ApplicationList{Apps: apps}, nil
}

func (l *loggyServer) GetOrInsertDevice(ctx context.Context, device *pb.Device) (*pb.Device, error) {
	deviceid, err := uuid.FromString(device.Id)
	if err != nil {
		return nil, err
	}
	if len(device.Appid) == 0 {
		return nil, fmt.Errorf("failed to add device. no app id")
	}
	entry := &service.Device{
		ID:      deviceid,
		AppID:   device.Appid,
		Details: device.Details,
	}
	exists := &service.Device{}
	l.db.Where(entry).FirstOrCreate(&exists)
	return &pb.Device{
		Id:      exists.ID.String(),
		Appid:   exists.AppID,
		Details: exists.Details,
	}, nil
}

func (l *loggyServer) ListDevices(ctx context.Context, appid *pb.ApplicationId) (*pb.DeviceList, error) {
	var devices []*pb.Device
	l.db.Where("application_id = ?", appid.Id).Find(&devices)
	return &pb.DeviceList{Devices: devices}, nil
}

func (l *loggyServer) InsertSession(ctx context.Context, session *pb.Session) (*pb.SessionId, error) {
	deviceid, err := uuid.FromString(session.Deviceid)
	if err != nil {
		return nil, err
	}
	exists := &service.Session{
		AppID:    session.Appid,
		DeviceID: deviceid,
	}
	l.db.Create(&exists)
	return &pb.SessionId{
		Id: exists.ID,
	}, nil
}

func (l *loggyServer) ListSessions(ctx context.Context, query *pb.SessionQuery) (*pb.SessionList, error) {
	var entries []*service.Session
	var sessions []*pb.Session
	l.db.Where("application_id = ?", query.Appid).Where("device_id = ?", query.Deviceid).Find(&entries)
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
	var entries []*service.Message
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

func (l *loggyServer) GetSessionStats(ctx context.Context, sessionid *pb.SessionId) (*pb.SessionStats, error) {
	var debugCount int64
	var infoCount int64
	var errorCount int64
	var warnCount int64
	var crashCount int64
	l.db.Model(&service.Message{}).Where("session_id = ?", sessionid.Id).Where("level = ?", 0).Count(&debugCount)
	l.db.Model(&service.Message{}).Where("session_id = ?", sessionid.Id).Where("level = ?", 1).Count(&infoCount)
	l.db.Model(&service.Message{}).Where("session_id = ?", sessionid.Id).Where("level = ?", 2).Count(&errorCount)
	l.db.Model(&service.Message{}).Where("session_id = ?", sessionid.Id).Where("level = ?", 3).Count(&warnCount)
	l.db.Model(&service.Message{}).Where("session_id = ?", sessionid.Id).Where("level = ?", 4).Count(&crashCount)
	return &pb.SessionStats{
		DebugCount: int32(debugCount),
		InfoCount:  int32(infoCount),
		ErrorCount: int32(errorCount),
		WarnCount:  int32(warnCount),
		CrashCount: int32(crashCount),
	}, nil
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
	session := &service.Session{}

	err := l.db.Where("id = ?", sessionid.Id).First(&session).Error
	if err != nil {
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
	l.lock.Lock()
	defer l.lock.Unlock()

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
		l.lock.RLock()
		listeners := l.listeners[in.Sessionid]
		for _, receiverid := range listeners {
			if client, ok := l.receivers[receiverid]; ok {
				client <- in
			}
		}
		l.lock.RUnlock()
	}
}

func (l *loggyServer) Receive(receiverid *pb.ReceiverId, stream pb.LoggyService_ReceiveServer) error {
	l.lock.RLock()
	client := l.receivers[receiverid.Id]
	l.lock.RUnlock()

	for in := range client {
		stream.Send(in)
	}
	return nil
}

func (l *loggyServer) Search(ctx context.Context, query *pb.Query) (*pb.MessageList, error) {
	result, err := l.indexer.Search(bleve.NewSearchRequest(bleve.NewQueryStringQuery(query.Query)))
	if err != nil {
		log.Println(err)
	}
	var messages []*pb.Message
	for _, hit := range result.Hits {
		msg := &service.Message{}
		err := l.db.Where("id = ?", hit.ID).First(&msg).Error
		if err != nil {
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

	db, err := gorm.Open(sqlite.Open("db/test.db"), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}

	// Migrate the schema
	db.AutoMigrate(&service.Application{})
	db.AutoMigrate(&service.Device{})
	db.AutoMigrate(&service.Session{})
	db.AutoMigrate(&service.Message{})
	db.AutoMigrate(&service.WaitlistUser{})

	var indexer bleve.Index
	if _, err := os.Stat(IndexPath); os.IsNotExist(err) {
		indexer, _ = bleve.New(IndexPath, bleve.NewIndexMapping())
	} else {
		indexer, _ = bleve.Open(IndexPath)
	}
	if err != nil {
		log.Fatalf("failed to create index: %v", err)
	}

	interceptor := service.NewAuthInterceptor("Auth")
	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(interceptor.Unary()),
		grpc.StreamInterceptor(interceptor.Stream()),
	)
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
