# Godbs #
## Sample go database helper.

1.step1 install

    go get github.com/easedot/godbs

2.step2 import

    import (
        "database/sql"
        "log"
        "time"
        "github.com/easedot/godbs"
    )
            
3.step3 init database connection
    
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
    db := godbs.NewHelper(dbConn, false)        

4.step4 sample

   4.1 query 

    var articles []Article
    q := Article{Title: "test_title", Content: "test_content"}
    err = db.Query(&q, &articles)
    if err != nil {
        ...
    }
    for _, article := range articles {
        log.Printf("%+v\n", article)
    }

   4.2 find

    var transId int64 = 7
    article := Article{ID: transId}
    if err := db.Find(&article); err != nil {
        ...
    }
    log.Printf("%+v\n", article)

   4.3 create

    createTitle := "test create"
    c := Article{Title: createTitle, Content: "jhh test 2"}
    if err := db.Create(&c); err != nil {
        ...
    }

   4.3 trans update

    err = db.WithTrans(
        func(tx *DbHelper) error {
            err:= tx.Update(u)
            return err
        },
    )
    