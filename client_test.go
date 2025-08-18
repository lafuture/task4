package awesomeProject5

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"testing"
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

//func main() {
//	http.HandleFunc("/search", SearchServer)
//
//	log.Println("Server started at http://localhost:8080")
//	err := http.ListenAndServe(":8080", nil)
//	if err != nil {
//		log.Fatal(err)
//	}
//}

type SearchTestCase struct {
	Query    string
	OrderBy  string
	OrderFld string
	Limit    string
	Offset   string
	Result   []string // ожидаемые имена пользователей
	IsError  bool
}

var readXMLFunc = readXML

func TestSearchServer(t *testing.T) {
	// создаём временный XML
	tmpFile, err := os.CreateTemp("", "dataset_*.xml")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	xmlData := `
<Users>
    <row>
        <id>1</id>
        <first_name>Alice</first_name>
        <last_name>Smith</last_name>
        <age>25</age>
        <about>hello</about>
    </row>
    <row>
        <id>2</id>
        <first_name>Bob</first_name>
        <last_name>Brown</last_name>
        <age>30</age>
        <about>developer</about>
    </row>
    <row>
        <id>3</id>
        <first_name>Charlie</first_name>
        <last_name>Johnson</last_name>
        <age>20</age>
        <about>developer</about>
    </row>
</Users>
`
	tmpFile.WriteString(xmlData)
	tmpFile.Close()

	// подменяем readXMLFunc
	oldReadXMLFunc := readXMLFunc
	readXMLFunc = func(_ string) []People {
		return readXML(tmpFile.Name())
	}
	defer func() { readXMLFunc = oldReadXMLFunc }()

	// тестовые кейсы
	cases := []SearchTestCase{
		{
			Query:   "developer",
			Limit:   "10",
			Offset:  "0",
			Result:  []string{"Bob", "Charlie"},
			IsError: false,
		},
		{
			Query:   "Alice",
			Limit:   "10",
			Offset:  "0",
			Result:  []string{"Alice"},
			IsError: false,
		},
		{
			Query:   "Alice",
			Limit:   "1",
			Offset:  "0",
			Result:  []string{"Alice"},
			IsError: false,
		},
		{
			Query:   "Alice",
			Limit:   "1",
			Offset:  "10", // offset больше количества
			Result:  []string{},
			IsError: false,
		},
	}

	for i, tc := range cases {
		q := url.Values{}
		q.Set("query", tc.Query)
		q.Set("limit", tc.Limit)
		q.Set("offset", tc.Offset)
		q.Set("order_by", tc.OrderBy)
		q.Set("order_field", tc.OrderFld)

		req := httptest.NewRequest(http.MethodGet, "/search?"+q.Encode(), nil)
		w := httptest.NewRecorder()

		SearchServer(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK && !tc.IsError {
			t.Errorf("[%d] unexpected status: %d", i, resp.StatusCode)
			continue
		}
		if resp.StatusCode == http.StatusOK && tc.IsError {
			t.Errorf("[%d] expected error, got status 200", i)
			continue
		}

		body, _ := ioutil.ReadAll(resp.Body)
		var got []People
		if err := json.Unmarshal(body, &got); err != nil && !tc.IsError {
			t.Errorf("[%d] failed to parse json: %v", i, err)
			continue
		}

		if len(got) != len(tc.Result) {
			t.Errorf("[%d] expected %d results, got %d", i, len(tc.Result), len(got))
			continue
		}

		for j := range got {
			if got[j].Firstname != tc.Result[j] {
				t.Errorf("[%d] expected %s, got %s", i, tc.Result[j], got[j].Firstname)
			}
		}
	}
}
