package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
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

	// time.Sleep(time.Second * 10)

	ctxCliente := r.Context()

	log.Println("Requisição iniciada...")
	ctxServer, cancel := context.WithTimeout(ctxCliente, 200*time.Millisecond)
	defer cancel()

	client := &http.Client{}
	req, err := http.NewRequest(http.MethodGet, URL_SERVICE, nil)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Erro ao criar requisição ao serviço de cotação: %v\n", err)
		log.Println("Erro ao criar requisição ao serviço de cotação...")
		return
	}

	log.Println("Executando requisição ao serviço de cotação...")
	resp, err := client.Do(req.WithContext(ctxServer))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Erro ao fazer requisição ao serviço de cotação: %v\n", err)
		log.Println("Erro ao fazer requisição ao serviço de cotação...")
		return
	}
	defer resp.Body.Close()

	log.Println("Lendo a resposta do serviço de cotação...")
	res, err := io.ReadAll(resp.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Erro ao ler resposta: %v\n", err)
		log.Println("Erro ao ler resposta do serviço de cotação...")
		return
	}

	var cotacao CoinQuery
	err = json.Unmarshal(res, &cotacao)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Erro ao fazer parser da resposta: %v\n", err)
		log.Println("Erro ao fazer parser da resposta...")
		return
	}

	err = salvarCotacaoBD(ctxServer, &cotacao)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Erro ao salvar cotação: %v\n", err)
		log.Println("Erro ao salvar cotação no BD...")
		return
	}

	log.Println("Enviando resposta ao client...")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(cotacao.Usdbrl.Bid)
}

func salvarCotacaoBD(ctxParent context.Context, cotacao *CoinQuery) error {

	ctxBD, cancel := context.WithTimeout(ctxParent, 10*time.Millisecond)
	defer cancel()

	db, err := gorm.Open(sqlite.Open("cotacao.db"), &gorm.Config{})
	if err != nil {
		return err
	}
	db = db.WithContext(ctxBD)

	// Migrate the schema
	// db.AutoMigrate(&Coin{})

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

	result := db.Create(&coin)
	if result.Error != nil {
		if err == context.DeadlineExceeded {
			log.Println("Timeout excedido no BD...")
			return err
		}

		return err
	}

	return nil
}
