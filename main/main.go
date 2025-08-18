package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
)

type People struct {
	Id            int    `xml:"id"`
	Guid          string `xml:"guid"`
	IsActive      bool   `xml:"isActive"`
	Balance       string `xml:"balance"`
	Picture       string `xml:"picture"`
	Age           int    `xml:"age"`
	EyeColor      string `xml:"eyeColor"`
	Firstname     string `xml:"first_name"`
	Lastname      string `xml:"last_name"`
	Gender        string `xml:"gender"`
	Company       string `xml:"company"`
	Email         string `xml:"email"`
	Phone         string `xml:"phone"`
	Adress        string `xml:"adress"`
	About         string `xml:"about"`
	Registered    string `xml:"registered"`
	FavoriteFruit string `xml:"favoriteFruit"`
}

type Users struct {
	Users []People `xml:"row"`
}

func readXML(path string) []People {
	rawfile, _ := os.Open(path)
	defer rawfile.Close()
	file, _ := ioutil.ReadAll(rawfile)
	var data Users
	err := xml.Unmarshal(file, &data)
	if err != nil {
		panic(err)
	}
	return data.Users
}

func sortUsers(res []People, order_field, order_by string) ([]People, error) {
	switch order_field {
	case "Name":
		sort.Slice(res, func(i, j int) bool {
			first := res[i].Firstname + res[i].Lastname
			second := res[j].Firstname + res[j].Lastname
			if order_by == "-1" {
				return first < second
			} else {
				return first > second
			}
		})
	case "Id":
		sort.Slice(res, func(i, j int) bool {
			if order_by == "-1" {
				return res[i].Id < res[j].Id
			} else {
				return res[i].Id > res[j].Id
			}
		})
	case "Age":
		sort.Slice(res, func(i, j int) bool {
			if order_by == "-1" {
				return res[i].Age < res[j].Age
			} else {
				return res[i].Age > res[j].Age
			}
		})
	case "":
		sort.Slice(res, func(i, j int) bool {
			first := res[i].Firstname + res[i].Lastname
			second := res[j].Firstname + res[j].Lastname
			if order_by == "-1" {
				return first < second
			} else {
				return first > second
			}
		})
	default:
		return nil, fmt.Errorf("unknown field")
	}
	return res, nil
}

func SearchServer(w http.ResponseWriter, req *http.Request) {
	res := []People{}
	users := readXML("dataset.xml")

	query := req.URL.Query().Get("query")
	for _, user := range users {
		if strings.Contains(user.Firstname, query) || strings.Contains(user.Lastname, query) || strings.Contains(user.About, query) {
			res = append(res, user)
		}
	}

	order_field := req.URL.Query().Get("order_field")
	order_by := req.URL.Query().Get("order_by")
	if order_by != "0" {
		var err error
		res, err = sortUsers(res, order_field, order_by)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}

	limit, err := strconv.Atoi(req.URL.Query().Get("limit"))
	if err != nil {
		http.Error(w, "invalid limit", http.StatusBadRequest)
		return
	}
	offset, err := strconv.Atoi(req.URL.Query().Get("offset"))
	if err != nil {
		http.Error(w, "invalid offset", http.StatusBadRequest)
		return
	}
	if offset > len(res) {
		offset = len(res)
	}
	end := offset + limit
	if end > len(res) {
		end = len(res)
	}

	w.Header().Set("Content-Type", "application/json")
	jsonData, err := json.Marshal(res[offset:end])
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	w.Write(jsonData)
}

func main() {
	http.HandleFunc("/search", SearchServer)

	log.Println("Server started at http://localhost:8080")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal(err)
	}
}
