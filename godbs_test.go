package godbs

import (
	"database/sql"
	"fmt"
	"log"
	"testing"
	"time"

	"github.com/magiconair/properties/assert"
	"github.com/spf13/viper"
)

func TestCURD(t *testing.T) {
	type Author struct {
		ID        int64
		Name      string
		CreatedAt string
		UpdatedAt string
	}
	type Article struct {
		ID        int64 `pk:"id"`
		Title     string
		Content   string
		UpdatedAt time.Time
		CreatedAt time.Time
		Author    Author `db:"-"`
	}
	dsn := "user:password@tcp(0.0.0.0:3306)/article?loc=Asia%2FJakarta&parseTime=1"
	dbConn, err := sql.Open(`mysql`, dsn)
	if err != nil && viper.GetBool("debug") {
		log.Println(err)
	}
	defer dbConn.Close()

	db := NewHelper(dbConn, nil, false)

	t.Run("SqlSlice", func(t *testing.T) {
		//query:="select * from article where title like '%jhh%' order by id limit 2"
		query:="select * from article where id=9"
		articles,err:=db.SqlSlice(query)
		if err!=nil{
			t.Log(err)
		}
		for i, article := range articles {
			if i==0{
				assert.Equal(t,article[i],"9")
			}
			log.Printf("%+v\n", article)
		}
	})


	t.Run("SqlMap", func(t *testing.T) {
		//query:="select * from article where title like '%jhh%' order by id limit 2"
		query:="select * from article where id=2"
		articles,err:=db.SqlMap(query)
		if err!=nil{
			t.Log(err)
		}
		for _, article := range articles {
			//assert.Matches(t,article['title'],"jhh")
			assert.Equal(t,article["id"],"2")
			//log.Printf("%+v\n", article)
		}
	})

	t.Run("SqlStructMap", func(t *testing.T) {
		articles:= map[interface{}]Article{}
		query:="title like '%jhh%' order by id limit 2"
		if err:=db.SqlStructMap(query,&articles);err!=nil{
			t.Log(err)
		}
		for k, article := range articles {
			assert.Matches(t,article.Title,"jhh")
			assert.Equal(t,article.ID,k)
			//log.Printf("%+v\n", article)
		}
	})

	t.Run("SqlStructSlice", func(t *testing.T) {
		var articles []Article
		query:="title like '%jhh%' order by id limit 2"
		if err:=db.SqlStructSlice(query,&articles);err!=nil{
			t.Log(err)
		}
		assert.Equal(t, len(articles),2)
		for _, article := range articles {
			assert.Matches(t,article.Title,"jhh")
			//log.Printf("%+v\n", article)
		}
	})

	t.Run("Query", func(t *testing.T) {
		var articles []Article
		q := Article{Title: "jhh2", Content: "jhh test 2"}
		//q := Article{ID: 9}
		err := db.Query(&q, &articles)
		if err != nil {
			log.Println(err)
		}
		for _, article := range articles {
			assert.Equal(t, article.Title, "jhh2")
			//log.Printf("%+v\n", article)
		}

	})

	t.Run("Find", func(t *testing.T) {
		a := Article{ID: 9}
		if err := db.Find(&a); err != nil {
			t.Log(err)
		}
		assert.Equal(t, a.Title, "jhh2")
	})

	t.Run("Create", func(t *testing.T) {
		createTitle := "test create"
		c := Article{Title: createTitle, Content: "jhh test 2"}
		if err := db.Create(&c); err != nil {
			t.Log(err)
		}

		cf := Article{ID: c.ID}
		if err := db.Find(&cf); err != nil {
			t.Log(err)
		}
		assert.Equal(t, cf.Title, createTitle)

		updateTitle := "test update"
		u := Article{ID: c.ID, Title: updateTitle}
		if err := db.Update(u); err != nil {
			t.Log(err)
		}
		uf := Article{ID: u.ID}
		if err := db.Find(&uf); err != nil {
			t.Log(err)
		}
		assert.Equal(t, uf.Title, updateTitle)

		d := Article{ID: c.ID}
		if err := db.Delete(d); err != nil {
			t.Log(err)
		}
		//?check delete
	})
	t.Run("Trans", func(t *testing.T) {
		transTitle := "test update2"
		var transId int64 = 7
		of := Article{ID: transId}
		if err := db.Find(&of); err != nil {
			t.Log(err)
		}

		u := Article{ID: transId, Title: transTitle}
		err = db.WithTrans(
			func(tx *DbHelper) error {
				if err := tx.Update(u); err != nil {
					t.Log(err)
				}
				return fmt.Errorf("rollback test")
			},
		)
		//check rollback update,title rollback old
		rf := Article{ID: transId}
		if err := db.Find(&rf); err != nil {
			t.Log(err)
		}
		assert.Equal(t, rf.Title, of.Title, fmt.Sprintf("rollback update to old:%s", rf.Title))

		err = db.WithTrans(
			func(tx *DbHelper) error {
				if err := tx.Update(u); err != nil {
					t.Log(err)
				}
				return nil
			},
		)
		if err != nil {
			t.Log(err)
		}
		uf := Article{ID: transId}
		if err := db.Find(&uf); err != nil {
			t.Log(err)
		}
		//commit update check
		assert.Equal(t, uf.Title, transTitle)

		if err := db.Update(of); err != nil {
			t.Log(err)
		}
		lf := Article{ID: transId}
		if err := db.Find(&lf); err != nil {
			t.Log(err)
		}
		//commit update check
		assert.Equal(t, lf.Title, of.Title)

	})
}

var convertData = map[string]string{
	"":                      "",
	"F":                     "f",
	"Foo":                   "foo",
	"FooB":                  "foo_b",
	"FooID":                 "foo_id",
	" FooBar\t":             "foo_bar",
	"HTTPStatusCode":        "http_status_code",
	"ParseURLDoParse":       "parse_url_do_parse",
	"Convert Space":         "convert_space",
	"Skip   MultipleSpaces": "skip_multiple_spaces",
}
var (
	toSnakes = []struct {
		name string
		fun  func(string) string
	}{
		{"toSnake", toSnake},
	}
)

func BenchmarkToSnake(b *testing.B) {
	for _, snake := range toSnakes {
		b.Run(snake.name, func(b *testing.B) {
			for k, _ := range convertData {
				for i := 0; i < b.N; i++ {
					snake.fun(k)
				}
			}
		})
	}
}
func TestToSnake(t *testing.T) {
	//setup test
	for _, snake := range toSnakes {
		t.Run(snake.name, func(t *testing.T) {
			for seed, want := range convertData {
				result := snake.fun(seed)
				assert.Equal(t, result, want)
			}
		})
	}
	//close test
}
