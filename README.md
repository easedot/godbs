# ****godbs
# sample go database helper

```go
package main

import (
	"database/sql"
	"log"
	"time"

	"github.com/easedot/godbs"
)

type Article struct {
	ID        int64 `pk:"id"`
	Title     string
	Content   string
	UpdatedAt time.Time
	CreatedAt time.Time
}
func main(){
	dsn := "user:password@tcp(0.0.0.0:3306)/article"
	dbConn, err := sql.Open(`mysql`, dsn)
	if err != nil  {
		log.Println(err)
	}
	defer dbConn.Close()

	db := godbs.NewHelper(dbConn, nil, false)

	var articles []Article
	q := Article{Title: "jhh2", Content: "jhh test 2"}
	err = db.Query(&q, &articles)
	if err != nil {
		log.Println(err)
	}
	for _, article := range articles {
		log.Printf("%+v\n", article)
	}

}

```