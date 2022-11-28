package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
)

const (
	EnvTeamsWebhook  = "TEAMS_WEBHOOK"
	EnvTeamsTitle    = "TEAMS_TITLE"
	EnvTeamsMessage  = "TEAMS_MESSAGE"
	EnvTeamsColor    = "TEAMS_COLOR"
	EnvTeamsCardMode = "TEAMS_CARD_MODE" // TEXT (default) | TEMPLATE
	EnvTeamsCardPath = "TEAMS_CARD_PATH" // Template path
)

func main() {
	endpoint := os.Getenv(EnvTeamsWebhook)
	if endpoint == "" {
		fmt.Fprintln(os.Stderr, "URL is required")
		os.Exit(1)
	}
	mode := os.Getenv(EnvTeamsCardMode)

	var msg map[string]any
	if mode == "" || mode == "TEXT" {
		msg = prepareLegacyCard()
	} else if mode == "TEMPLATE" {
		msg = prepareAdaptiveCard()
	} else {
		fmt.Fprintf(os.Stderr, "Unkown mode: %s\n", mode)
		os.Exit(1)
	}

	// Send the prepared message
	if err := send(endpoint, msg); err != nil {
		fmt.Fprintf(os.Stderr, "Error sending message: %s\n", err)
		os.Exit(2)
	}
}

func prepareLegacyCard() map[string]any {
	text := os.Getenv(EnvTeamsMessage)
	if text == "" {
		fmt.Fprintln(os.Stderr, "Message is required")
		os.Exit(1)
	}

	// Reference fields https://learn.microsoft.com/en-us/outlook/actionable-messages/message-card-reference
	msg := map[string]any{
		"title":      os.Getenv(EnvTeamsTitle),
		"text":       text,
		"themeColor": os.Getenv(EnvTeamsColor),
	}

	return msg
}

func prepareAdaptiveCard() map[string]any {
	path := os.Getenv(EnvTeamsCardPath)
	if path == "" {
		fmt.Fprintln(os.Stderr, "Card path is required")
		os.Exit(1)
	}

	// Template fields https://learn.microsoft.com/en-us/outlook/actionable-messages/adaptive-card
	// Load template
	jsonFile, err := os.Open(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to load card template:\n%s", err)
		os.Exit(2)
	}
	defer jsonFile.Close()
	byteValue, _ := ioutil.ReadAll(jsonFile)
	var tmpl map[string]any
	jsonErr := json.Unmarshal(byteValue, &tmpl)
	if jsonErr != nil {
		fmt.Fprintf(os.Stderr, "Card template is not a valid json file:\n%s", jsonErr)
		os.Exit(2)
	}

	// Add replacing parameters ?

	// Create output object
	att := map[string]any{
		"contentType": "application/vnd.microsoft.card.adaptive",
		"contentUrl":  nil,
		"content":     tmpl,
	}
	attachments := make([]map[string]any, 1)
	attachments[0] = att
	msg := map[string]any{
		"type":        "message",
		"attachments": attachments,
	}

	return msg
}

func send(endpoint string, msg map[string]any) error {
	enc, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	if endpoint == "console" {
		enc, err = json.MarshalIndent(msg, "", "  ")
		fmt.Fprintf(os.Stdout, "JSON message sent to the webhook:\n%s", enc)
		return nil
	}

	b := bytes.NewBuffer(enc)
	res, err := http.Post(endpoint, "application/json", b)
	if err != nil {
		return err
	}

	if res.StatusCode >= 299 {
		return fmt.Errorf("Error on message: %s\n", res.Status)
	}
	fmt.Println(res.Status)
	return nil
}
