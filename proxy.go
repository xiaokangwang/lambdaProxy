package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"github.com/aws/aws-lambda-go/lambda"
	"io"
	"net/http"
	"strings"
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

func HandleRequest(ctx context.Context, req ProxyRequest) (ProxyResp, error) {
	sp:=ProxyResp{}
	b:=base64.NewDecoder(base64.URLEncoding,strings.NewReader(req.Body))
	requ,err:=http.NewRequest(req.Method,req.URL,b)
	if err != nil {
		return sp,err
	}
	for _,header := range req.Header  {
		requ.Header.Add(header.N,header.C)
	}

	var DefaultTransport http.RoundTripper = &http.Transport{}

	resp,err:=DefaultTransport.RoundTrip(requ)
	if err != nil {
		return sp,err
	}

	bf:=&bytes.Buffer{}
	res:=base64.NewEncoder(base64.URLEncoding,bf)

	_,err=io.Copy(res,resp.Body)

	res.Close()
	if err != nil {
		return sp,err
	}

	sp.Body = bf.String()

	sp.StatusCode=resp.StatusCode
	sp.Status=resp.Status
	sp.Header=make([]ProxyRequestCookie,0)
	for headerk,header := range resp.Header {
		for _,items := range header {
			sp.Header = append(sp.Header, ProxyRequestCookie{N: headerk, C: items})
		}

	}

	return sp,nil
}

func main() {
	lambda.Start(HandleRequest)
}
