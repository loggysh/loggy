package main

import (
	"context"
	"errors"
	"flag"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
	"io"
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/blevesearch/bleve"
	empty "github.com/golang/protobuf/ptypes/empty"
	uuid "github.com/satori/go.uuid"
	"github.com/tuxcanfly/loggy/loggy"
	pb "github.com/tuxcanfly/loggy/loggy"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"

	"github.com/tuxcanfly/loggy/service"
)

var IndexPath = "loggy.index"


type loggyServer struct {
	jwtManager *service.JWTManager
	lock          sync.RWMutex
	db            *gorm.DB
	indexer       bleve.Index
	notifications chan *pb.Session
	receivers     map[int32]chan *pb.Message
	listeners     map[int32][]int32 // sessionid -> []receivers

	loggy.UnimplementedLoggyServiceServer
}

func (l *loggyServer) InsertWaitListUser(ctx context.Context, app *pb.WaitListUser) (*empty.Empty, error) {
	entry := &service.WaitlistUser{
		Email: app.Email,
	}
	l.db.Where(entry).FirstOrCreate(&entry)
	return &empty.Empty{}, nil
}

func (l *loggyServer) GetOrInsertApplication(ctx context.Context, app *pb.Application) (*pb.Application, error) {
	split := strings.SplitN(app.Id, "/", 2)
	if len(split) != 2 {
		return &pb.Application{}, errors.New("invalid app id")
	}
	userID := split[0]
	appID := split[1]
	entry := &service.Application{
		ID:     appID,
		UserID: userID,
		Name:   app.Name,
		Icon:   app.Icon,
	}
	exists := &service.Application{}
	l.db.Where(entry).FirstOrCreate(&exists)
	return &pb.Application{
		Id:   exists.ID,
		Name: exists.Name,
		Icon: exists.Icon,
	}, nil
}

func (l *loggyServer) ListApplications(ctx context.Context, userid *pb.UserId) (*pb.ApplicationList, error) {
	var entries []*service.Application
	var apps []*pb.Application
	l.db.Where("user_id = ?", userid.Id).Find(&entries)
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
	entry := &service.Device{
		ID:      deviceid,
		Details: device.Details,
	}
	exists := &service.Device{}
	l.db.Where(entry).FirstOrCreate(&exists)
	return &pb.Device{
		Id:      exists.ID.String(),
		Details: exists.Details,
	}, nil
}

func (l *loggyServer) ListDevices(ctx context.Context, appid *pb.ApplicationId) (*pb.DeviceList, error) {
	var devices []*pb.Device
	var sessions []*service.Session
	l.db.Where("application_foreign_key = ?", appid.Id).Select("distinct(device_foreign_key)").Find(&sessions)
	for _, session := range sessions {
		device := &service.Device{}
		l.db.Where("id = ?", session.DeviceID).First(&device)
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
	var debugCount int32
	var infoCount int32
	var errorCount int32
	var warnCount int32
	var crashCount int32
	l.db.Model(&service.Message{}).Where("session_id = ?", sessionid.Id).Where("level = ?", 0).Count(&debugCount)
	l.db.Model(&service.Message{}).Where("session_id = ?", sessionid.Id).Where("level = ?", 1).Count(&infoCount)
	l.db.Model(&service.Message{}).Where("session_id = ?", sessionid.Id).Where("level = ?", 2).Count(&errorCount)
	l.db.Model(&service.Message{}).Where("session_id = ?", sessionid.Id).Where("level = ?", 3).Count(&warnCount)
	l.db.Model(&service.Message{}).Where("session_id = ?", sessionid.Id).Where("level = ?", 4).Count(&crashCount)
	return &pb.SessionStats{
		DebugCount: debugCount,
		InfoCount:  infoCount,
		ErrorCount: errorCount,
		WarnCount:  warnCount,
		CrashCount: crashCount,
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
/* Login is a unary RPC to login user which I'll be leaving out for now since we're using REST Server for authentication
func (l *loggyServer) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	var entry *service.User
	var user *pb.UserId
	l.db.Where("email = ?", req.Email).Model()

	token, err := l.jwtManager.Generate(entry)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "cannot generate access token")
	}




	if user == nil || !user.CheckPassword(req.GetPassword()) {
		return nil, status.Errorf(codes.NotFound, "incorrect username/password")
	}

	token, err := l.jwtManager.Generate(user)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "cannot generate access token")
	}

	res := &pb.LoginResponse{AccessToken: token}
	return res, nil
} */
const (
	secretKey     = "secret"
	tokenDuration = 15 * time.Minute
)
func main() {
	prefix := flag.String("prefix", "logs", "Prefix for logs. (logs)")
	server := flag.String("server", "localhost", "Server to connect to. (localhost)")
	flag.Parse()

	db, err := gorm.Open("sqlite3", "db/test.db")
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}
	defer db.Close()

	// Migrate the schema
	db.AutoMigrate(&service.Application{})
	db.AutoMigrate(&service.Device{})
	db.AutoMigrate(&service.Session{})
	db.AutoMigrate(&service.Message{})
	db.AutoMigrate(&service.User{})

	var indexer bleve.Index
	if _, err := os.Stat(IndexPath); os.IsNotExist(err) {
		indexer, err = bleve.New(IndexPath, bleve.NewIndexMapping())
	} else {
		indexer, err = bleve.Open(IndexPath)
	}
	if err != nil {
		log.Fatalf("failed to create index: %v", err)
	}
	jwtManager := service.NewJWTManager(secretKey, tokenDuration)
	interceptor := service.NewAuthInterceptor(jwtManager)
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

