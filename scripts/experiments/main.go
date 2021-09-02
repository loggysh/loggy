package main

import (
	"fmt"
	"net/url"
	"path"
)

func authUrl() url.URL {
	url := url.URL{
		Scheme: "http",
		Host:   "localhost" + ":8080",
		Path:   "/api/public",
	}
	return url
}

func main() {

	someUrl := authUrl()
	someUrl.Path = path.Join(someUrl.Path, "/verify/key")
	q, _ := url.ParseQuery(someUrl.RawQuery)
	q.Add("api_key", "somekey")
	someUrl.RawQuery = q.Encode()

	fmt.Println(someUrl.String())
}
