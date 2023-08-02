package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"fyne.io/fyne/v2/dialog"
)

type Ingredient struct {
	Name     string  `json:"name"`
	Quantity float64 `json:"quantity"`
	Unit     string  `json:"unit"`
	Notes    string  `json:"notes"`
}

type Recipe struct {
	Id              string       `json:"_id,omitempty"`
	Title           string       `json:"title"`
	Description     string       `json:"description"`
	Category        string       `json:"category"`
	Country         string       `json:"country"`
	MainIngredient  string       `json:"mainingredient"`
	PrepTime        int          `json:"preptime"`
	DefaultPortions int          `json:"defaultportions"`
	Ingredients     []Ingredient `json:"ingredients"`
	Image           []byte       `json:"image"`
}

func (recipe Recipe) addNewRecipe() bool {

	httpClient := http.Client{}

	body := map[string]interface{}{
		"dataSource": "mongodb-atlas",
		"database":   credentials["database"],
		"collection": credentials["collection"],
		"document":   recipe,
	}

	jsonBody, err := json.Marshal(body)

	// Display popup error
	if err != nil {
		errorDialog := dialog.NewError(err, mainWindow)
		errorDialog.Show()
		return false
	}

	req, err := http.NewRequest("POST", "https://eu-central-1.aws.data.mongodb-api.com/app/"+credentials["appId"]+"/endpoint/data/v1/action/insertOne", bytes.NewBuffer(jsonBody))

	req.Header.Add("Accept", "application/json")
	req.Header.Add("content-type", "application/json")
	req.Header.Add("email", credentials["email"])
	req.Header.Add("password", credentials["password"])

	if err != nil {
		errorDialog := dialog.NewError(err, mainWindow)
		errorDialog.Show()
		return false
	}

	rawResponse, err := httpClient.Do(req)

	if err != nil {
		errorDialog := dialog.NewError(err, mainWindow)
		errorDialog.Show()
		return false

	} else if rawResponse.StatusCode != 201 {

		defer rawResponse.Body.Close()
		responseBody, _ := ioutil.ReadAll(rawResponse.Body)
		errorDialog := dialog.NewInformation("Error", "Insert failed: "+fmt.Sprint(rawResponse.StatusCode)+string(responseBody), mainWindow)
		errorDialog.Show()
		return false

	} else {
		successDialog := dialog.NewInformation("OK", "Recipe succesfully added!", mainWindow)
		successDialog.Show()
		return true
	}

}

func (recipe Recipe) updateRecipe(documentId string) bool {

	httpClient := http.Client{}

	// Set filter to current recipe's ID
	filter := map[string]map[string]string{
		"_id": {
			"$oid": documentId,
		},
	}

	body := map[string]interface{}{
		"dataSource": "mongodb-atlas",
		"database":   credentials["database"],
		"collection": credentials["collection"],
		"filter":     filter,
		"update":     map[string]Recipe{"$set": recipe},
	}

	jsonBody, err := json.Marshal(body)

	if err != nil {
		errorDialog := dialog.NewError(err, mainWindow)
		errorDialog.Show()
		return false
	}

	req, err := http.NewRequest("POST", "https://eu-central-1.aws.data.mongodb-api.com/app/"+credentials["appId"]+"/endpoint/data/v1/action/updateOne", bytes.NewBuffer(jsonBody))

	req.Header.Add("Accept", "application/json")
	req.Header.Add("content-type", "application/json")
	req.Header.Add("email", credentials["email"])
	req.Header.Add("password", credentials["password"])

	if err != nil {
		errorDialog := dialog.NewError(err, mainWindow)
		errorDialog.Show()
		return false
	}

	rawResponse, err := httpClient.Do(req)

	if err != nil {
		errorDialog := dialog.NewError(err, mainWindow)
		errorDialog.Show()
		return false

	} else if rawResponse.StatusCode != 200 {

		defer rawResponse.Body.Close()
		responseBody, _ := ioutil.ReadAll(rawResponse.Body)
		errorDialog := dialog.NewInformation("Error", "Update failed: "+fmt.Sprint(rawResponse.StatusCode)+string(responseBody), mainWindow)
		errorDialog.Show()
		return false

	} else {
		successDialog := dialog.NewInformation("OK", "Recipe succesfully updated!", mainWindow)
		successDialog.Show()
		return true
	}
}
