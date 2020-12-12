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

// Initializing global variable client of type mongo client
var client *mongo.Client

// Initializing global variable obj_id of type ObjectID
var obj_id primitive.ObjectID

type url_struct struct {
    // Key name should start with capital letter e.g -: 'Name', 'Time', etc
	Url         string `json:"url"`
}

type id_struct struct{
    Id          primitive.ObjectID `json:"_id,omitempty" bson:"_id,omitempty"`
}

type result_struct struct{
    Id           primitive.ObjectID `json:"_id,omitempty" bson:"_id,omitempty"`
    Url          string `json:"url,omitempty" bson:"url,omitempty"`
    Name         string `json:"name,omitempty" bson:"name,omitempty"`
    Reviews      string `json:"reviews,omitempty" bson:"reviews,omitempty"`
    Price        string `json:"price,omitempty" bson:"price,omitempty"`
    Discription  string `json:"discription,omitempty" bson:"discription,omitempty"`
    Image        string `json:"image,omitempty" bson:"image,omitempty"`
    Created_at   time.Time `json:"created_at,omitempty" bson:"created_at,omitempty"`
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

	// Initializing variables
	var productName string
    var productReviews string
    var productPrice string
    var productDescription string
    var productImage string

    // Providing CSS selectors and scraping required data.
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
    resp, err := http.Post("http://web2:8081/save",
        "application/json", bytes.NewBuffer(newReqBody))
    if err != nil {
        print(err)
    }
    defer resp.Body.Close()

    body, err := ioutil.ReadAll(resp.Body)

    // Removing double quotes from first and last position otherwise invalid hex error will appear
    newVal := string(body)[1 : len(string(body))-2]

    // converting string to ObjectID
    obj_id, err = primitive.ObjectIDFromHex(newVal)
    if err !=nil{
        fmt.Println(err)
    }
    fmt.Println(obj_id)

    // Searching the recently created object to be returned as the json response
    w.Header().Set("content-type", "application/json")
	var result_one result_struct
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)

    quickstartDatabase := client.Database("test")
    scrapedCollection := quickstartDatabase.Collection("scrapes")

    // Finding the document by id to be returned in response
	err = scrapedCollection.FindOne(ctx, result_struct{Id: obj_id}).Decode(&result_one)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{ "message": "` + err.Error() + `" }`))
		return
	}
    // Encoding the result to JSON basically adding a logical structure to raw bytes
	json.NewEncoder(w).Encode(result_one)
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
	// Delaying the execution until we get the result for the function, our cursor
	defer cursor.Close(ctx)

	// Looping through cursor
	for cursor.Next(ctx) {
		var result_one result_struct
		// To decode into a struct, use cursor.Decode()
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

	// Provided the URI here, instead we can use a environment variables to input the URI
	clientOptions := options.Client().ApplyURI("mongodb+srv://gouser:gouser@cluster0.ar69a.mongodb.net/test?retryWrites=true&w=majority")
	client, _ = mongo.Connect(ctx, clientOptions)

	// Initializing the router for our api endpoints
	router := mux.NewRouter().StrictSlash(true)
	router.HandleFunc("/scrape_amazon", scrapelink).Methods("POST")
	router.HandleFunc("/all", GetPeopleEndpoint).Methods("GET")
	log.Fatal(http.ListenAndServe(":8080", router))
}