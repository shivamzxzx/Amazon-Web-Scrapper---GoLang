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
	"io/ioutil"
	"encoding/json"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Initializing global variable client of type mongo client
var client *mongo.Client

// Initializing global variable obj_id of type ObjectID
var obj_id primitive.ObjectID

type scrape_struct struct{
    Url          string `json:"url"`
    Name         string `json:"name"`
    Reviews      string `json:"reviews"`
    Price        string `json:"price"`
    Discription  string `json:"discription"`
    Image        string `json:"image"`
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
    // Pushing scraped data to our collection
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

    // Assigning scrapedResult.InsertedID which is a interface as primitive.ObjectID
    obj_id = scrapedResult.InsertedID.(primitive.ObjectID)
    w.Header().Set("content-type", "application/json")
    json.NewEncoder(w).Encode(obj_id)

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
	router.HandleFunc("/save", saveData).Methods("POST")
	log.Fatal(http.ListenAndServe(":8081", router))
}