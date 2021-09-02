package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

var envDomain = os.Getenv("DOMAIN")

func domain() string {
	if envDomain == "" {
		return "localhost"
	}
	return envDomain
}

func authUrl() url.URL {
	url := url.URL{
		Scheme: "http",
		Host:   domain() + ":8080",
		Path:   "/api/public",
	}
	return url
}

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

var s = []string{"/loggy.LoggyService/Notify", "/loggy.LoggyService/RegisterReceive", "/loggy.LoggyService/Receive"}

//android methods - GetOrInsertApplication, GetOrInsertDevice, InsertSession, RegisterSend

func InterceptAndVerify(server string, allowed []string, interceptor *AuthInterceptor, ctx context.Context) (context.Context, error) {
	if !contains(allowed, server) {
		ctx, err := interceptor.authorize(ctx, server)
		if err != nil {
			return ctx, err
		}
		return ctx, nil
	}
	return ctx, nil
}

func (interceptor *AuthInterceptor) Unary() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		log.Println("--> unary interceptor: ", info.FullMethod)
		ctx, err := InterceptAndVerify(info.FullMethod, s, interceptor, ctx)
		if err != nil {
			return ctx, err
		}
		return handler(ctx, req)
	}
}

func (interceptor *AuthInterceptor) Stream() grpc.StreamServerInterceptor {
	return func(
		srv interface{},
		stream grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		log.Println("--> stream interceptor: ", info.FullMethod)
		newCtx, err := InterceptAndVerify(info.FullMethod, s, interceptor, stream.Context())
		if err != nil {
			fmt.Println(err)
		}

		md, _ := metadata.FromIncomingContext(newCtx)
		if len(md["user_id"]) != 0 {
			stream.SendHeader(metadata.Pairs("user_id", md["user_id"][0]))
		}
		return handler(srv, stream)
	}

}

func (interceptor *AuthInterceptor) authorize(ctx context.Context, method string) (context.Context, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ctx, status.Errorf(codes.Unauthenticated, "metadata is not provided")
	}

	client := md["client"]

	if len(client) == 0 {
		log.Println("client in metadata is not provided. proceeding with default")
	}

	var userID = ""

	if len(client) == 0 || client[0] == "web" {
		token := md["authorization"]
		metaUserID := md["user_id"]
		if len(token) == 0 {
			return ctx, status.Errorf(codes.Unauthenticated, "authorization token is not provided")
		}
		if len(metaUserID) == 0 {
			return ctx, status.Errorf(codes.Unauthenticated, "user id is not provided")
		}
		//Encode the data
		postBody, _ := json.Marshal(map[string]string{
			"token":   token[0],
			"user_id": metaUserID[0],
		})
		responseBody := bytes.NewBuffer(postBody)
		//Leverage Go's HTTP Post function to make request

		u := authUrl()
		u.Path = path.Join(u.Path, "/verify")

		resp, err := http.Post(u.String(), "application/json", responseBody)
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
			userID = metaUserID[0]
		}

	} else if client[0] == "android" {
		apiKey := md["api_key"]
		if len(apiKey) == 0 {
			return ctx, status.Errorf(codes.Unauthenticated, "api key is not provided")
		}
		//Leverage Go's HTTP Post function to make request

		u := authUrl()
		u.Path = path.Join(u.Path, "/verify/key")
		q, _ := url.ParseQuery(u.RawQuery)
		q.Add("api_key", apiKey[0])
		u.RawQuery = q.Encode()

		resp, err := http.Get(u.String())

		if resp.StatusCode == http.StatusOK {

			defer resp.Body.Close()
			//Read the response body
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				return ctx, status.Errorf(codes.Unauthenticated, "invalid user")
			}

			result := make(map[string]string)
			json.Unmarshal(body, &result)
			userID = result["user_id"]

		}

		//Handle Error
		if err != nil {
			log.Fatalf("An Error Occured %v", err)
			return ctx, status.Errorf(codes.Unauthenticated, "error occoured")
		}
	}

	if len(userID) > 0 {
		newMD := metadata.Pairs("user_id", userID)
		ctx = metadata.NewIncomingContext(ctx, metadata.Join(md, newMD))
		log.Println("Authorization Request granted")
	} else {
		log.Println("Authorization failed")
	}

	return ctx, nil
}
