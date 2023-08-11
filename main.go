package main

import (
	"math"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/validation"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"golang.org/x/crypto/bcrypt"
)

var mainApp fyne.App
var mainWindow fyne.Window
var navTree *widget.Tree
var searchBar *widget.Entry
var credentials map[string]string
var isMobile bool

var ingredients []string
var categories []string
var countries []string
var newRecipeButton *widget.Button

// Current results
var currentRecipes []Recipe
var currentCount int
var currentQuery map[string]string
var currentPage int

type Config struct {
	maximumImageSizePx   uint
	resultsPerPage       int
	desktopDefaultWidth  float32
	desktopDefaultHeight float32
}

var config Config

func main() {

	// Load config
	setConfig()

	// General visual settings
	mainApp = app.NewWithID("MealTimeApp")
	mainApp.Settings().SetTheme(theme.DarkTheme())
	icon, _ := fyne.LoadResourceFromPath("resources/icon.png")
	mainApp.SetIcon(icon)

	mainWindow = mainApp.NewWindow("MealTime")
	mainWindow.Resize(fyne.NewSize(config.desktopDefaultWidth, config.desktopDefaultHeight))
	mainWindow.CenterOnScreen()

	// Determine device
	device := fyne.CurrentDevice()
	isMobile = device.IsMobile()

	// Check if config is available
	atlasAppId := mainApp.Preferences().StringWithFallback("atlasAppId", "")

	if atlasAppId == "" {
		displaySettingsPage("new")

	} else {
		displayLoginPage("Welcome to MealTime!")
	}

	mainWindow.Show()
	mainApp.Run()

}

// displayInitialPage creates the first page users will see when the log in
func displayInitialPage() {

	initializeNavigation()

	// mobile layout is different - no default display of all recipes
	if isMobile == true {
		mainWindow.SetContent(container.NewBorder(searchBar, newRecipeButton, nil, nil, navTree))
		currentQuery = map[string]string{}

	} else {
		currentQuery = map[string]string{"type": "query", "fieldName": "", "fieldValue": ""}

		currentRecipes, currentCount = getRecipes("", "", 0, config.resultsPerPage)

		allPages := int(math.Ceil(float64(currentCount) / float64(config.resultsPerPage)))

		currentPage = 1
		displayResults(allPages, "All recipes")
		mainWindow.Canvas().Focus(searchBar)

	}

}

func displaySettingsPage(mode string) {

	var welcomeLabel *widget.Label

	// mode can be "edit" or "new"
	if mode == "new" {
		welcomeLabel = widget.NewLabel("Welcome to MealTime! Please configure your app:")

	} else {
		welcomeLabel = widget.NewLabel("Change MongoDB settings:")
	}

	appIdEntry := &widget.Entry{PlaceHolder: "MongoDB App ID"}
	emailEntry := &widget.Entry{PlaceHolder: "E-mail"}
	passwordEntry := &widget.Entry{PlaceHolder: "Password", Password: true}
	dbEntry := &widget.Entry{PlaceHolder: "Database name"}
	collEntry := &widget.Entry{PlaceHolder: "Collection name"}

	// Prefill for edit mode
	if mode == "edit" {
		appIdEntry.SetText(mainApp.Preferences().String("atlasAppId"))
		emailEntry.SetText(mainApp.Preferences().String("atlasAppEmail"))
		dbEntry.SetText(mainApp.Preferences().String("atlasDbName"))
		collEntry.SetText(mainApp.Preferences().String("atlasCollName"))

	}

	// Field validators
	appIdEntry.Validator = validation.NewRegexp(`.+`, "Field is required.")
	emailEntry.Validator = validation.NewRegexp(`.+`, "Field is required.")
	passwordEntry.Validator = validation.NewRegexp(`.+`, "Field is required.")
	dbEntry.Validator = validation.NewRegexp(`.+`, "Field is required.")
	collEntry.Validator = validation.NewRegexp(`.+`, "Field is required.")

	formElements := []*widget.Entry{appIdEntry, emailEntry, passwordEntry, dbEntry, collEntry}

	submitButton := &widget.Button{Text: "Submit", OnTapped: func() {}, Icon: theme.LoginIcon()}
	submitButton.Disable()

	// Set all fields to run validation on change
	for _, elem := range formElements {

		elem.OnChanged = func(s string) {

			formValid := true
			for _, entry := range formElements {
				err := entry.Validate()
				if err != nil {
					formValid = false
				}
			}

			if formValid == true {
				submitButton.Enable()

			} else {
				submitButton.Disable()
			}
		}
	}

	submitButton.OnTapped = func() {

		loginStatusCode := validateMongoLogin(appIdEntry.Text, dbEntry.Text, collEntry.Text, emailEntry.Text, passwordEntry.Text)

		if loginStatusCode == 200 {

			// Store hashed password for quick password validation later
			hashed, _ := bcrypt.GenerateFromPassword([]byte(passwordEntry.Text), bcrypt.DefaultCost)
			mainApp.Preferences().SetString("atlasAppPassword", string(hashed))

			mainApp.Preferences().SetString("atlasAppId", appIdEntry.Text)
			mainApp.Preferences().SetString("atlasDbName", dbEntry.Text)
			mainApp.Preferences().SetString("atlasCollName", collEntry.Text)
			mainApp.Preferences().SetString("atlasAppEmail", emailEntry.Text)

			// Proceed to login page
			displayLoginPage("App succesfully configured, you can now log in.")

		} else {

			responses := map[int]string{
				401: "Login failed - wrong credentials!",
				404: "Login failed - App not found!",
				400: "Login failed - database/collection not found!",
			}

			message, exists := responses[loginStatusCode]

			if exists == false {
				message = "Login failed - unknown error!"
			}

			wrongCredentialsDialog := dialog.NewInformation("Error", message, mainWindow)
			wrongCredentialsDialog.Show()
		}

	}

	// Settings page layout
	var settingsPage *fyne.Container

	if isMobile {
		settingsPage = container.NewVBox(
			layout.NewSpacer(),
			welcomeLabel,
			appIdEntry,
			emailEntry,
			passwordEntry,
			dbEntry,
			collEntry,
			submitButton,
			layout.NewSpacer(),
		)

	} else {
		settingsPage = container.NewGridWithRows(3,
			layout.NewSpacer(),
			container.NewGridWithColumns(3,
				layout.NewSpacer(),
				container.NewVBox(welcomeLabel, appIdEntry, emailEntry, passwordEntry, dbEntry, collEntry, submitButton),
				layout.NewSpacer()),
			layout.NewSpacer())
	}

	mainWindow.SetContent(settingsPage)

}

func displayLoginPage(welcomeText string) {

	welcomeLabel := widget.NewLabel(welcomeText)
	passwordEntry := &widget.Entry{PlaceHolder: "Password", Password: true}
	changeSettButton := &widget.Button{Text: "Change settings", Icon: theme.SettingsIcon(), OnTapped: func() { displaySettingsPage("edit") }}

	loginButton := &widget.Button{Text: "Login", Icon: theme.LoginIcon()}
	loginButton.OnTapped = func() {

		storedHash := mainApp.Preferences().String("atlasAppPassword")
		err := bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(passwordEntry.Text))

		if err != nil {

			wrongPasswordDialog := dialog.NewInformation("Error", "Wrong password!", mainWindow)
			wrongPasswordDialog.Show()

		} else {

			// Load credentials
			credentials = map[string]string{
				"appId":      mainApp.Preferences().String("atlasAppId"),
				"database":   mainApp.Preferences().String("atlasDbName"),
				"collection": mainApp.Preferences().String("atlasCollName"),
				"email":      mainApp.Preferences().String("atlasAppEmail"),
				"password":   passwordEntry.Text,
			}

			displayInitialPage()
		}
	}

	// Login page layout
	var loginPage *fyne.Container

	if isMobile {
		loginPage = container.NewVBox(
			layout.NewSpacer(),
			container.NewCenter(welcomeLabel),
			passwordEntry,
			loginButton,
			changeSettButton,
			layout.NewSpacer(),
		)

	} else {
		loginPage = container.NewGridWithRows(3,
			layout.NewSpacer(),
			container.NewGridWithColumns(3,
				layout.NewSpacer(),
				container.NewVBox(container.NewCenter(welcomeLabel), passwordEntry, loginButton, changeSettButton),
				layout.NewSpacer()),
			layout.NewSpacer())
	}

	mainWindow.SetContent(loginPage)
	mainWindow.Canvas().Focus(passwordEntry)

}
