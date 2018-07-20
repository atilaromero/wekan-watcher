package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path"

	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	"github.com/gorilla/mux"
)

func main() {
	port, ok := os.LookupEnv("PORT")
	if !ok {
		port = "80"
	}
	mongoURL, ok := os.LookupEnv("MONGO_URL")
	if !ok {
		log.Fatalf("MONGO_URL not set. Example: MONGO_URL=mongodb://myserver:27017")
		return
	}
	mongoDatabase, ok := os.LookupEnv("MONGO_DATABASE")
	if !ok {
		mongoDatabase = "sard"
	}
	mongoCollection, ok := os.LookupEnv("MONGO_COLLECTION")
	if !ok {
		mongoCollection = "material"
	}

	client, err := mgo.Dial(mongoURL)
	if err != nil {
		log.Fatalf("could not connect to mongo database: %v\n", err)
		return
	}

	collection := client.DB(mongoDatabase).C(mongoCollection)

	r := mux.NewRouter()
	r.HandleFunc("/", getListTodo(collection)).Methods("GET")

	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatalf("could not start server: %v\n", err)
		return
	}
}

func getListTodo(collection *mgo.Collection) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		docs := make([]struct {
			Path   string `bson:"path" json:"evidencePath"`
			Output string `json:"outputPath"`
		}, 0)

		err := collection.Find(
			bson.M{"state": "todo"},
		).Limit(100).Select(bson.M{"path": 1}).All(&docs)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, "error fetching database: %v\n", err)
			return
		}

		for i := 0; i < len(docs); i++ {
			docs[i].Output = path.Join(path.Dir(docs[i].Path), "SARD")
		}

		docsJSON, err := json.Marshal(docs)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, "error building json: %v\n", err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(docsJSON)
	}
}
