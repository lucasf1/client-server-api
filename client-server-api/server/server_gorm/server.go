package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

const URL_SERVICE = "https://economia.awesomeapi.com.br/json/last/USD-BRL"

type CoinQuery struct {
	Usdbrl struct {
		Code       string `json:"code"`
		Codein     string `json:"codein"`
		Name       string `json:"name"`
		High       string `json:"high"`
		Low        string `json:"low"`
		VarBid     string `json:"varBid"`
		PctChange  string `json:"pctChange"`
		Bid        string `json:"bid"`
		Ask        string `json:"ask"`
		Timestamp  string `json:"timestamp"`
		CreateDate string `json:"create_date"`
	} `json:"USDBRL"`
}

type Coin struct {
	gorm.Model
	ID        uint `gorm:"primaryKey"`
	Code      string
	CodeIn    string
	Descricao string
	Valor     float64
}

func main() {
	http.HandleFunc("/cotacao", HandleCotacao)
	http.ListenAndServe(":8080", nil)
}

func HandleCotacao(w http.ResponseWriter, r *http.Request) {

	// time.Sleep(time.Second)

	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, 200*time.Millisecond)
	defer cancel()

	req, err := http.Get(URL_SERVICE)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Erro ao fazer requisição ao serviço de cotação: %v\n", err)
		return
	}
	defer req.Body.Close()

	res, err := io.ReadAll(req.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Erro ao ler resposta: %v\n", err)
		return
	}

	var cotacao CoinQuery
	err = json.Unmarshal(res, &cotacao)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Erro ao fazer parser da resposta: %v\n", err)
		return
	}

	err = salvarCotacaoBD(&cotacao)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Erro ao salvar cotação: %v\n", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(cotacao.Usdbrl.Bid)
}

func salvarCotacaoBD(cotacao *CoinQuery) error {
	db, err := gorm.Open(sqlite.Open("cotacao.db"), &gorm.Config{})
	if err != nil {
		return err
	}

	// Migrate the schema
	db.AutoMigrate(&Coin{})

	valor, err := strconv.ParseFloat(cotacao.Usdbrl.Bid, 64)
	if err != nil {
		return err
	}

	coin := Coin{
		Code:      cotacao.Usdbrl.Code,
		CodeIn:    cotacao.Usdbrl.Codein,
		Descricao: cotacao.Usdbrl.Name,
		Valor:     valor,
	}

	db.Create(&coin)

	return nil
}
