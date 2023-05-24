package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const URL_SERVICE = "http://localhost:8080/cotacao"

func main() {
	err := makeRequest()
	if err != nil {
		panic(err)
	}
}

func makeRequest() error {
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, URL_SERVICE, nil)
	if err != nil {
		return err
	}

	fmt.Println("Executando requisição...")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	fmt.Println("Lendo a resposta...")
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}

	valor := strings.TrimSpace(string(body))
	valor = strings.ReplaceAll(valor, "\"", "")

	bidValue, err := strconv.ParseFloat(valor, 64)
	if err != nil {
		return err
	}

	err = saveBid(bidValue)
	if err != nil {
		return err
	}

	return nil
}

func saveBid(bidValue float64) error {

	fmt.Println("Abrindo arquivo...")
	arquivo, err := os.OpenFile("cotacao.txt", os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer arquivo.Close()

	fmt.Println("Escrevendo no arquivo...")
	n, err := arquivo.WriteString(fmt.Sprintf("Dólar: {%.4f}\n", bidValue))
	if err != nil {
		return err
	}
	fmt.Printf("%d bytes writen\n", n)

	return nil
}
