// Copyright 2018-2019 The trust-net Authors
// REST API for spendr application
package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/trust-net/dag-lib-go/api"
	"github.com/trust-net/dag-lib-go/log"
	"net/http"
	"strconv"
)

var logger = log.NewLogger("Client API")

// A world state resource for spendr application
type Resource struct {
	Key   string `json:"key,omitempty"`
	Owner string `json:"owner,omitempty"`
	Value uint64 `json:"value"`
}

// response to successful submission of a transaction
type OpcodeResponse struct {
	Payload     string `json:"payload"`
	Description string `json:"description"`
}

// A spendr application OpCode request to transfer value from owned resource
type OpCodeXferValueRequest struct {
	// The unique identifier for an owned resource within spendr application
	Source string `json:"source"`
	// The unique identifier for destination resource within spendr application
	Destination string `json:"destination"`
	// 64 bit unsigned integer value to transfer from owned resource to destination resource
	Value uint64 `json:"value"`
}

func getFoo(w http.ResponseWriter, r *http.Request) {
	logger.Debug("Recieved GET /ping from: %s", r.RemoteAddr)
	setHeaders(w)
	json.NewEncoder(w).Encode("pong!")
}

func setHeaders(w http.ResponseWriter) {
	w.Header().Set("content-type", "application/json")
}

func getResourceByKey(w http.ResponseWriter, r *http.Request) {
	// fetch request params
	params := mux.Vars(r)
	logger.Debug("Recieved GET /resources/%s from: %s", params["key"], r.RemoteAddr)
	// set headers
	setHeaders(w)
	// fetch resource from spendr app
	owner, value, err := doGetResource(params["key"])
	if err != nil {
		logger.Debug("did not get %s: %s", params["key"], err)
		w.WriteHeader(404)
		json.NewEncoder(w).Encode(err.Error())
	} else {
		json.NewEncoder(w).Encode(Resource{
			Key:   params["key"],
			Owner: fmt.Sprintf("%x", owner),
			Value: value,
		})
	}
}

func submitTransaction(w http.ResponseWriter, r *http.Request) {
	logger.Debug("Recieved POST /transactions from: %s", r.RemoteAddr)
	// set headers
	setHeaders(w)
	// parse request body
	req, err := api.ParseSubmitRequest(r)
	if err != nil {
		logger.Debug("Failed to decode request body: %s", err)
		w.WriteHeader(400)
		json.NewEncoder(w).Encode(err.Error())
		return
	}
	// submit transaction to app
	if tx, err := doSubmitTransaction(req.DltRequest()); err != nil {
		logger.Debug("Failed to submit transaction: %s", err)
		w.WriteHeader(http.StatusNotAcceptable)
		json.NewEncoder(w).Encode(err.Error())
	} else {
		// respond back with transaction submission result
		json.NewEncoder(w).Encode(api.NewSubmitResponse(tx))
	}
}

func requestResourceCreationPayload(w http.ResponseWriter, r *http.Request) {
	logger.Debug("Recieved POST /opcode/create from: %s", r.RemoteAddr)
	// set headers
	setHeaders(w)
	// parse request body
	req := &Resource{}
	if err := json.NewDecoder(r.Body).Decode(req); err != nil {
		logger.Debug("Malformed request: %s", err)
		w.WriteHeader(400)
		json.NewEncoder(w).Encode(err.Error())
		return
	}
	// respond with payload for the request
	json.NewEncoder(w).Encode(OpcodeResponse{
		Payload:     base64.StdEncoding.EncodeToString(makeResourceCreationPayload(req.Key, int64(req.Value))),
		Description: "Create " + req.Key,
	})
}

func requestXferValuePayload(w http.ResponseWriter, r *http.Request) {
	logger.Debug("Recieved POST /opcode/xfer from: %s", r.RemoteAddr)
	// set headers
	setHeaders(w)
	// parse request body
	req := &OpCodeXferValueRequest{}
	if err := json.NewDecoder(r.Body).Decode(req); err != nil {
		logger.Debug("Malformed request: %s", err)
		w.WriteHeader(400)
		json.NewEncoder(w).Encode(err.Error())
		return
	}
	// respond with payload for the request
	json.NewEncoder(w).Encode(OpcodeResponse{
		Payload:     base64.StdEncoding.EncodeToString(makeXferValuePayload(req.Source, req.Destination, int64(req.Value))),
		Description: "Transfer from " + req.Source + " to " + req.Destination,
	})
}

func StartServer(listenPort int) error {
	// if not a valid port, do not start
	if listenPort < 1024 {
		return fmt.Errorf("Invalid port: %d", listenPort)
	}

	router := mux.NewRouter()
	router.HandleFunc("/foo", getFoo).Methods("GET")
	router.HandleFunc("/resources/{key}", getResourceByKey).Methods("GET")
	router.HandleFunc("/transactions", submitTransaction).Methods("POST")
	router.HandleFunc("/opcode/create", requestResourceCreationPayload).Methods("POST")
	router.HandleFunc("/opcode/xfer", requestXferValuePayload).Methods("POST")
	go func() {
		logger.Error("End of server: %s", http.ListenAndServe(":"+strconv.Itoa(listenPort), router))
	}()
	return nil
}
