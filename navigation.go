package main

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"math"
	"strconv"
	"strings"

	"image/jpeg"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/validation"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/nfnt/resize"
	"golang.org/x/exp/slices"
)

func initializeNavigation() {

	ingredients = getDistinctFieldValues("mainingredient")
	categories = getDistinctFieldValues("category")
	countries = getDistinctFieldValues("country")

	navTree = createNavigationTree(categories, ingredients, countries)

	newRecipeButton = widget.NewButton("Add new recipe        ", func() { recipeEntry(Recipe{}, "new") })
	newRecipeButton.SetIcon(theme.ContentAddIcon())

	searchBar = widget.NewEntry()
	searchBar.SetPlaceHolder("Search for recipe...")

	searchBar.OnSubmitted = func(searchTerm string) {

		if len(strings.TrimSpace(searchTerm)) == 0 {
			return
		}

		searchBar.SetText("")

		currentQuery["type"] = "text"
		currentQuery["searchTerm"] = searchTerm

		currentRecipes, currentCount = getRecipesByText(currentQuery["searchTerm"], 0, resultsPerPage)

		allPages := int(math.Ceil(float64(currentCount) / float64(resultsPerPage)))

		currentPage = 1
		displayResults(allPages, searchTerm)

	}

}

func createNavigationTree(categ []string, ingr []string, countr []string) *widget.Tree {

	tree := widget.NewTree(
		func(id widget.TreeNodeID) []widget.TreeNodeID {
			switch id {
			case "":
				return []widget.TreeNodeID{"All recipes", "By category", "By main ingredient", "By country"}
			case "By category":
				return categ
			case "By main ingredient":
				return ingr
			case "By country":
				return countr
			}
			return []string{}
		},
		func(id widget.TreeNodeID) bool {
			return id == "" || id == "By category" || id == "By main ingredient" || id == "By country"
		},
		func(branch bool) fyne.CanvasObject {
			return widget.NewLabel("Node")
		},
		func(id widget.TreeNodeID, branch bool, o fyne.CanvasObject) {
			o.(*widget.Label).SetText(id)
		},
	)

	tree.OpenBranch("By category")

	tree.OnSelected = func(id string) {

		if slices.Contains(categ, id) {
			currentQuery["fieldName"] = "category"
			currentQuery["fieldValue"] = id

		} else if slices.Contains(ingr, id) {
			currentQuery["fieldName"] = "mainingredient"
			currentQuery["fieldValue"] = id

		} else if slices.Contains(countr, id) {
			currentQuery["fieldName"] = "country"
			currentQuery["fieldValue"] = id

		} else if id == "All recipes" {
			currentQuery["fieldName"] = ""
			currentQuery["fieldValue"] = ""

		} else {
			// no query if you click on main tree elements
			return
		}

		currentQuery["type"] = "query"

		currentRecipes, currentCount = getRecipes(currentQuery["fieldName"], currentQuery["fieldValue"], 0, resultsPerPage)

		allPages := int(math.Ceil(float64(currentCount) / float64(resultsPerPage)))
		currentPage = 1
		displayResults(allPages, id)
	}

	return tree
}

// displayResults creates a page with current query results
func displayResults(allPages int, searchTerm string) {

	resultsLabel := widget.NewLabel("Results for: " + searchTerm)
	searchContainer := container.NewVBox(searchBar, widget.NewSeparator(), resultsLabel)

	// Placeholder image for recipes that don't have one
	imagePlaceholder := canvas.NewImageFromResource(resourcePlaceholderJpg)
	imagePlaceholder.FillMode = canvas.ImageFillContain
	imagePlaceholder.SetMinSize(fyne.NewSize(50, 50))

	// Result list
	recipeList := widget.NewList(
		func() int {
			return len(currentRecipes)
		},
		func() fyne.CanvasObject {
			return container.NewHBox(imagePlaceholder, widget.NewLabel("placeholder"))
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			o.(*fyne.Container).RemoveAll()

			if len(currentRecipes[i].Image) == 0 {
				o.(*fyne.Container).Add(imagePlaceholder)

			} else {
				recipeImage := canvas.NewImageFromResource(fyne.NewStaticResource("img", currentRecipes[i].Image))
				recipeImage.SetMinSize(fyne.NewSize(50, 50))
				o.(*fyne.Container).Add(recipeImage)

			}

			o.(*fyne.Container).Add(container.NewVBox(layout.NewSpacer(), widget.NewLabel(currentRecipes[i].Title), layout.NewSpacer()))

		})

	recipeList.OnSelected = func(id widget.ListItemID) { displayRecipeDetails(id, allPages, searchTerm) }

	// Pagination button placeholder that determines button width
	pagButtonPlaceholder := fmt.Sprint(allPages) + " "

	// Displays page numbers
	paginationTable := widget.NewTable(
		func() (int, int) {
			return 1, allPages
		},
		func() fyne.CanvasObject {
			return widget.NewButton(pagButtonPlaceholder, func() {})
		},
		func(i widget.TableCellID, o fyne.CanvasObject) {

			o.(*widget.Button).SetText(fmt.Sprint(i.Col + 1))

			// Switch to another page
			o.(*widget.Button).OnTapped = func() {

				// Check if this was a normal query or text search
				if currentQuery["type"] == "query" {
					currentRecipes, currentCount = getRecipes(currentQuery["fieldName"], currentQuery["fieldValue"], resultsPerPage*(i.Col), resultsPerPage)

				} else {
					currentRecipes, currentCount = getRecipesByText(currentQuery["searchTerm"], resultsPerPage*(i.Col), resultsPerPage)
				}

				currentPage = i.Col + 1
				displayResults(allPages, searchTerm)

			}

			if i.Col+1 == currentPage {
				o.(*widget.Button).Disable()
			}

		})

	// Mobile layout has recipe list across whole screen
	if isMobile {
		backButton := &widget.Button{Icon: theme.NavigateBackIcon(), OnTapped: func() { mainWindow.SetContent(container.NewBorder(searchBar, newRecipeButton, nil, nil, navTree)) }}

		paginationContainer := container.NewBorder(nil, nil, nil, backButton, paginationTable)
		contentContainer := container.NewBorder(searchContainer, paginationContainer, nil, nil, recipeList)
		mainWindow.SetContent(contentContainer)

	} else {
		contentContainer := container.NewBorder(searchContainer, paginationTable, nil, nil, recipeList)
		mainWindow.SetContent(container.NewBorder(nil, nil, container.NewBorder(nil, newRecipeButton, nil, nil, navTree), nil, contentContainer))
	}

}

func displayRecipeDetails(id widget.ListItemID, allPages int, searchTerm string) {
	chosenRecipe := currentRecipes[id]

	editRecipeButton := widget.NewButtonWithIcon("Edit recipe", theme.DocumentCreateIcon(), func() { recipeEntry(currentRecipes[id], "edit") })
	backButton := widget.NewButtonWithIcon("Back", theme.NavigateBackIcon(), func() { displayResults(allPages, searchTerm) })

	descriptionLabel := widget.NewLabel(chosenRecipe.Description)
	descriptionLabel.Wrapping = fyne.TextWrapWord

	// Prepare ingredient list
	ingredientTable := container.NewVBox()
	for j, ingr := range chosenRecipe.Ingredients {
		ingrText := ingr.Name
		if ingr.Quantity != 0 {

			// Check if integer
			if math.Round(ingr.Quantity) == ingr.Quantity {
				ingrText += " " + fmt.Sprint(int(ingr.Quantity))

			} else {
				ingrText += " " + strconv.FormatFloat(ingr.Quantity, 'f', 2, 64)
			}

		}
		if len(ingr.Unit) != 0 {
			ingrText += " " + ingr.Unit
		}

		if len(ingr.Notes) != 0 && ingr.Notes != "/" {
			ingrText += " (" + ingr.Notes + ")"
		}

		ingrLabel := widget.NewLabel(fmt.Sprint(j+1) + ". " + ingrText)
		ingrLabel.Wrapping = fyne.TextWrapWord
		ingredientTable.Add(ingrLabel)
	}

	// Displays recipe image if available
	imageContainer := container.NewMax()
	if len(chosenRecipe.Image) != 0 {
		imageRes := fyne.NewStaticResource(chosenRecipe.Id, chosenRecipe.Image)
		canvasImage := canvas.NewImageFromResource(imageRes)
		canvasImage.FillMode = canvas.ImageFillContain
		canvasImage.SetMinSize(fyne.NewSize(200, 200))
		imageContainer.Add(canvasImage)
	}

	// Page layout
	titleLabel := canvas.NewText(chosenRecipe.Title, color.White)
	titleLabel.TextSize = 20

	ingredientsTitle := canvas.NewText("Ingredients:", color.White)
	ingredientsTitle.TextSize = 16

	preparationTitle := canvas.NewText("Preparation:", color.White)
	preparationTitle.TextSize = 16

	detailsContainer := container.NewVBox(
		container.New(layout.NewCenterLayout(), titleLabel),
		widget.NewLabel(""),
		imageContainer,
		container.New(layout.NewCenterLayout(), widget.NewLabel("Category: "+chosenRecipe.Category)),
		container.NewHBox(layout.NewSpacer(), widget.NewLabel(fmt.Sprint(chosenRecipe.PrepTime)+" min"), widget.NewLabel("|"), widget.NewLabel(fmt.Sprint(chosenRecipe.DefaultPortions)+" portions"), layout.NewSpacer()),
		ingredientsTitle,
		ingredientTable,
		preparationTitle,
		descriptionLabel,
		container.NewHBox(layout.NewSpacer(), backButton, editRecipeButton, layout.NewSpacer()),
	)

	if isMobile {
		mainWindow.SetContent(container.NewVScroll(detailsContainer))

	} else {
		mainWindow.SetContent(container.NewBorder(nil, nil, container.NewBorder(nil, newRecipeButton, nil, nil, navTree), nil, container.NewVScroll(detailsContainer)))
	}

}

// recipeEntry displays a page for adding a new recipe or editiing an existing one
func recipeEntry(recipe Recipe, mode string) {

	recipeImage := []byte{}

	allPages := int(math.Ceil(float64(currentCount) / float64(resultsPerPage)))
	backButton := widget.NewButtonWithIcon("Back", theme.NavigateBackIcon(), func() { displayResults(allPages, currentQuery["fieldValue"]) })

	// Entry fields
	titleEntry := &widget.Entry{PlaceHolder: "Recipe title"}
	prepEntry := &widget.Entry{PlaceHolder: "Preparation time [min]"}
	portionEntry := &widget.Entry{PlaceHolder: "Number of portions"}

	descriptionEntry := widget.NewMultiLineEntry()
	descriptionEntry.SetPlaceHolder("Description")
	descriptionEntry.SetMinRowsVisible(5)

	mainIngredientSelect := widget.NewSelectEntry(ingredients)
	mainIngredientSelect.SetPlaceHolder("Main ingredient")

	categorySelect := widget.NewSelectEntry(categories)
	categorySelect.SetPlaceHolder("Category")

	countrySelect := widget.NewSelectEntry(countries)
	countrySelect.SetPlaceHolder("Country")

	// Validators
	titleEntry.Validator = validation.NewRegexp(`.+`, "Field is required.")
	descriptionEntry.Validator = validation.NewRegexp(`.+`, "Field is required.")
	prepEntry.Validator = validation.NewRegexp(`^[0-9]*[1-9][0-9]*$`, "Value has to be a number.")
	portionEntry.Validator = validation.NewRegexp(`^[0-9]*[1-9][0-9]*$`, "Value has to be a number.")
	mainIngredientSelect.Validator = validation.NewRegexp(`.+`, "Field is required.")
	categorySelect.Validator = validation.NewRegexp(`.+`, "Field is required.")
	countrySelect.Validator = validation.NewRegexp(`.+`, "Field is required.")

	// Create one ingredient row
	name, qty, unit, note := createIngredientRow(1)

	// Store values of ingredient fields for later access
	ingredientData := [][]*widget.Entry{}
	ingredientData = append(ingredientData, []*widget.Entry{name, qty, unit, note})

	ingredientEntry := container.NewHBox(name, qty, unit, note)
	addIngrButton := &widget.Button{Icon: theme.ContentAddIcon()}

	ingrContainer := container.NewVBox(container.NewBorder(nil, nil, nil, addIngrButton, ingredientEntry))

	var recipeEntryContainer *fyne.Container

	submitButton := &widget.Button{Text: "Submit", Icon: theme.ConfirmIcon(), OnTapped: func() {

		prepTime, _ := strconv.Atoi(prepEntry.Text)
		DefaultPortions, _ := strconv.Atoi(portionEntry.Text)

		// Create Ingredient objects
		var ingredients []Ingredient
		for _, singleIngredient := range ingredientData {

			ingrQtyErr := singleIngredient[1].Validate()

			if len(strings.TrimSpace(singleIngredient[0].Text)) == 0 || ingrQtyErr != nil {
				continue
			}

			newIngrQty, _ := strconv.ParseFloat(singleIngredient[1].Text, 64)

			newIngredient := Ingredient{
				Name:     singleIngredient[0].Text,
				Quantity: newIngrQty,
				Unit:     singleIngredient[2].Text,
				Notes:    singleIngredient[3].Text,
			}

			ingredients = append(ingredients, newIngredient)

		}

		// Create Recipe object
		newDocument := Recipe{
			Title:           titleEntry.Text,
			Description:     descriptionEntry.Text,
			Category:        categorySelect.Text,
			Country:         countrySelect.Text,
			MainIngredient:  mainIngredientSelect.Text,
			PrepTime:        prepTime,
			DefaultPortions: DefaultPortions,
			Ingredients:     ingredients,
			Image:           recipeImage,
		}

		var addUpdateOperation bool

		if mode == "new" {
			addUpdateOperation = newDocument.addNewRecipe()

		} else {
			addUpdateOperation = newDocument.updateRecipe(recipe.Id)
		}

		if addUpdateOperation == true {
			allPages := int(math.Ceil(float64(currentCount) / float64(resultsPerPage)))
			displayResults(allPages, currentQuery["fieldValue"])
		}

	}}

	submitButton.Disable()

	// Adds ingredient row
	addIngrButton.OnTapped = func() {

		name, qty, unit, note = createIngredientRow(len(ingrContainer.Objects) + 1)
		ingredientData = append(ingredientData, []*widget.Entry{name, qty, unit, note})
		extraIngredientEntry := container.NewHBox(name, qty, unit, note)

		ingrContainer.Add(container.NewBorder(nil, nil, nil, nil, extraIngredientEntry))
		ingrContainer.Refresh()
		recipeEntryContainer.Refresh()

	}

	addImageButton := &widget.Button{Text: "Add image", OnTapped: func() {}, Icon: theme.MediaPhotoIcon()}
	addImageContainer := container.NewGridWithColumns(2, container.NewHBox(container.NewVBox(layout.NewSpacer(), addImageButton, layout.NewSpacer()), layout.NewSpacer()))

	// All entries for easier validation
	entryElements := []*widget.Entry{titleEntry, descriptionEntry, prepEntry, portionEntry, &categorySelect.Entry, &mainIngredientSelect.Entry}

	for _, elem := range entryElements {

		elem.OnChanged = func(s string) {

			formValid := true
			for _, entry := range entryElements {
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

	// Add image functionality
	addImage := func(f fyne.URIReadCloser, err error) {

		// In case file dialog is cancelled or file cannot be accessed
		if err != nil || f == nil {
			return
		}

		chosenImage, _, err := image.Decode(f)

		if err != nil {
			errorDialog := dialog.NewError(err, mainWindow)
			errorDialog.Show()

		} else {

			bounds := chosenImage.Bounds()
			width := bounds.Dx()
			height := bounds.Dy()

			// Check if image is too large and resize if necessary
			if width > 300 || height > 300 {

				if width > height {
					chosenImage = resize.Resize(300, 0, chosenImage, resize.Lanczos3)

				} else {
					chosenImage = resize.Resize(0, 300, chosenImage, resize.Lanczos3)
				}
			}

			// Encode as jpeg and save in recipeImage
			imageBuffer := new(bytes.Buffer)
			jpeg.Encode(imageBuffer, chosenImage, nil)
			recipeImage = imageBuffer.Bytes()

			imageRes := fyne.NewStaticResource(f.URI().Name(), recipeImage)
			canvasImage := canvas.NewImageFromResource(imageRes)
			canvasImage.FillMode = canvas.ImageFillContain
			canvasImage.SetMinSize(fyne.NewSize(100, 100))

			// Modify container to include the new image
			addImageContainer.RemoveAll()
			addImageContainer.Add(container.NewHBox(container.NewVBox(layout.NewSpacer(), addImageButton, layout.NewSpacer()), layout.NewSpacer()))
			addImageContainer.Add(canvasImage)

			addImageButton.SetText("Change image")

		}

	}

	addImageButton.OnTapped = func() {
		fileDialog := dialog.NewFileOpen(addImage, mainWindow)
		fileDialog.Show()
	}

	// Prefill fields for edit mode
	if mode == "edit" {

		titleEntry.Text = recipe.Title
		descriptionEntry.Text = recipe.Description
		prepEntry.Text = fmt.Sprint(recipe.PrepTime)
		portionEntry.Text = fmt.Sprint(recipe.DefaultPortions)
		categorySelect.Text = recipe.Category
		mainIngredientSelect.Text = recipe.MainIngredient
		countrySelect.Text = recipe.Country

		ingredientData = [][]*widget.Entry{}
		ingrContainer = container.NewVBox()

		for j, ingr := range recipe.Ingredients {

			name, qty, unit, note = createIngredientRow(len(ingrContainer.Objects) + 1)
			name.Text, qty.Text, unit.Text, note.Text = ingr.Name, fmt.Sprint(ingr.Quantity), ingr.Unit, ingr.Notes
			ingredientData = append(ingredientData, []*widget.Entry{name, qty, unit, note})
			extraIngredientEntry := container.NewHBox(name, qty, unit, note)

			if j == 0 {
				ingrContainer.Add(container.NewBorder(nil, nil, nil, addIngrButton, extraIngredientEntry))

			} else {
				ingrContainer.Add(container.NewBorder(nil, nil, nil, nil, extraIngredientEntry))
			}

		}

		ingrContainer.Refresh()

	}

	// Display current image for edit mode if image exists
	if mode == "edit" {
		if len(recipe.Image) != 0 {
			addImageButton.SetText("Change image")

			imageRes := fyne.NewStaticResource(recipe.Id, recipe.Image)
			canvasImage := canvas.NewImageFromResource(imageRes)
			canvasImage.FillMode = canvas.ImageFillContain
			canvasImage.SetMinSize(fyne.NewSize(100, 100))
			addImageContainer.Add(canvasImage)

		}
	}

	// Page layout
	recipeEntryContainer = container.NewVBox(
		titleEntry,
		descriptionEntry,
		prepEntry,
		portionEntry,
		categorySelect,
		mainIngredientSelect,
		countrySelect,
		ingrContainer,
		addImageContainer,
		container.NewHBox(layout.NewSpacer(), backButton, submitButton, layout.NewSpacer()),
	)

	if isMobile {
		mainWindow.SetContent(recipeEntryContainer)

	} else {
		mainWindow.SetContent(container.NewBorder(nil, nil, container.NewBorder(nil, newRecipeButton, nil, nil, navTree), nil, recipeEntryContainer))
	}

}

func createIngredientRow(i int) (*widget.Entry, *widget.Entry, *widget.Entry, *widget.Entry) {

	w1 := &widget.Entry{PlaceHolder: "Ingredient " + fmt.Sprint(i)}
	w2 := &widget.Entry{PlaceHolder: "Quantity"}
	w3 := &widget.Entry{PlaceHolder: "Unit"}
	w4 := &widget.Entry{PlaceHolder: "Note"}

	// Decimal number validation
	w2.Validator = validation.NewRegexp(`^\d*(\.)?(\d{0,3})?$`, "Value has to be a number.")

	return w1, w2, w3, w4
}
