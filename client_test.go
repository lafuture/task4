package awesomeProject5

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"
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

func TestSortUsers(t *testing.T) {
	//users := readXML("dataset.xml")
	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer ts.Close()

	cl := SearchClient{URL: ts.URL}

	resp1, err := cl.FindUsers(SearchRequest{Limit: 100, Offset: 0, Query: "", OrderField: "Name", OrderBy: -1})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp1.Users) == 0 {
		t.Fatal(err)
	}

	_, err2 := cl.FindUsers(SearchRequest{Limit: -1, Offset: 0})
	if err2.Error() != "limit must be > 0" {
		t.Fatal(err)
	}

	_, err3 := cl.FindUsers(SearchRequest{Limit: 25, Offset: -1})
	if err3.Error() != "offset must be > 0" {
		t.Fatal(err)
	}

	resp2, err := cl.FindUsers(SearchRequest{Limit: 100, Offset: 0, Query: ";(ВОВ(ВО№(", OrderField: "Name", OrderBy: -1})
	if resp2 == nil {
		t.Fatal(err)
	}
}

func TestStatus401(t *testing.T) {
	//users := readXML("dataset.xml")
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer ts.Close()

	cl := SearchClient{URL: ts.URL}

	_, err := cl.FindUsers(SearchRequest{Limit: 100, Offset: 0, Query: "", OrderField: "Name", OrderBy: -1})
	if err.Error() != "Bad AccessToken" {
		t.Fatal(err)
	}
}

func TestStatus500(t *testing.T) {
	//users := readXML("dataset.xml")
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	cl := SearchClient{URL: ts.URL}

	_, err := cl.FindUsers(SearchRequest{Limit: 100, Offset: 0, Query: "", OrderField: "Name", OrderBy: -1})
	if err.Error() != "SearchServer fatal error" {
		t.Fatal(err)
	}
}

func TestStatus400_1(t *testing.T) {
	//users := readXML("dataset.xml")
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, `{"error":"ErrorBadOrderField"}`)
	}))
	defer ts.Close()

	cl := SearchClient{URL: ts.URL}

	_, err := cl.FindUsers(SearchRequest{Limit: 100, Offset: 0, Query: "", OrderField: "Name", OrderBy: -1})
	if !strings.Contains(err.Error(), "OrderFeld") {
		t.Fatal(err)
	}
}

func TestStatus400_2(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, `{"error": *#(SBAD}`)
	}))
	defer ts.Close()

	cl := SearchClient{URL: ts.URL}

	_, err := cl.FindUsers(SearchRequest{Limit: 100, Offset: 0, Query: "", OrderField: "Name", OrderBy: -1})
	if !strings.Contains(err.Error(), "json") {
		t.Fatal(err)
	}
}

func TestStatus400_3(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, `{"error":"OtherError"}`)
	}))
	defer ts.Close()

	cl := SearchClient{URL: ts.URL}

	_, err := cl.FindUsers(SearchRequest{Limit: 100, Offset: 0, Query: "", OrderField: "Name", OrderBy: -1})
	if !strings.Contains(err.Error(), "unknown") {
		t.Fatal(err)
	}
}

func TestJson(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"name": *(SNdJS*#}`)
	}))
	defer ts.Close()

	cl := SearchClient{URL: ts.URL}

	_, err := cl.FindUsers(SearchRequest{Limit: 100, Offset: 0, Query: "", OrderField: "Name", OrderBy: -1})
	if !strings.Contains(err.Error(), "cant unpack result json") {
		t.Fatal(err)
	}
}

func TestGet(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	ts.Close()

	cl := SearchClient{URL: ts.URL}

	_, err := cl.FindUsers(SearchRequest{Limit: 100, Offset: 0, Query: "", OrderField: "Name", OrderBy: -1})
	if !strings.Contains(err.Error(), "unknown error") {
		t.Fatal(err)
	}
}

func TestTimeout(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
	}))
	defer ts.Close()

	cl := SearchClient{URL: ts.URL}

	_, err := cl.FindUsers(SearchRequest{Limit: 100, Offset: 0, Query: "", OrderField: "Name", OrderBy: -1})
	if !strings.Contains(err.Error(), "timeout") {
		t.Fatal(err)
	}
}
