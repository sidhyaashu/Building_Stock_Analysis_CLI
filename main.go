package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"slices"
	"strconv"
	"time"
)

type Stock struct {
	Ticker string
	Gap float64
	OpeningPrice float64
}

func Load(path string) ([]Stock,error){
	f,err := os.Open(path)

	if err != nil {
		fmt.Println(err)
		return nil ,err
	}

	defer f.Close()

	r := csv.NewReader(f)
	rows,err := r.ReadAll()

	if err != nil {
		fmt.Println(err)
		return nil,err
	}


	rows = slices.Delete(rows,0,1)

	var stocks []Stock

	for _, row := range rows {
		ticker := row[0]
		gap , err := strconv.ParseFloat(row[1],64) 
		if err != nil {
			continue
		}
		openingPrice , err := strconv.ParseFloat(row[2],64) 
		if err != nil {
			continue
		}


		stocks = append(stocks, Stock{
			Ticker: ticker,
			Gap: gap,
			OpeningPrice: openingPrice,
		})
	}

	return stocks , nil
}


var accountBalance = 10000.0
var lossTolerance = .02
var maxLossPerTrade = accountBalance * lossTolerance
var ProfitPercent = .8

type Position struct{
	EntryPrice float64
	Shares int
	TakeProfitPrice float64
	StopLossPrice float64
	Profit float64
}


func Calculate(gapPercent,openingPrice float64) Position {
	closingPrice := openingPrice / (1 + gapPercent)
	gapValue := closingPrice - openingPrice
	profitFromGap := ProfitPercent * gapValue

	stopLoss := openingPrice - profitFromGap
	takeProfit := openingPrice + profitFromGap

	shares := int(maxLossPerTrade/math.Abs(stopLoss-openingPrice))

	profit := math.Abs(openingPrice - takeProfit) * float64(shares)
	profit = math.Round(profit*100) / 100


	return Position{
		EntryPrice: math.Round(openingPrice *100)/100,
		Shares: shares,
		TakeProfitPrice: math.Round(takeProfit*100)/100,
		StopLossPrice: math.Round(stopLoss*100)/100 ,
		Profit: math.Round(profit*100)/100,
	}
}

type Selection struct {
	Ticker string
	Position
	Articles []Article
}

const (
	url = "https://seeking-alpha.p.rapidapi.com/news/v2/list-by-symbol?size=56&id="
	apiKeyHeader = "X-RapidAPI-Key"
	apiKey = "Enter Your Key"
)

type attributes struct {
	PublishOn time.Time `json:"publishOn"`
	Title string `json:"title"`
}

type SeekingAlphaNews struct {
	Attributes attributes `json:"attributes`
}

type SeekingAlphaResponse struct {
	Data []SeekingAlphaNews `json:"data"`
}

type Article struct {
	PublishOn time.Time 
	Headline string
}


func FetchNews(ticker string) ([]Article , error) {
	req, err := http.NewRequest(http.MethodGet, url+ticker,nil)

	if err != nil {
		return nil, err
	}

	req.Header.Add(apiKeyHeader,apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}


	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil , fmt.Errorf(" Unsuccesfull status code %d received", resp.StatusCode)
	}

	res := &SeekingAlphaResponse{}
	json.NewDecoder(resp.Body).Decode(res)

	var articles []Article

	for _, item := range res.Data {
		art := Article{
			PublishOn: item.Attributes.PublishOn,
			Headline: item.Attributes.Title,
		}

		articles = append(articles, art)
	}

	return articles, nil
}

func Deliver(filePath string,selection []Selection) error {
	file, err := os.Create(filePath)

	if err != nil{
		return fmt.Errorf("Error  creating file : %w",err)

	}

	defer file.Close()

	encoder := json.NewEncoder(file)
	err = encoder.Encode(selection)
	if err != nil {
		return fmt.Errorf("Error encoding selection : %w",err)

	}

	return nil
}

func main(){

	stocks, err := Load("./opg.csv")

	if err != nil {
		fmt.Println(err)
		return
	}

	slices.DeleteFunc(stocks, func(s Stock) bool {
		return math.Abs(s.Gap) < .1
	})

	var selection []Selection

	for _, stock := range stocks {
		position := Calculate(stock.Gap,stock.OpeningPrice)

		articles, err := FetchNews(stock.Ticker)
		if err != nil {
			log.Printf("Erro loading news about %s , %v", stock.Ticker,err)
			continue
		}else{
			log.Printf("Found %d articles about %s", len(articles),stock.Ticker)
		}

		sel := Selection{
			Ticker: stock.Ticker,
			Position: position,
			Articles: articles,
		}

		selection = append(selection, sel)


	}

	outputPath := "./opg.json"

	err = Deliver(outputPath,selection)

	if err != nil {
		log.Printf("Error writing output %v",err)
		return

	}

	log.Printf("Finish writing Output %s\n",outputPath)
	
}