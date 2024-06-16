package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
)

type Person struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Age  int    `json:"age"`
}

type Phone struct {
	ID       int    `json:"id"`
	Number   string `json:"number"`
	PersonID int    `json:"person_id"`
}

type Address struct {
	ID      int    `json:"id"`
	City    string `json:"city"`
	State   string `json:"state"`
	Street1 string `json:"street1"`
	Street2 string `json:"street2"`
	ZipCode string `json:"zip_code"`
}

type AddressJoin struct {
	ID        int `json:"id"`
	PersonID  int `json:"person_id"`
	AddressID int `json:"address_id"`
}

type CombinedInfo struct {
	Name        string `json:"name"`
	PhoneNumber string `json:"phone_number"`
	City        string `json:"city"`
	State       string `json:"state"`
	Street1     string `json:"street1"`
	Street2     string `json:"street2"`
	ZipCode     string `json:"zip_code"`
}

type PersonCreateRequest struct {
	Name        string `json:"name"`
	PhoneNumber string `json:"phone_number"`
	City        string `json:"city"`
	State       string `json:"state"`
	Street1     string `json:"street1"`
	Street2     string `json:"street2"`
	ZipCode     string `json:"zip_code"`
}

var db *sql.DB // this connection is global

func main() {

	fmt.Println("ROUTER STARTED")

	defer db.Close()
	//making mera db connection here
	makeDbConnection()

	router := gin.Default()

	router.GET("/person/:person_id/info", GetpersonDetails) // future main in routes ko ek group main rakh sakte h
	router.POST("/person/create", CreatePerson)

	if err := router.Run(":8080"); err != nil {
		fmt.Println("Failed to start server: %v", err)
	}

}

func makeDbConnection() {
	var err error
	db, err = sql.Open("mysql", "root:anuj@tcp(localhost:3306)/cetec") // for future replace them with the env variables
	if err != nil {
		fmt.Println("ERROR _IN MAKINF_DB CONNECTION", err)
	}
	fmt.Println("DB CONNECTION SUCCESSFULLY DONE")
}

func GetpersonDetails(c *gin.Context) {
	personIDStr := c.Param("person_id")
	personID, err := strconv.Atoi(personIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid person_id"})
		return
	}

	var combinedInfo CombinedInfo

	// Execute a single query to fetch person, phone, and address details
	err = db.QueryRow(`
        SELECT 
            p.name AS Name,
            ph.number AS PhoneNumber,
            a.city AS City,
            a.state AS State,
            a.street1 AS Street1,
            a.street2 AS Street2,
            a.zip_code AS ZipCode
        FROM person p
        LEFT JOIN phone ph ON p.id = ph.person_id
        LEFT JOIN address_join as aj ON p.id = aj.person_id
        LEFT JOIN address as a ON aj.address_id = a.id
        WHERE p.id = ?
    `, personID).Scan(
		&combinedInfo.Name,
		&combinedInfo.PhoneNumber,
		&combinedInfo.City,
		&combinedInfo.State,
		&combinedInfo.Street1,
		&combinedInfo.Street2,
		&combinedInfo.ZipCode,
	)

	if err != nil {
		fmt.Println("Error fetching combined information:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Error fetching combined information"})
		return
	}

	// Return JSON response with combinedInfo
	c.JSON(http.StatusOK, combinedInfo)
}

func CreatePerson(c *gin.Context) {
	var req PersonCreateRequest

	if err := c.ShouldBindJSON(&req); err != nil { // meri request bodyh json ko yaha pe retieve karlo
		fmt.Println(err)
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid request body"})
		return
	}

	tx, err := db.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Error starting transaction"})
		return
	}
	defer func() {
		if err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Transaction rolled back"})
		}
	}()

	result, err := tx.Exec("INSERT INTO person (name) VALUES (?)", req.Name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Error inserting into person table"})
		return
	}
	personID, err := result.LastInsertId()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Error getting last insert ID"})
		return
	}

	_, err = tx.Exec("INSERT INTO phone (person_id, number) VALUES (?, ?)", personID, req.PhoneNumber)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Error inserting into phone table"})
		return
	}

	_, err = tx.Exec(`
        INSERT INTO address (city, state, street1, street2, zip_code)
        VALUES (?, ?, ?, ?, ?)
    `, req.City, req.State, req.Street1, req.Street2, req.ZipCode)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Error inserting into address table"})
		return
	}
	addressID, err := result.LastInsertId()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Error getting last insert ID"})
		return
	}

	_, err = tx.Exec("INSERT INTO address_join (person_id, address_id) VALUES (?, ?)", personID, addressID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Error inserting into address_join table"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Error committing transaction"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Person created successfully"})
}
