package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"

	"fyne.io/fyne/v2/dialog"
)

type pipelineStage map[string]interface{}

type getRecipesResponse struct {
	Documents []struct {
		Recipes    []Recipe
		TotalCount []map[string]int
	}
}

type getRecipesByTextResponse struct {
	Documents []struct {
		Docs []Recipe
		Meta []map[string]map[string]int
	}
}

// getDistinctFieldValues returns an array of all distinct values of a particular field that exist in the collection
func getDistinctFieldValues(fieldName string) []string {

	httpClient := http.Client{}

	// Group documents by chosen field
	groupStage := make(map[string]interface{})
	groupStage["$group"] = map[string]string{"_id": "$" + fieldName}

	// Pipeline with one stage only - group
	var pipeline []pipelineStage
	pipeline = append(pipeline, groupStage)

	body := map[string]interface{}{
		"dataSource": "mongodb-atlas",
		"database":   credentials["database"],
		"collection": credentials["collection"],
		"pipeline":   pipeline,
	}

	jsonBody, _ := json.Marshal(body)

	req, err := http.NewRequest("POST", "https://eu-central-1.aws.data.mongodb-api.com/app/"+credentials["appId"]+"/endpoint/data/v1/action/aggregate", bytes.NewBuffer(jsonBody))

	req.Header.Add("Accept", "application/json")
	req.Header.Add("email", credentials["email"])
	req.Header.Add("password", credentials["password"])

	if err != nil {
		errorDialog := dialog.NewError(err, mainWindow)
		errorDialog.Show()
		return []string{}
	}

	rawResponse, err := httpClient.Do(req)

	if err != nil {
		errorDialog := dialog.NewError(err, mainWindow)
		errorDialog.Show()
		return []string{}
	}

	defer rawResponse.Body.Close()

	var response map[string][]map[string]string

	err = json.NewDecoder(rawResponse.Body).Decode(&response)

	if err != nil {
		errorDialog := dialog.NewError(err, mainWindow)
		errorDialog.Show()
		return []string{}
	}

	fieldValues := []string{}

	for _, doc := range response["documents"] {
		fieldValues = append(fieldValues, doc["_id"])
	}

	return fieldValues
}

func getRecipes(fieldName string, fieldValue string, offset int, perPage int) (results []Recipe, totalCount int) {

	httpClient := http.Client{}

	var matchStage pipelineStage

	if fieldName == "" {
		matchStage = pipelineStage{"$match": map[string]string{}} // Match all documents

	} else {
		matchStage = pipelineStage{"$match": map[string]string{fieldName: fieldValue}}
	}

	skipStage := pipelineStage{"$skip": offset}
	limitStage := pipelineStage{"$limit": perPage}
	countStage := pipelineStage{"$count": "totalCount"}

	// Two pipelines - one for a limited number of documents, the other for the count of all matched documents
	resultPipeline := []pipelineStage{matchStage, skipStage, limitStage}
	countPipeline := []pipelineStage{matchStage, countStage}

	combinedPipeline := []map[string]map[string][]pipelineStage{{
		"$facet": {
			"recipes":    resultPipeline,
			"totalCount": countPipeline,
		},
	}}

	body := map[string]interface{}{
		"dataSource": "mongodb-atlas",
		"database":   credentials["database"],
		"collection": credentials["collection"],
		"pipeline":   combinedPipeline,
	}

	jsonBody, _ := json.Marshal(body)

	req, err := http.NewRequest("POST", "https://eu-central-1.aws.data.mongodb-api.com/app/"+credentials["appId"]+"/endpoint/data/v1/action/aggregate", bytes.NewBuffer(jsonBody))

	if err != nil {
		errorDialog := dialog.NewError(err, mainWindow)
		errorDialog.Show()
		return []Recipe{}, 0
	}

	req.Header.Add("Accept", "application/json")
	req.Header.Add("email", credentials["email"])
	req.Header.Add("password", credentials["password"])

	rawResponse, err := httpClient.Do(req)

	if err != nil {
		errorDialog := dialog.NewError(err, mainWindow)
		errorDialog.Show()
		return []Recipe{}, 0
	}

	defer rawResponse.Body.Close()

	responseBody, err := ioutil.ReadAll(rawResponse.Body)

	if err != nil {
		errorDialog := dialog.NewError(err, mainWindow)
		errorDialog.Show()
		return []Recipe{}, 0
	}

	var response getRecipesResponse

	if err := json.Unmarshal(responseBody, &response); err != nil {
		errorDialog := dialog.NewError(err, mainWindow)
		errorDialog.Show()
		return []Recipe{}, 0
	}

	if len(response.Documents[0].Recipes) == 0 {
		return []Recipe{}, 0

	} else {
		return response.Documents[0].Recipes, response.Documents[0].TotalCount[0]["totalCount"]
	}
}

// getRecipesByText uses Atlas Search to perform full text search on documents
func getRecipesByText(searchTerm string, offset int, perPage int) (results []Recipe, totalCount int) {

	httpClient := http.Client{}

	searchStage := pipelineStage{"$search": map[string]interface{}{
		"text": map[string]interface{}{
			"path":  map[string]string{"wildcard": "*"}, // Search in all fields
			"query": searchTerm,
		},
		"count": map[string]string{"type": "total"},
	}}

	skipStage := pipelineStage{"$skip": offset}
	limitStage := pipelineStage{"$limit": perPage}

	// Get SEARCH_META metadata that contain count of all matched documents
	countStage := pipelineStage{"$facet": map[string]interface{}{
		"docs": []int{},
		"meta": []interface{}{map[string]string{"$replaceWith": "$$SEARCH_META"}, map[string]int{"$limit": 1}},
	}}

	pipeline := []pipelineStage{searchStage, skipStage, limitStage, countStage}

	body := map[string]interface{}{
		"dataSource": "mongodb-atlas",
		"database":   credentials["database"],
		"collection": credentials["collection"],
		"pipeline":   pipeline,
	}

	jsonBody, _ := json.Marshal(body)

	req, err := http.NewRequest("POST", "https://eu-central-1.aws.data.mongodb-api.com/app/"+credentials["appId"]+"/endpoint/data/v1/action/aggregate", bytes.NewBuffer(jsonBody))

	if err != nil {
		errorDialog := dialog.NewError(err, mainWindow)
		errorDialog.Show()
		return []Recipe{}, 0
	}

	req.Header.Add("Accept", "application/json")
	req.Header.Add("email", credentials["email"])
	req.Header.Add("password", credentials["password"])

	rawResponse, err := httpClient.Do(req)

	if err != nil {
		errorDialog := dialog.NewError(err, mainWindow)
		errorDialog.Show()
		return []Recipe{}, 0
	}

	defer rawResponse.Body.Close()

	responseBody, err := ioutil.ReadAll(rawResponse.Body)

	if err != nil {
		errorDialog := dialog.NewError(err, mainWindow)
		errorDialog.Show()
		return []Recipe{}, 0
	}
	var response getRecipesByTextResponse

	if err := json.Unmarshal(responseBody, &response); err != nil {
		errorDialog := dialog.NewError(err, mainWindow)
		errorDialog.Show()
		return []Recipe{}, 0
	}

	docs := response.Documents[0]

	if len(docs.Docs) == 0 {
		return docs.Docs, 0

	} else {
		return docs.Docs, docs.Meta[0]["count"]["total"]
	}

}

// validateMongoLogin checks input credentials by making a query for one document
func validateMongoLogin(appId string, db string, coll string, email string, pass string) (statusCode int) {

	httpClient := http.Client{}

	body := map[string]string{
		"dataSource": "mongodb-atlas",
		"database":   db,
		"collection": coll,
		"filter":     "",
	}

	bodyJson, _ := json.Marshal(body)

	req, err := http.NewRequest("POST", "https://eu-central-1.aws.data.mongodb-api.com/app/"+appId+"/endpoint/data/v1/action/findOne", bytes.NewBuffer(bodyJson))

	if err != nil {
		return 500
	}

	req.Header.Add("Accept", "application/json")
	req.Header.Add("email", email)
	req.Header.Add("password", pass)

	resp, err := httpClient.Do(req)

	if err != nil {
		return 500

	} else {
		return resp.StatusCode
	}

}
