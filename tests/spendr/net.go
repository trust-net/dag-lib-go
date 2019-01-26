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
type ResourceByKey struct {
	Key   string `json:"key,omitempty"`
	Owner string `json:"owner,omitempty"`
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
		json.NewEncoder(w).Encode(ResourceByKey{
			Key:   params["key"],
			Owner: fmt.Sprintf("%x", owner),
			Value: value,
		})
	}
}

func requestAnchor(w http.ResponseWriter, r *http.Request) {
	logger.Debug("Recieved POST /anchors from: %s", r.RemoteAddr)
	// set headers
	setHeaders(w)
	// parse request body
	req, err := api.ParseAnchorRequest(r)
	if err != nil {
		logger.Debug("Failed to decode request body: %s", err)
		w.WriteHeader(400)
		json.NewEncoder(w).Encode(err.Error())
		return
	}
	// fetch an anchor from app
	if anchor := doRequestAnchor(req.SubmitterBytes(), req.NextSeq, req.LastTxBytes()); anchor == nil {
		logger.Debug("failed to anchor!!!")
		w.WriteHeader(400)
		json.NewEncoder(w).Encode("no anchor")
	} else {
		// respond back with anchor
		json.NewEncoder(w).Encode(api.NewAnchorResponse(anchor))
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
	if err := doSubmitTransaction(req.DltTransaction()); err != nil {
		logger.Debug("Failed to submit transaction: %s", err)
		w.WriteHeader(http.StatusNotAcceptable)
		json.NewEncoder(w).Encode(err.Error())
	} else {
		// respond back with transaction submission result
		json.NewEncoder(w).Encode(api.NewSubmitResponse(req.DltTransaction()))
	}
}

func requestResourceCreation(w http.ResponseWriter, r *http.Request) {
	logger.Debug("Recieved POST /opcode/create from: %s", r.RemoteAddr)
	// set headers
	setHeaders(w)
	// parse request body
	req := &ResourceByKey{}
	if err := json.NewDecoder(r.Body).Decode(req); err != nil {
		logger.Debug("Malformed request: %s", err)
		w.WriteHeader(400)
		json.NewEncoder(w).Encode(err.Error())
		return
	}
	// respond with payload for the request
	json.NewEncoder(w).Encode(base64.StdEncoding.EncodeToString(makeResourceCreationPayload(req.Key, int64(req.Value))))
}

func StartServer(listenPort int) error {
	// if not a valid port, do not start
	if listenPort < 1024 {
		return fmt.Errorf("Invalid port: %d", listenPort)
	}

	router := mux.NewRouter()
	router.HandleFunc("/foo", getFoo).Methods("GET")
	router.HandleFunc("/resources/{key}", getResourceByKey).Methods("GET")
	router.HandleFunc("/anchors", requestAnchor).Methods("POST")
	router.HandleFunc("/transactions", submitTransaction).Methods("POST")
	router.HandleFunc("/opcode/create", requestResourceCreation).Methods("POST")
	go func() {
		logger.Error("End of server: %s", http.ListenAndServe(":"+strconv.Itoa(listenPort), router))
	}()
	return nil
}
