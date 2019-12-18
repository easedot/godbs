# godbs
## Sample go database helper.

1.step1 install
    
    ```go
    
    go get github.com/robteix/testmod    
    ```
2.step2 import
    
    ```go
    
    import (
        "database/sql"
        "log"
        "time"
        "github.com/easedot/godbs"
    )
    ```
3.step3 init database connection
    
    ```go
    
    type Article struct {
        ID        int64 
        Title     string
        Content   string
        UpdatedAt time.Time
        CreatedAt time.Time
    }
	dsn := "user:password@tcp(0.0.0.0:3306)/article"
	dbConn, err := sql.Open("mysql", dsn)
	if err != nil  {
		log.Println(err)
	}
	defer dbConn.Close()
	     
	db := godbs.NewHelper(dbConn, nil, false)
    
    ```    

4.step4 query create update delete

   4.1
   ```go
	var articles []Article
	q := Article{Title: "test_title", Content: "test_content"}
	err = db.Query(&q, &articles)
	if err != nil {
		log.Println(err)
	}
	for _, article := range articles {
		log.Printf("%+v\n", article)
	}

   ``` 

