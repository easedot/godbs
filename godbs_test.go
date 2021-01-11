package godbs

import (
	"database/sql"
	"fmt"
	"log"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCURD(t *testing.T) {
	type Author struct {
		ID        int64
		Name      string
		CreatedAt string
		UpdatedAt string
	}
	type Article struct {
		ID        int64 `"id"`
		Title     string
		Content   string
		UpdatedAt sql.NullTime
		CreatedAt sql.NullTime
		Author    Author `db:"-"`
	}
	//dsn := "user:password@tcp(0.0.0.0:3306)/article?loc=Asia%2FShanghai&parseTime=1"
	dsn := "user:password@tcp(0.0.0.0:3306)/article?parseTime=true"
	dbConn, err := sql.Open(`mysql`, dsn)
	if err != nil {
		log.Println(err)
	}
	defer dbConn.Close()

	db := NewHelper(dbConn, true)

	t.Run("SqlSlice", func(t *testing.T) {
		query:="select * from article where id=1"
		articles,err:=db.SqlSlice(query)
		if err!=nil{
			t.Error(err)
		}
		assert.Len(t,articles,1)
		assert.Equal(t,"1",articles[0][0])
	})


	t.Run("SqlMap", func(t *testing.T) {
		//query:="select * from article where title like '%jhh%' order by id limit 2"
		query:="select * from article where id=2"
		articles,err:=db.SqlMap(query)
		if err!=nil{
			t.Log(err)
		}
		assert.Len(t,articles,1)
		assert.Equal(t,"2",articles[0]["id"])
	})

	t.Run("SqlStructMap", func(t *testing.T) {
		r := map[interface{}]*Article{}
		query:="id in (1,2,3) order by id limit 2"
		if err:=db.SqlStructMap(query,&r);err!=nil{
			t.Log(err)
		}
		for k, article := range r {
			assert.Equal(t,k,article.ID)
			//log.Printf("%+v\n", article)
		}
	})

	t.Run("SqlStructSlice", func(t *testing.T) {
		var r []Article
		query:="id in (1,2) order by id limit 2"
		if err:=db.SqlStructSlice(query,&r);err!=nil{
			t.Log(err)
		}
		assert.Equal(t, len(r),2)
		assert.Equal(t,int64(1),r[0].ID)
		//for _, article := range r {
		//	log.Printf("%+v\n", article)
		//}
	})

	t.Run("Query", func(t *testing.T) {
		var r []*Article
		//q := Article{Title: "jhh2", Content: "jhh test 2"}
		q := Article{ID: 1}
		err := db.Query(&q, &r)
		if err != nil {
			log.Println(err)
		}
		for _, article := range r {
			assert.Equal(t,int64(1), article.ID)
			//log.Printf("%+v\n", article)
		}

	})

	t.Run("Find", func(t *testing.T) {
		a := Article{ID: 1}
		if err := db.Find(&a); err != nil {
			t.Log(err)
		}
		assert.Equal(t, int64(1),a.ID)
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
		assert.Equal(t, createTitle,cf.Title)

		updateTitle := "test update"
		u := Article{ID: c.ID, Title: updateTitle}
		if err := db.Update(u); err != nil {
			t.Log(err)
		}
		uf := Article{ID: u.ID}
		if err := db.Find(&uf); err != nil {
			t.Log(err)
		}
		assert.Equal(t, updateTitle,uf.Title)

		d := Article{ID: c.ID}
		if err := db.Delete(d); err != nil {
			t.Log(err)
		}
		//?check delete
	})

	t.Run("TransRollback",func(t *testing.T){
		transTitle := "test update2"
		var transID int64 = 1
		of := Article{ID: transID}
		if err := db.Find(&of); err != nil {
			t.Log(err)
		}

		u := Article{ID: transID, Title: transTitle}
		err = db.WithTrans(
			func(tx *DbHelper) error {
				if err := tx.Update(u); err != nil {
					t.Log(err)
				}
				return fmt.Errorf("rollback test")
			},
		)
		//check rollback update,title rollback old
		rf := Article{ID: transID}
		if err := db.Find(&rf); err != nil {
			t.Log(err)
		}
		assert.Equal(t, of.Title,rf.Title, fmt.Sprintf("rollback update to old:%s", rf.Title))

	})

	t.Run("TransCommit", func(t *testing.T) {
		transTitle := "test update2"
		var transID int64 = 1
		of := Article{ID: transID}
		if err := db.Find(&of); err != nil {
			t.Log(err)
		}

		u := Article{ID: transID, Title: transTitle}
		err = db.WithTrans(
			func(tx *DbHelper) error {
				err:= tx.Update(u)
				return err
			},
		)
		if err != nil {
			t.Log(err)
		}
		uf := Article{ID: transID}
		if err := db.Find(&uf); err != nil {
			t.Log(err)
		}
		//commit update check
		assert.Equal(t, transTitle,uf.Title)

		if err := db.Update(of); err != nil {
			t.Log(err)
		}
		lf := Article{ID: transID}
		if err := db.Find(&lf); err != nil {
			t.Log(err)
		}
		//commit update check
		assert.Equal(t, of.Title,lf.Title)

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
			for k:= range convertData {
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
