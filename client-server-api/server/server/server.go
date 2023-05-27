package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	_ "github.com/mattn/go-sqlite3"
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
	ID        uint
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

	ctxCliente := r.Context()

	log.Println("Requisição iniciada...")
	ctxServer, cancel := context.WithTimeout(ctxCliente, 200*time.Millisecond)
	defer cancel()

	// time.Sleep(time.Second * 10)

	client := &http.Client{}
	req, err := http.NewRequest(http.MethodGet, URL_SERVICE, nil)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Erro ao criar requisição ao serviço de cotação: %v\n", err)
		log.Println("Erro ao criar requisição ao serviço de cotação...:", err)
		return
	}

	log.Println("Executando requisição ao serviço de cotação...")
	resp, err := client.Do(req.WithContext(ctxServer))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Erro ao fazer requisição ao serviço de cotação: %v\n", err)
		log.Println("Erro ao fazer requisição ao serviço de cotação...:", err)
		return
	}
	defer resp.Body.Close()

	log.Println("Lendo a resposta do serviço de cotação...")
	res, err := io.ReadAll(resp.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Erro ao ler resposta: %v\n", err)
		log.Println("Erro ao ler resposta do serviço de cotação...:", err)
		return
	}

	var cotacao CoinQuery
	err = json.Unmarshal(res, &cotacao)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Erro ao fazer parser da resposta: %v\n", err)
		log.Println("Erro ao fazer parser da resposta...:", err)
		return
	}

	err = salvarCotacaoBD(ctxServer, &cotacao)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Erro ao salvar cotação: %v\n", err)
		log.Println("Erro ao salvar cotação no BD...:", err)
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

	// time.Sleep(time.Second * 10)

	db, err := sql.Open("sqlite3", "./cotacao.db")
	if err != nil {
		return err
	}
	defer db.Close()

	stmt, err := db.Prepare("INSERT INTO coins (code, code_in, descricao, valor) VALUES ($1, $2, $3, $4)")
	if err != nil {
		log.Println("Erro ao salvar cotação no BD...:", err)
		return err
	}
	defer stmt.Close()

	_, err = stmt.ExecContext(ctxBD, cotacao.Usdbrl.Code, cotacao.Usdbrl.Codein, cotacao.Usdbrl.Name, cotacao.Usdbrl.Bid)
	if err != nil {
		if err == context.DeadlineExceeded {
			log.Println("Timeout excedido no BD...:", err)
		}
		return err
	}

	return nil
}
