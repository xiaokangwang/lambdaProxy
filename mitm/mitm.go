package main

/* unset https_proxy http_proxy all_proxy ALL_PROXY */

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"github.com/kr/mitm"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"

	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/lambda"

	"encoding/json"
	"fmt"
	"os"
)

func main() {
	ca, err := genCA()
	if err != nil {
		panic(err)
	}

	p := &mitm.Proxy{
		CA: &ca,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: false,
		},
		TLSServerConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
		},
		Wrap: func(upstream http.Handler) http.Handler {
			return http.HandlerFunc(rttPLAIN)
		},

		WrapTLS: func(upstream http.Handler) http.Handler {
			return http.HandlerFunc(rttTLS)
		},
	}

	l, err := net.Listen("tcp", "localhost:12299")
	if err != nil {
		panic(err)
	}
	defer l.Close()

	if err := http.Serve(l, p); err != nil {
		if !strings.Contains(err.Error(), "use of closed network") {
			panic(err)
		}
	}

}

func genCA() (cert tls.Certificate, err error) {
	if _, ok := os.Stat("ca.pem"); ok != nil {
		certPEM, keyPEM, err := mitm.GenCA("LambdaProxy")
		if err != nil {
			return tls.Certificate{}, err
		}

		err = ioutil.WriteFile("ca.pem", certPEM, 0600)
		if err != nil {
			return tls.Certificate{}, err
		}

		err = ioutil.WriteFile("key.pem", keyPEM, 0600)
		if err != nil {
			return tls.Certificate{}, err
		}
	}

	certPEM, err := ioutil.ReadFile("ca.pem")
	if err != nil {
		return tls.Certificate{}, err
	}
	keyPEM, err := ioutil.ReadFile("key.pem")
	if err != nil {
		return tls.Certificate{}, err
	}

	cert, err = tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return tls.Certificate{}, err
	}
	cert.Leaf, err = x509.ParseCertificate(cert.Certificate[0])
	return cert, err
}

type ProxyRequest struct {
	URL    string               `json:"url"`
	Method string               `json:"method"`
	Body   string               `json:"body"`
	Header []ProxyRequestCookie `json:"header"`
}

type ProxyRequestCookie struct {
	N string
	C string
}

type ProxyResp struct {
	StatusCode int                  `json:"statusc"`
	Status     string               `json:"status"`
	Body       string               `json:"body"`
	Header     []ProxyRequestCookie `json:"header"`
}

func rttTLS(w http.ResponseWriter, r *http.Request) {
	rtt(w,r,true)
}
func rttPLAIN(w http.ResponseWriter, r *http.Request) {
	rtt(w,r,false)
}

func rtt(w http.ResponseWriter, r *http.Request,istls bool) {
	host:=r.Host
	pto:="http"

	if istls {
		pto="https"
	}


	url := r.URL.RequestURI()

	urlx:=fmt.Sprintf("%v://%v%v",pto,host,url)

	url=urlx

	method := r.Method

	headers := make([]ProxyRequestCookie, 0)



	for headerk, header := range r.Header {
		for _, items := range header {
			headers = append(headers, ProxyRequestCookie{N: headerk, C: items})
		}

	}



	bf := &bytes.Buffer{}
	res := base64.NewEncoder(base64.URLEncoding, bf)

	_, err := io.Copy(res, r.Body)
	if err != nil {
		log.Println(err)
		return
	}

	res.Close()

	// Create Lambda service client
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	client := lambda.New(sess, &aws.Config{Region: aws.String("ap-northeast-1")})

	// Get the 10 most recent items
	request := ProxyRequest{url, method, bf.String(),
		headers}

	fmt.Println(url)

	payload, err := json.Marshal(request)
	if err != nil {
		fmt.Println("Error marshalling MyGetItemsFunction request")
		os.Exit(0)
	}

	result, err := client.Invoke(&lambda.InvokeInput{FunctionName: aws.String("progressProxyRequest"), Payload: payload})
	if err != nil {
		fmt.Println("Error calling progressProxyRequest")
	}

	var resp ProxyResp

	err = json.Unmarshal(result.Payload, &resp)

	//fmt.Println(resp)

	for _, header := range resp.Header {
		w.Header().Add(header.N, header.C)
	}

	w.WriteHeader(resp.StatusCode)

	bodyb := bytes.NewBufferString(resp.Body)

	resw := base64.NewDecoder(base64.URLEncoding, bodyb)

	_, err = io.Copy(w, resw)

	if err != nil {
		fmt.Println(err)
	}

}
