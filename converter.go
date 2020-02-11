package main

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type Server struct {
	Address    string
	HTTPServer *http.Server
}

func (s *Server) Shutdown() error {
	return s.HTTPServer.Shutdown(context.Background())
}

func (s *Server) Start() error {
	addr := s.Address
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	addr = ln.Addr().(*net.TCPAddr).String()

	errors := make(chan error, 1)

	go func(listener net.Listener, errors chan error) {
		errors <- s.HTTPServer.Serve(listener)
		close(errors)
	}(ln, errors)
	time.Sleep(1 * time.Microsecond)

	if len(errors) > 0 {
		return <-errors
	}
	log.Println("Service is listening on http://" + addr + "/")
	return nil
}

func (s *Server) Serve() error {
	var gracefulStop = make(chan os.Signal, 1)
	signal.Notify(gracefulStop, syscall.SIGTERM)
	signal.Notify(gracefulStop, syscall.SIGINT)

	err := s.Start()
	if err != nil {
		close(gracefulStop)
		return err
	}
	log.Println("Please press Ctrl+C to stop service")
	<-gracefulStop
	log.Println("Gracefully stopping service")

	return s.Shutdown()
}

type InputStruct struct {
	A string `json:"a"`
	B int    `json:"b"`
	C bool   `json:"c"`
}

type OutputStruct struct {
	D string `json:"d"`
	E bool   `json:"e"`
	F int    `json:"f"`
}

func convert(input *InputStruct) (*OutputStruct, error) {
	output := OutputStruct{
		D: input.A,
		E: input.C,
		F: input.B,
	}
	return &output, nil
}

func handler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		data, err := ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(
				w,
				err.Error(),
				http.StatusBadRequest,
			)
			return
		}
		var input InputStruct
		err = json.Unmarshal(data, &input)
		if err != nil {
			http.Error(
				w,
				err.Error(),
				http.StatusBadRequest,
			)
			return
		}
		output, err := convert(&input)
		if err != nil {
			http.Error(
				w,
				err.Error(),
				http.StatusInternalServerError,
			)
			return
		}
		data, err = json.Marshal(output)
		if err != nil {
			http.Error(
				w,
				err.Error(),
				http.StatusInternalServerError,
			)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write(data); err != nil {
			log.Print(err)
		}
		return
	}
	http.Error(
		w,
		"Method Not Allowed",
		http.StatusMethodNotAllowed,
	)
}

func main() {
	server := Server{
		HTTPServer: &http.Server{
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 10 * time.Second,
		},
		Address: "localhost:8088",
	}
	http.HandleFunc("/", handler)
	err := server.Serve()
	if err != nil {
		log.Println(err)
	}
}
