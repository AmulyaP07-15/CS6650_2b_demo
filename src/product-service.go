// every Go file starts by saying which package it belongs to
// main means this file builds into a program you can run
package main

// import pulls in the libraries this file uses
import (
	// net/http has the standard HTTP status codes like 500
	"net/http"
	// strconv converts text to numbers, like "42" into 42
	"strconv"
	// sync has the lock we use to protect the map
	"sync"

	// gin is the web framework that handles requests and responses
	"github.com/gin-gonic/gin"
)

// Product is the shape of one product, matching the Product schema in api.yaml
type Product struct {
	// product id, the json tag is the name used in the request and response body
	ProductID int `json:"product_id"`
	// stock keeping unit, a text code for the product
	SKU string `json:"sku"`
	// name of the company that makes it
	Manufacturer string `json:"manufacturer"`
	// which category the product is in
	CategoryID int `json:"category_id"`
	// weight in grams, a pointer so a missing weight is nil instead of 0
	Weight *int `json:"weight"`
	// another id the spec asks for
	SomeOtherID int `json:"some_other_id"`
}

// var block declares variables shared by the whole file
var (
	// products stores every product in memory, the key is the product id
	products = map[int]Product{}
	// mu is a lock, many readers can hold it together but a writer holds it alone
	mu sync.RWMutex
)

// errJSON sends an error response shaped like the Error schema in api.yaml
func errJSON(c *gin.Context, code int, errCode, msg, details string) {
	// build the body with the two required fields
	body := gin.H{"error": errCode, "message": msg}
	// details is optional so only add it when we have something
	if details != "" {
		// put the extra info into the body
		body["details"] = details
	}
	// send the status code and the body as json
	c.JSON(code, body)
}

// seed fills the store with products 1 to 100 when the server starts
func seed() {
	// loop i from 1 up to 100
	for i := 1; i <= 100; i++ {
		// a weight of 0 grams, stored in a variable so we can point to it
		w := 0
		// save a sample product under id i
		products[i] = Product{
			// the id is the loop number
			ProductID: i,
			// sku becomes SKU1, SKU2 and so on
			SKU: "SKU" + strconv.Itoa(i),
			// placeholder manufacturer name
			Manufacturer: "Seed",
			// placeholder category
			CategoryID: 1,
			// point to the weight variable above
			Weight: &w,
			// placeholder value
			SomeOtherID: 1,
		}
	}
}

// validate checks each field against the rules in api.yaml
// it returns an empty string if everything is fine, otherwise the problem
func validate(p Product) string {
	// switch runs the first case that is true
	switch {
	// product id must be 1 or more
	case p.ProductID < 1:
		return "product_id must be at least 1"
	// sku must have between 1 and 100 characters
	case p.SKU == "" || len(p.SKU) > 100:
		return "sku must be between 1 and 100 chars"
	// manufacturer must have between 1 and 200 characters
	case p.Manufacturer == "" || len(p.Manufacturer) > 200:
		return "manufacturer must be between 1 and 200 chars"
	// category id must be 1 or more
	case p.CategoryID < 1:
		return "category_id must be at least 1"
	// nil means the client didn't send a weight at all
	case p.Weight == nil:
		return "weight is required"
	// the star reads the number the pointer points to, it can't be below 0
	case *p.Weight < 0:
		return "weight can't be negative"
	// some other id must be 1 or more
	case p.SomeOtherID < 1:
		return "some_other_id must be at least 1"
	}
	// nothing failed so the product is valid
	return ""
}

// getProduct runs for GET /products/:productId
// the spec only allows 200, 404 and 500 for this endpoint
func getProduct(c *gin.Context) {
	// read productId from the url and turn the text into a number
	id, err := strconv.Atoi(c.Param("productId"))
	// if it isn't a number or is below 1, GET has no 400 so we say not found
	if err != nil || id < 1 {
		// send a 404 with the reason in details
		errJSON(c, 404, "NOT_FOUND", "product not found", "productId must be a positive integer")
		// stop here
		return
	}

	// take a read lock so writers wait but other readers can still read
	mu.RLock()
	// look up the product, found is false if the id isn't in the map
	p, found := products[id]
	// release the read lock
	mu.RUnlock()

	// the id is valid but there is no product with it
	if !found {
		// send a 404
		errJSON(c, 404, "NOT_FOUND", "product not found", "")
		// stop here
		return
	}
	// send 200 with the product as json
	c.JSON(200, p)
}

// addDetails runs for POST /products/:productId/details
// returns 204 when saved, 400 for bad input, 404 if the product doesn't exist
func addDetails(c *gin.Context) {
	// read productId from the url and turn it into a number
	id, err := strconv.Atoi(c.Param("productId"))
	// POST does allow 400, so a bad id is invalid input
	if err != nil || id < 1 {
		// send a 400 with the reason
		errJSON(c, 400, "INVALID_INPUT", "invalid product id", "productId must be a positive integer")
		// stop here
		return
	}

	// an empty Product to fill from the request body
	var p Product
	// read the json body into p, err is set if the json is broken
	if err := c.ShouldBindJSON(&p); err != nil {
		// send a 400 with gin's error text in details
		errJSON(c, 400, "INVALID_INPUT", "bad json body", err.Error())
		// stop here
		return
	}

	// the id in the url and the id in the body must be the same product
	if p.ProductID != id {
		// send a 400 because they disagree
		errJSON(c, 400, "INVALID_INPUT", "product_id doesn't match url", "")
		// stop here
		return
	}

	// run all the field checks, msg is empty if they pass
	if msg := validate(p); msg != "" {
		// send a 400 with the failed rule in details
		errJSON(c, 400, "INVALID_INPUT", "invalid product data", msg)
		// stop here
		return
	}

	// take the write lock so nobody else reads or writes until we finish
	mu.Lock()
	// defer runs this when the function ends, so the lock is always released
	defer mu.Unlock()

	// check the product exists, the underscore ignores the product itself
	if _, exists := products[id]; !exists {
		// send a 404 because there is nothing to add details to
		errJSON(c, 404, "NOT_FOUND", "product not found", "")
		// stop here, defer still releases the lock
		return
	}
	// replace the stored product with the new details
	products[id] = p

	// send 204, which means success with no body
	c.Status(204)
}

// main is where the program starts
func main() {
	// load the 100 sample products
	seed()

	// create an empty router with no extras
	r := gin.New()
	// add the logger so every request prints one line in the terminal
	r.Use(gin.Logger())

	// add our own crash catcher so a panic becomes a 500 instead of killing the server
	r.Use(gin.CustomRecovery(func(c *gin.Context, _ any) {
		// stop the request and send a 500 shaped like the Error schema
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			// the error code
			"error": "INTERNAL_ERROR",
			// the readable message
			"message": "unexpected server error",
		})
	}))

	// send GET /products/anything to getProduct
	r.GET("/products/:productId", getProduct)
	// send POST /products/anything/details to addDetails
	r.POST("/products/:productId/details", addDetails)

	// test route that crashes on purpose so we can show a 500
	r.GET("/panic", func(c *gin.Context) {
		// panic crashes this request and our recovery turns it into a 500
		panic("forced error for testing")
	})

	// start the server on port 8080, which must match container_port in terraform
	r.Run(":8080")
}
