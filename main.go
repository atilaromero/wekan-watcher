package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/gorilla/mux"
)

type config struct {
	port       string
	graphqlURL string
	user       string
	pass       string
	userID     string
	token      string
	list       string
	board      string
}

func main() {
	port, ok := os.LookupEnv("PORT")
	if !ok {
		port = "80"
	}
	graphqlURL, ok := os.LookupEnv("GRAPHQL_URL")
	if !ok {
		log.Fatalf("GRAPHQL_URL not set. Example: GRAPHQL_URL=http://myserver:80")
		return
	}
	user, ok := os.LookupEnv("USER")
	if !ok {
		log.Fatalf("Graphql USER not set.")
		return
	}
	pass, ok := os.LookupEnv("PASS")
	if !ok {
		log.Fatalf("Graphql PASS not set.")
		return
	}
	list, ok := os.LookupEnv("LIST")
	if !ok {
		log.Fatalf("Wekan LIST not set.")
		return
	}
	board, ok := os.LookupEnv("BOARD")
	if !ok {
		log.Fatalf("Wekan BOARD not set.")
		return
	}

	cnf := config{
		port:       port,
		graphqlURL: graphqlURL,
		user:       user,
		pass:       pass,
		list:       list,
		board:      board,
	}

	if cnf.token == "" {
		err := cnf.getToken()
		if err != nil {
			log.Fatalf("error in getToken: %v", err)
			return
		}
	}

	r := mux.NewRouter()
	r.HandleFunc("/", getListTodo(cnf)).Methods("GET")

	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatalf("could not start server: %v\n", err)
		return
	}
}

type ToDo struct {
	Path    string `json:"evidencePath"`
	Output  string `json:"outputPath"`
	Profile string `json:"profile"`
}

func getListTodo(cnf config) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {

		docs, err := cnf.listTodo()
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, "error fetching cards: %v\n", err)
			return
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

func (cnf config) listTodo() ([]ToDo, error) {
	todos := []ToDo{}
	q := fmt.Sprintf(`
	query{
		board(auth:{userId:"%s", token:"%s"}, title:"%s"){
			customFields{
				id: _id
				name
			}
			list(title:"%s"){
				cards{
					title
					customFields{
						id: _id
						value
					}
				}
			}
		}
	}
	`, cnf.userID, cnf.token, cnf.board, cnf.list)
	d := struct {
		Errors []struct {
			Message string
		}
		Data struct {
			Board struct {
				CustomFields []struct {
					ID   string
					Name string
				}
				List struct {
					Cards []struct {
						Title        string
						CustomFields []struct {
							ID    string
							Value string
						}
					}
				}
			}
		}
	}{}
	err := cnf.query(q, &d)
	if err != nil {
		return nil, err
	}
	if len(d.Errors) > 0 {
		return nil, fmt.Errorf(d.Errors[0].Message)
	}
	fieldNames := make(map[string]string)
	for _, x := range d.Data.Board.CustomFields {
		fieldNames[x.Name] = x.ID
	}
	pathID, ok := fieldNames["path"]
	if !ok {
		return todos, fmt.Errorf("field path not found")
	}
	statusID, ok := fieldNames["status"]
	if !ok {
		return todos, fmt.Errorf("field status not found")
	}
	profileID, ok := fieldNames["profile"]
	if !ok {
		return todos, fmt.Errorf("field profile not found")
	}
	for _, card := range d.Data.Board.List.Cards {
		_path := ""
		status := ""
		profile := "pedo"
		for _, field := range card.CustomFields {
			if field.ID == pathID {
				_path = field.Value
			}
			if field.ID == statusID {
				status = field.Value
			}
			if field.ID == profileID {
				profile = field.Value
			}
		}
		todo := status == "todo" || status == ""
		if _path != "" && todo {
			todo := ToDo{
				Path:    _path,
				Profile: profile,
				Output:  path.Join(path.Dir(_path), "SARD"),
			}
			todos = append(todos, todo)
		}
	}
	return todos, nil
}

func (cnf *config) getToken() error {
	q := fmt.Sprintf(`
	query{
		authorize(user:"%s", password:"%s"){
			userId
			token
		}
	}
	`, cnf.user, cnf.pass)
	d := struct {
		Errors []struct {
			Message string
		}
		Data struct {
			Authorize struct {
				UserId string
				Token  string
			}
		}
	}{}
	err := cnf.query(q, &d)
	if err != nil {
		return err
	}
	if len(d.Errors) > 0 {
		return fmt.Errorf(d.Errors[0].Message)
	}
	cnf.userID = d.Data.Authorize.UserId
	cnf.token = d.Data.Authorize.Token
	return nil
}

func (cnf *config) query(q string, v interface{}) error {
	r, err := http.Post(cnf.graphqlURL, "application/graphql", strings.NewReader(q))
	if err != nil {
		return err
	}
	defer r.Body.Close()
	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return err
	}
	err = json.Unmarshal(b, v)
	if err != nil {
		return err
	}
	return nil
}
