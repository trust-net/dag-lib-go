package main

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/trust-net/dag-lib-go/log"
	"net/http"
	"strconv"
)

var logger = log.NewLogger("Client API")

func getFoo(w http.ResponseWriter, r *http.Request) {
	logger.Debug("Recieved GET /ping from: %s", r.RemoteAddr)
	setHeaders(w)
	json.NewEncoder(w).Encode("pong!")
}

type ResourceByKey struct {
	Key   string `json:"key,omitempty"`
	Owner string `json:"owner,omitempty"`
	Value uint64 `json:"value"`
}

func setHeaders(w http.ResponseWriter) {
	w.Header().Set("content-type", "application/json")
}

func getResourceByKey(w http.ResponseWriter, r *http.Request) {
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

func StartServer(listenPort int) error {
	// if not a valid port, do not start
	if listenPort < 1024 {
		return fmt.Errorf("Invalid port: %d", listenPort)
	}

	router := mux.NewRouter()
	router.HandleFunc("/foo", getFoo).Methods("GET")
	router.HandleFunc("/resources/{key}", getResourceByKey).Methods("GET")
	go func() {
		logger.Error("End of server: %s", http.ListenAndServe(":"+strconv.Itoa(listenPort), router))
	}()
	return nil
}
