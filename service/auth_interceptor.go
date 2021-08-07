package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type AuthInterceptor struct {
	name string
}

func NewAuthInterceptor(name string) *AuthInterceptor {
	return &AuthInterceptor{name}
}
func contains(slice []string, item string) bool {
	set := make(map[string]struct{}, len(slice))
	for _, s := range slice {
		set[s] = struct{}{}
	}

	_, ok := set[item]
	return ok
}

func (interceptor *AuthInterceptor) Unary() grpc.UnaryServerInterceptor {
	s := []string{"/loggy.LoggyService/Notify", "/loggy.LoggyService/RegisterReceive", "/loggy.LoggyService/Receive"}
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		log.Println("--> unary interceptor: ", info.FullMethod)
		if !contains(s, info.FullMethod) {
			err := interceptor.authorize(ctx, info.FullMethod)
			if err != nil {
				return nil, err
			}
		}

		return handler(ctx, req)
	}
}

func (interceptor *AuthInterceptor) Stream() grpc.StreamServerInterceptor {
	s := []string{"/loggy.LoggyService/Notify", "/loggy.LoggyService/RegisterReceive", "/loggy.LoggyService/Receive", "/loggy.LoggyService/Recv"}
	return func(
		srv interface{},
		stream grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		log.Println("--> stream interceptor: ", info.FullMethod)
		if !contains(s, info.FullMethod) {
			err := interceptor.authorize(stream.Context(), info.FullMethod)
			if err != nil {
				return err
			}
		}
		return handler(srv, stream)
	}

}

func (interceptor *AuthInterceptor) authorize(ctx context.Context, method string) error {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return status.Errorf(codes.Unauthenticated, "metadata is not provided")
	}

	token := md["authorization"]

	if len(token) == 0 {
		return status.Errorf(codes.Unauthenticated, "authorization token is not provided")
	}
	userID := md["user_id"]
	if len(userID) == 0 {
		return status.Errorf(codes.Unauthenticated, "user id is not provided")
	}
	//Encode the data
	postBody, _ := json.Marshal(map[string]string{
		"Token":  token[0],
		"UserID": userID[0],
	})
	responseBody := bytes.NewBuffer(postBody)
	//Leverage Go's HTTP Post function to make request
	resp, err := http.Post(BuildUrl(), "application/json", responseBody)
	//Handle Error
	if err != nil {
		log.Fatalf("An Error Occured %v", err)
	}

	defer resp.Body.Close()
	//Read the response body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln(err)
	}

	sb := string(body)
	fmt.Println(sb)
	if sb != `{"message":"token valid"}` {
		return status.Error(codes.PermissionDenied, "no permission to access this RPC")
	}
	return nil
}

func BuildUrl() (s string) {
	if os.Getenv("DOMAIN") == "localhost" {
		authUrl := "http://localhost:8080/api/public/verify"
		return authUrl
	} else if len(os.Getenv("DOMAIN")) == 0 {
		authUrl := "http://localhost:8080/api/public/verify"
		return authUrl
	} else {
		authUrl := "http://" + os.Getenv("DOMAIN") + ":8080/api/public/verify"
		return authUrl
	}
}