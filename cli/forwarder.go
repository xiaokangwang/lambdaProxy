package main

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/lambda"

	"encoding/json"
	"fmt"
	"os"
)


type ProxyRequest struct {
	URL string `json:"url"`
	Method string `json:"method"`
	Body string `json:"body"`
	Header []ProxyRequestCookie `json:"header"`
}

type ProxyRequestCookie struct {
	N string
	C string
}

type ProxyResp struct {
	StatusCode int `json:"statusc"`
	Status string `json:"status"`
	Body string `json:"body"`
	Header []ProxyRequestCookie `json:"header"`
}

func main() {
	// Create Lambda service client
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	client := lambda.New(sess, &aws.Config{Region: aws.String("us-west-1")})

	// Get the 10 most recent items
	request := ProxyRequest{"https://kkdev.org/cdn-cgi/trace","GET","",
		make([]ProxyRequestCookie,0)}

	payload, err := json.Marshal(request)
	if err != nil {
		fmt.Println("Error marshalling MyGetItemsFunction request")
		os.Exit(0)
	}

	result, err := client.Invoke(&lambda.InvokeInput{FunctionName: aws.String("progressProxyRequest"), Payload: payload})
	if err != nil {
		fmt.Println("Error calling progressProxyRequest")
		os.Exit(0)
	}



	var resp ProxyResp

	err = json.Unmarshal(result.Payload, &resp)

	fmt.Println( resp)

}