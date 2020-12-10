package main

import (
	"fmt"
	"log"
	"net/http"
    "github.com/gorilla/mux"
	"context"
	"time"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"github.com/gocolly/colly"
	"io/ioutil"
	"encoding/json"
	"bytes"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var client *mongo.Client


type url_struct struct {
    // Below key name should start with capital letter e.g -: 'Name', 'Time', etc
	Url         string `json:"url"`
}

type scrape_struct struct{
    Url          string `json:"url"`
    Name         string `json:"name"`
    Reviews      string `json:"reviews"`
    Price        string `json:"price"`
    Discription  string `json:"discription"`
    Image        string `json:"image"`
}

type result_struct struct{
    Id           primitive.ObjectID `json:"_id,omitempty" bson:"_id,omitempty"`
    Url          string `json:"url"`
    Name         string `json:"name"`
    Reviews      string `json:"reviews"`
    Price        string `json:"price"`
    Discription  string `json:"discription"`
    Image        string `json:"image"`
    Created_at   time.Time `json:"created_at"`
}

var obj_id primitive.ObjectID

type response_struct struct{
    Id           primitive.ObjectID `json:"_id,omitempty" bson:"_id,omitempty"`
    Url          string `json:"url,omitempty" bson:"url,omitempty"`
    Name         string `json:"name,omitempty" bson:"name,omitempty"`
    Reviews      string `json:"reviews,omitempty" bson:"reviews,omitempty"`
    Price        string `json:"price,omitempty" bson:"price,omitempty"`
    Discription  string `json:"discription,omitempty" bson:"discription,omitempty"`
    Image        string `json:"image,omitempty" bson:"image,omitempty"`
    Created_at   time.Time `json:"created_at,omitempty" bson:"created_at,omitempty"`
}

func homeLink(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Welcome!")
}

func scrapelink(w http.ResponseWriter, r *http.Request) {

    // Parsing Request Json
    reqBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		fmt.Fprintf(w, "No data provided!")
	}
	fmt.Printf("Request body: %s \n", reqBody)


    var u url_struct
    err = json.Unmarshal(reqBody, &u)
    if err != nil{
        fmt.Println(err)
    }

    // Scraping given webpage
    c := colly.NewCollector()

	c.OnRequest(func(r *colly.Request) {
		fmt.Println("Visiting", r.URL)
	})
	var productName string
    var productReviews string
    var productPrice string
    var productDescription string
    var productImage string

	c.OnHTML("div#ppd", func(e *colly.HTMLElement) {
        productName = e.ChildText("span#productTitle")
        productDescription = e.ChildText("div#feature-bullets > ul > li:nth-child(4) > span")
        productPrice = e.ChildText("div#olp-upd-new-used > span > a > span.a-size-base.a-color-price")
        productReviews = e.ChildText("span#acrCustomerReviewText")

        fmt.Printf("Product Name: %s \n", productName)
        fmt.Printf("Product Reviews: %s \n", productReviews)
        fmt.Printf("Product Price: %s \n", productPrice)
        fmt.Printf("Product Discription: %s \n", productDescription)
	})

	c.OnHTML("div#imgTagWrapperId", func(e *colly.HTMLElement) {

        productImage = e.ChildAttr("img", "data-old-hires")
        fmt.Printf("Product Image: %s \n", productImage)
    })


	c.Visit(u.Url)

	// Creating byte data to be passed as object to the POST api to save the data
	newReqBody, err := json.Marshal(map[string]string{
	            "url": u.Url,
                "name": productName,
                "reviews": productReviews,
                "price": productPrice,
                "discription": productDescription,
                "image": productImage,
            })

    // Making request to the POST api to save the data
    fmt.Printf("Product Image new: %s \n", productImage)
    resp, err := http.Post("http://localhost:8080/save",
        "application/json", bytes.NewBuffer(newReqBody))
    if err != nil {
        print(err)
    }
    fmt.Println(resp)
    defer resp.Body.Close()

    fmt.Println(obj_id)

    // Searching the recently created object to be returned as the json response
    w.Header().Set("content-type", "application/json")
	var result_one response_struct
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)

    quickstartDatabase := client.Database("test")
    scrapedCollection := quickstartDatabase.Collection("scrapes")
	err = scrapedCollection.FindOne(ctx, response_struct{Id: obj_id}).Decode(&result_one)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{ "message": "` + err.Error() + `" }`))
		return
	}
	json.NewEncoder(w).Encode(result_one)
}

func saveData(w http.ResponseWriter, r *http.Request) {

    //POST api to save data
    reqBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		fmt.Fprintf(w, "No data provided!")
	}
	fmt.Printf("Request body save: %s \n", reqBody)

	var scraped_data scrape_struct
    err = json.Unmarshal(reqBody, &scraped_data)
    if err != nil{
        fmt.Println(err)
    }
    ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)

    quickstartDatabase := client.Database("test")
    scrapedCollection := quickstartDatabase.Collection("scrapes")
    created_at := time.Now()
    scrapedResult, err := scrapedCollection.InsertOne(ctx, bson.D{
        {Key: "url", Value: scraped_data.Url},
        {Key: "name", Value: scraped_data.Name},
        {Key: "reviews", Value: scraped_data.Reviews},
        {Key: "price", Value: scraped_data.Price},
        {Key: "discription", Value: scraped_data.Discription},
        {Key: "image", Value: scraped_data.Image},
        {Key: "created_at", Value: created_at},
    })

            if err != nil {
                log.Fatal(err)
            }
    fmt.Printf("Inserted %v documents into pod collection!\n", scrapedResult.InsertedID)
    obj_id = scrapedResult.InsertedID.(primitive.ObjectID)
    w.Header().Set("Content-Type", "application/json")

    json.NewEncoder(w).Encode(scrapedResult)

}

func GetPeopleEndpoint(response http.ResponseWriter, request *http.Request) {

    // Fetching all the objects created and return the json object
    ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
    quickstartDatabase := client.Database("test")
    scrapedCollection := quickstartDatabase.Collection("scrapes")

	response.Header().Set("content-type", "application/json")
	var result []result_struct

	cursor, err := scrapedCollection.Find(ctx, bson.M{})
	if err != nil {
		response.WriteHeader(http.StatusInternalServerError)
		response.Write([]byte(`{ "message": "` + err.Error() + `" }`))
		return
	}
	defer cursor.Close(ctx)
	for cursor.Next(ctx) {
		var result_one result_struct
		cursor.Decode(&result_one)
		result = append(result, result_one)
	}
	if err := cursor.Err(); err != nil {
		response.WriteHeader(http.StatusInternalServerError)
		response.Write([]byte(`{ "message": "` + err.Error() + `" }`))
		return
	}
	json.NewEncoder(response).Encode(result)
}


func main() {
    fmt.Println("Starting the application...")
    // Initializing the db instance
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	clientOptions := options.Client().ApplyURI("mongodb+srv://gouser:gouser@cluster0.ar69a.mongodb.net/test?retryWrites=true&w=majority")
	client, _ = mongo.Connect(ctx, clientOptions)
	// Initializing the router for our api endpoints
	router := mux.NewRouter().StrictSlash(true)
	router.HandleFunc("/", homeLink).Methods("GET")
	router.HandleFunc("/scrape_amazon", scrapelink).Methods("POST")
	router.HandleFunc("/save", saveData).Methods("POST")
	router.HandleFunc("/all", GetPeopleEndpoint).Methods("GET")
	log.Fatal(http.ListenAndServe(":8080", router))
}