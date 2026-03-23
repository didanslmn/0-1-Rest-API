package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"strconv"
	"time"
)

type Response struct {
	Success bool   `json:"sucess"`
	Message string `json:"message"`
	Data    any    `json:"result"`
}

type CalculateResult struct {
	Expression string  `json:"expression"`
	Result     float64 `json:"result"`
}

type ConvertResult struct {
	From   string  `json:"from"`
	To     string  `json:"to"`
	Amount float64 `json:"amount"`
	Result float64 `json:"result"`
	Rate   float64 `json:"rate"`
	Date   string  `json:"date"`
}

// semua kurs relatif terhadap usd

var rates = map[string]float64{
	"usd": 1,
	"eur": 0.85,
	"idr": 16000,
	"gbp": 0.75,
	"jpy": 150,
	"rub": 70.00,
}

// helper : json response

func writeJson(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJson(w, status, Response{Success: false, Message: message})
}

// handler: Get /kalkulate

func calculateHandler(w http.ResponseWriter, r *http.Request) {
	// hanya terima method get
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Mehod hanya mendukung method GET")
		return
	}

	op := r.URL.Query().Get("op")
	if op == "" {
		writeError(w, http.StatusBadRequest, "parameter op wajib diisi (add/substract/multiply/ devide...)")
		return
	}

	// sqrt hanya butuh 1 angka
	if op == "sqrt" {
		aStr := r.URL.Query().Get("a")
		a, err := strconv.ParseFloat(aStr, 64)
		if err != nil || a < 0 {
			writeError(w, http.StatusBadRequest, "parameter 'a' harus angka positif")
			return
		}
		result := math.Sqrt(a)
		writeJson(w, http.StatusOK, Response{
			Success: true,
			Data: CalculateResult{
				Expression: fmt.Sprintf("sqrt(%g)", a),
				Result:     result,
			},
		})
		return
	}

	// operasi laiinya (butuh 2 angka a dan b)

	aStr := r.URL.Query().Get("a")
	bStr := r.URL.Query().Get("b")

	a, errA := strconv.ParseFloat(aStr, 64)
	b, errB := strconv.ParseFloat(bStr, 64)

	if errA != nil {
		writeError(w, http.StatusBadRequest, "parameter a tidak valid")
		return
	}

	if errB != nil {
		writeError(w, http.StatusBadRequest, "parameter b tidak valid")
		return
	}

	var result float64
	var expr string

	switch op {
	case "add":
		result = a + b
		expr = fmt.Sprintf("%g + %g", a, b)
	case "substract":
		result = a - b
		expr = fmt.Sprintf("%g - %g", a, b)
	case "multiply":
		result = a * b
		expr = fmt.Sprintf("%g x %g")
	case "devide":
		if b == 0 {
			writeError(w, http.StatusBadRequest, "tidak bisa membagi dengan 0")
			return
		}
		result = a / b
		expr = fmt.Sprintf("%g / %g", a, b)
	case "power":
		result = math.Pow(a, b)
		expr = fmt.Sprintf("%g ^ %g", a, b)
	default:
		writeError(w, http.StatusBadRequest, "operasi tidak dikenal. gunakan(add/substract/multiply/devide/power/sqrt)")
		return
	}
	writeJson(w, http.StatusOK, Response{
		Success: true,
		Data: CalculateResult{
			Expression: expr,
			Result:     result,
		},
	})

}

// Handler: Get /convert
// contoh: /convert?from=USD&to=IDR&amount=100

func convertHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "hanya mendukung method GET")
		return
	}

	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")
	amountStr := r.URL.Query().Get("amount")

	// validasi paramater

	if from == "" || to == "" || amountStr == "" {
		writeError(w, http.StatusBadRequest, "parameter tidak boleh kosong")
		return
	}

	amount, err := strconv.ParseFloat(amountStr, 64)
	if err != nil || amount < 0 {
		writeError(w, http.StatusBadRequest, "parameter 'amount' harus lebih dari 0")
		return
	}

	// cek apakah mata uang tersedia
	fromRate, fromOK := rates[from]
	toRate, toOK := rates[to]

	if !fromOK {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("mata uang '%s' tidak tersedia", from))
		return
	}
	if !toOK {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("mata uang '%s' tidak tersedia", to))
	}

	// konversi: from-> usd->to
	inUSD := amount / fromRate
	result := inUSD * toRate
	rate := toRate / fromRate

	writeJson(w, http.StatusOK, Response{
		Success: true,
		Data: ConvertResult{
			From:   from,
			To:     to,
			Amount: amount,
			Result: math.Round(result*100) / 100,
			Rate:   math.Round(rate*10000) / 10000,
			Date:   time.Now().Format("2006-01-02"),
		},
	})
}

// handler: GET /rates

// tampilkan semua kurs yang tersedia

func ratesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, " hanya mendukung method GET")
		return
	}
	writeJson(w, http.StatusOK, Response{
		Success: true,
		Data:    rates,
	})
}

// Handler: GET /
// Halaman info semua endpoint yang tersedia

func homeHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		writeError(w, http.StatusNotFound, "Endpoint tidak ditemukan")
		return
	}

	info := map[string]any{
		"app": "Curency calculator API",
		"endpoint": []map[string]string{
			{
				"method":  "GET",
				"path":    "/calculate",
				"params":  "op,a,b",
				"example": "/calculate?op=multiply&a=5&b=3",
			},
			{
				"method":  "GET",
				"path":    "/convert",
				"params":  "from,to,amount",
				"example": "/convert?from=USD&to=IDR&amount=100",
			},
			{
				"method":  "GET",
				"path":    "/rates",
				"params":  "",
				"example": "/rates",
			},
		},
		"support currencies": []string{"usd", "eur", "idr", "gbp", "jpy", "rub"},
	}
	writeJson(w, http.StatusOK, Response{
		Success: true,
		Data:    info,
	})

}

// main

func main() {
	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/calculate", calculateHandler)
	http.HandleFunc("/convert", convertHandler)
	http.HandleFunc("/rates", ratesHandler)

	port := ":8080"
	log.Printf("server berjalan di http://localhost:%s", port)
	log.Fatal(http.ListenAndServe(port, nil))

}
