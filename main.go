package main

import (
	"fmt"
	"net/http"
	"log"
	"time"
	"os"
	"strconv"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type Item struct {
	Id int`json:"id"`
	Host string`json:"host"`
}

var items [128]Item

func cleanupPods() {
	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}
	// creates the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}
	for {
		for _, v := range items {
			if v.Id != 0 {
				_, err = clientset.CoreV1().Pods(os.Getenv("NAMESPACE")).Get(v.Host, metav1.GetOptions{})
				if errors.IsNotFound(err) {
					// Delete from DynamoDB
					fmt.Println("Pod",v.Host,"not found. Removing from DynamoDB") 
					deleteDynamoDB(v.Host)
				} else if statusError, isStatus := err.(*errors.StatusError); isStatus {
					fmt.Printf("Error getting pod %v\n", statusError.ErrStatus.Message)
				} else if err != nil {
					panic(err.Error())
				}
			}
		}
		time.Sleep(10 * time.Minute)
	}
}

func deleteDynamoDB(host string) {
	sess, err := session.NewSession(&aws.Config{
	    Region: aws.String(os.Getenv("REGION"))},
	)

	// Create DynamoDB client
	svc := dynamodb.New(sess)
	input := &dynamodb.DeleteItemInput{
	    Key: map[string]*dynamodb.AttributeValue{
	        "host": {
	            S: aws.String(host),
	        },
	    },
	    TableName: aws.String(os.Getenv("NAMESPACE")+".nodeid-reservation-service"),
	}

	_, err = svc.DeleteItem(input)

	if err != nil {
	    fmt.Println("Got error calling DeleteItem")
	    fmt.Println(err.Error())
	    return
	}
}

func scanDynamoDB() {
	sess, err := session.NewSession(&aws.Config{
	    Region: aws.String(os.Getenv("REGION"))},
	)
	// Create DynamoDB client
	svc := dynamodb.New(sess)
	params := &dynamodb.ScanInput{
	    TableName: 				   aws.String(os.Getenv("NAMESPACE")+".nodeid-reservation-service"),
	}
	result, err := svc.Scan(params)
	for _, i := range result.Items {
		
	    item := Item{}

	    err = dynamodbattribute.UnmarshalMap(i, &item)
	    items[item.Id] = item
	    if err != nil {
	        fmt.Println("Got error unmarshalling:")
	        fmt.Println(err.Error())
	    }
	}
}

func updateDynamoDB(host string, id int) {
	sess, err := session.NewSession(&aws.Config{
	    Region: aws.String(os.Getenv("REGION"))},
	)
	svc := dynamodb.New(sess)
	input := &dynamodb.UpdateItemInput{
        ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
            ":id": {
                N: aws.String(strconv.Itoa(id)),
            },
        },
	    TableName: aws.String(os.Getenv("NAMESPACE")+".nodeid-reservation-service"),
	    Key: map[string]*dynamodb.AttributeValue{
	        "host": {
	            S: aws.String(host),
	        },
	    },
	    ReturnValues:     aws.String("UPDATED_NEW"),
	    UpdateExpression: aws.String("set id = :id"),
	}

	_, err = svc.UpdateItem(input)
	fmt.Println("Created new DynamoDB entry for host", host, "as id:", id)
	if err != nil {
	    fmt.Println(err.Error())
	}
}

func generateNodeID(host string) int {
	scanDynamoDB() // Load into struct array

	// First check if exists currently, if so, return with the ID
	for _, v := range items {
		if v.Host == host {
			return v.Id
		}
	}

	// Assign the next available NodeID
	for k, v := range items {
		if v.Id == 0 && k != 0 {
			updateDynamoDB(host, k)
			return k
		}
	}
	return 0
}

func handler(w http.ResponseWriter, r *http.Request) {
    fmt.Fprintf(w, "%d", generateNodeID(r.PostFormValue("host")))
}

func main() {

	go cleanupPods()
    http.HandleFunc("/", handler)
    log.Fatal(http.ListenAndServe(":8080", nil))
}
