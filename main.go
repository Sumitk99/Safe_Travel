package main

import (
	"fmt"
	"github.com/gorilla/mux"
	"github.com/twilio/twilio-go"
	openapi "github.com/twilio/twilio-go/rest/api/v2010"
	"log"
	"net/http"
	"os"
	"time"
)

var fromPhoneNumber = os.Getenv("TWILIO_PHONE_NUMBER")
var toPhoneNumber = os.Getenv("CLIENT_PHONE_NUMBER")
var alertReceiver = os.Getenv("ALERT_RECEIVER")
var accountSid = os.Getenv("TWILIO_ACCOUNT_SID")
var authToken = os.Getenv("TWILIO_AUTH_TOKEN")

type AlertMessage struct {
	To      string `json:"to"`
	From    string `json:"from"`
	Message string `json:"message"`
}

func main() {
	router := mux.NewRouter()
	router.HandleFunc("/voice", voiceHandler).Methods("POST")
	router.HandleFunc("/gather", gatherHandler).Methods("POST")

	go startScheduler()

	log.Println("Starting server on :8080")
	log.Fatal(http.ListenAndServe(":8080", router))
}

func startScheduler() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		<-ticker.C
		log.Printf("Triggering call at %s\n", time.Now().Format(time.RFC3339))
		triggerCall()
	}
}

func triggerCall() {
	client := twilio.NewRestClientWithParams(twilio.ClientParams{
		Username: accountSid,
		Password: authToken,
	})

	params := &openapi.CreateCallParams{}
	params.SetTo(toPhoneNumber)
	params.SetFrom(fromPhoneNumber)
	params.SetUrl(os.Getenv("ONLINE_URL"))

	resp, err := client.Api.CreateCall(params)
	if err != nil {
		log.Println("Error triggering call:", err)
	} else {
		log.Println("Call triggered:", *resp.Sid)
	}
}

func voiceHandler(w http.ResponseWriter, r *http.Request) {
	response := `
        <Response>
            <Gather input="dtmf" timeout="10" numDigits="1" action="/gather">
                <Say>Please Enter the code</Say>
            </Gather>
        </Response>`
	w.Header().Set("Content-Type", "application/xml")
	w.Write([]byte(response))
}

func gatherHandler(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		log.Printf("Error parsing form data: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	digits := r.FormValue("Digits")
	log.Printf("User Response : %s", digits)

	var response string
	if digits == os.Getenv("SECURITY_CODE") {
		response = "Thank you. Your code is correct."
	} else {
		response = fmt.Sprintf("Alert, Incorrect Code entered by %s.", toPhoneNumber)
	}
	sendAlert(response)
}

func sendAlert(CodeStatus string) {
	client := twilio.NewRestClientWithParams(twilio.ClientParams{
		Username: accountSid,
		Password: authToken,
	})

	alert := AlertMessage{
		To:      alertReceiver,
		From:    fromPhoneNumber,
		Message: CodeStatus,
	}

	params := &openapi.CreateMessageParams{}
	params.SetTo(alert.To)
	params.SetFrom(alert.From)
	params.SetBody(alert.Message)

	_, err := client.Api.CreateMessage(params)
	if err != nil {
		log.Println("Error sending alert:", err)
	} else {
		log.Println("Alert sent successfully")
	}
}
