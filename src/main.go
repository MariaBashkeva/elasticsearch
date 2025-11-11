package main

import (
	"encoding/json"
	"github.com/dgrijalva/jwt-go"
	"go_day03/pkg/db"
	"html/template"
	"log"
	"math"
	"net/http"
	"strconv"
	"time"
	// "fmt"
)

type Store interface {
	GetPlaces(limit int, offset int) ([]db.Place, int, error)
	GetClosestPlace(lat float64, lon float64) ([]db.Place, error)
}

type Data struct {
	Name     string        `json:"name"`
	Total    int           `json:"total"`
	Places   []db.Place `json:"places"`
	PrevPage int           `json:"prev_page"`
	NextPage int           `json:"next_page"`
	Last     int           `json:"last_page"`
	Page     int           `json:"-"`
}

var (
	client Store
	jwtKey = []byte("secretKey228")
)

func init() {
	es, err := db.NewElastic()
	if err != nil {
		log.Fatal(err)
	}
	es.LoadData()
	client = es
}

// curl -H "Authorization: Bearer <token>" "http://127.0.0.1:8888/api/recommend?lat=55.674&lon=37.666"


func main() {
	http.HandleFunc("/", handlePages)
	http.HandleFunc("/api/places", handleApiPages)
	http.Handle("/api/recommend", MiddleWare(http.HandlerFunc(handleRecommend)))
	// http.HandleFunc("/api/recommend",handleRecommend)
	http.HandleFunc("/api/get_token", getToken)
	err := http.ListenAndServe(":8888", nil)
	if err != nil {
		log.Fatal(err)
	}
}

func getToken(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	claims := jwt.StandardClaims{
		ExpiresAt: time.Now().Add(5 * time.Minute).Unix(),
		IssuedAt:  time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtKey)
	if err != nil {
		http.Error(w, "Could not generate token", http.StatusInternalServerError)
	}
	json.NewEncoder(w).Encode(map[string]string{"token": tokenString})
}

func MiddleWare(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		tokenString := r.Header.Get("Authorization")
		if tokenString == "" {
			http.Error(w, "No token found", http.StatusUnauthorized)
			return
		}
		tokenString = tokenString[len("Bearer "):]

		claims := &jwt.StandardClaims{}
		tkn, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			return jwtKey, nil
		})
		if err != nil || !tkn.Valid {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func handleRecommend(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// latParam := r.URL.Query().Get("lat")
    // lonParam := r.URL.Query().Get("lon")

    // lat, err := strconv.ParseFloat(latParam, 64)
    // if err != nil {
    //     http.Error(w, fmt.Sprintf("Invalid 'lat' value: '%s'", latParam), http.StatusBadRequest)
    //     return
    // }

    // lon, err := strconv.ParseFloat(lonParam, 64)
    // if err != nil {
    //     http.Error(w, fmt.Sprintf("Invalid 'lon' value: '%s'", lonParam), http.StatusBadRequest)
    //     return
    // }

    // places, err := client.GetClosestPlace(lat, lon)
    // if err != nil {
    //     http.Error(w, "Failed to fetch recommendations", http.StatusInternalServerError)
    //     return
    // }

    // response := struct {
    //     Name   string  `json:"name"`
    //     Places []db.Place `json:"places"`
    // }{
    //     Name:   "Recommendation",
    //     Places: places,
    // }

    // json.NewEncoder(w).Encode(response)

	lat, err := strconv.ParseFloat(r.URL.Query().Get("lat"), 64)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
	lon, err := strconv.ParseFloat(r.URL.Query().Get("lon"), 64)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
	places, err := client.GetClosestPlace(lat, lon)
	var res = struct {
		Name   string        `json:"name"`
		Places []db.Place `json:"places"`
	}{
		Name:   "Recommendation",
		Places: places,
	}
	jsRes, err := json.MarshalIndent(res, "", "    ")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	w.WriteHeader(http.StatusOK)
	_, err = w.Write(jsRes)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func handleApiPages(w http.ResponseWriter, r *http.Request) {
	var res Data
	var err error
	w.Header().Set("Content-Type", "application/json")
	pageStr := r.URL.Query().Get("page")
	res.Page, err = strconv.Atoi(pageStr)
	if err != nil || res.Page < 0 {
		http.Error(w, `{
    "error": "Invalid 'page' value: 'foo'"
}
`, http.StatusBadRequest)
		return
	}
	if res.Page == 0 {
		res.PrevPage = 0
	} else {
		res.PrevPage = res.Page - 1
	}
	res.NextPage = res.Page + 1
	res.Last = res.Page - 1
	limit := 10
	offset := res.Page * limit
	res.Places, res.Total, err = client.GetPlaces(limit, offset)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	res.Last = int(math.Ceil(float64(res.Total)/float64(limit))) - 1
	res.Name = "Places"
	if res.Page > res.Last {
		http.Error(w, `{
    "error": "Invalid 'page' value: 'foo'"
}
`, http.StatusBadRequest)
		return
	}
	jsres, err := json.Marshal(res)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	w.WriteHeader(http.StatusOK)
	_, err = w.Write(jsres)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func handlePages(w http.ResponseWriter, r *http.Request) {
	pageStr := r.URL.Query().Get("page")
	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 0 {
		http.Error(w, "Invalid 'page' value: 'foo'", http.StatusBadRequest)
		return
	}

	limit := 10
	offset := page * limit

	places, total, err := client.GetPlaces(limit, offset)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	lastPage := int(math.Ceil(float64(total)/float64(limit))) - 1

	if page > lastPage {
		http.Error(w, "Invalid 'page' value: 'foo'", http.StatusBadRequest)
		return
	}

	tmpl, err := template.New("index.html").Funcs(
		template.FuncMap{
			"add": add,
			"sub": sub,
		},
	).ParseFiles("templates/index.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = tmpl.Execute(w, map[string]interface{}{
		"Places": places,
		"Page":   page,
		"Total":  total,
		"Last":   lastPage,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func add(a, b int) int { return a + b }
func sub(a, b int) int { return a - b }
