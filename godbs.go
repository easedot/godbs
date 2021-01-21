package godbs

import (
"database/sql"
"encoding/json"
"fmt"
"log"
"reflect"
"regexp"
	"strconv"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

const mySQLTimeFormat = "2006-01-02 15:04:05"

//For sql.DB and sql.Tx Replace
type Transaction interface {
	Exec(query string, args ...interface{}) (sql.Result, error)
	Prepare(query string) (*sql.Stmt, error)
	Query(query string, args ...interface{}) (*sql.Rows, error)
	QueryRow(query string, args ...interface{}) *sql.Row
}

type DbHelper struct {
	db    *sql.DB
	debug bool
	conn  Transaction
}

func NewHelper(db *sql.DB, debug bool) DbHelper {
	helper := DbHelper{}
	helper.db = db
	helper.conn = db
	helper.debug = debug
	return helper
}

type TransFunc func(tx *DbHelper) error

func (e *DbHelper) WithTrans(block TransFunc) error {
	tx, err := e.db.Begin()

	if err != nil {
		return err
	}
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		} else if err != nil {
			tx.Rollback()
		} else {
			tx.Commit()
		}
	}()
	db := NewHelper(e.db, e.debug)
	db.conn=tx
	err = block(&db)
	return err
}

func (e *DbHelper) Find(m interface{}) (err error) {
	table, id, idv, fields, _ := e.genInfo(m)
	q := fmt.Sprintf("SELECT %s FROM %s WHERE %s=%v ", strings.Join(fields, ","), table, id, idv)
	if e.debug {
		log.Println(q)
	}
	valuesPtr := e.genValues(m)
	//err = e.conn.QueryRow(q).Scan(valuesPtr...)

	err=e.conn.QueryRow(q).Scan(valuesPtr...)
	return
}

func (e *DbHelper) Query(m interface{}, outSlice interface{}) (err error) {
	valuePtr := reflect.ValueOf(outSlice)
	value := valuePtr.Elem()
	elemType := valuePtr.Type().Elem().Kind()
	if valuePtr.Kind()==reflect.Ptr{
		elemType = reflect.TypeOf(value.Interface()).Elem().Kind()
	}

	table, _, _, fields, values := e.genInfo(m)
	var query []string
	for k, v := range values {
		query = append(query, fmt.Sprintf("%s=%v", k, v))
	}
	q := fmt.Sprintf("SELECT %s FROM %s WHERE %s ",
		strings.Join(fields, ","), table, strings.Join(query, " and "))
	if e.debug {
		log.Println(q)
	}
	rows, err := e.conn.Query(q)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		newObj := e.newBy(m)
		values := e.genValues(newObj)
		rows.Scan(values...)
		if elemType ==reflect.Ptr{ //for outSlice= []*Object
			value.Set(reflect.Append(value, reflect.ValueOf(newObj)))
		}else{//for outSlice= []Object
			value.Set(reflect.Append(value, reflect.ValueOf(newObj).Elem()))
		}
	}
	return nil
}

func (e *DbHelper) Create(m interface{}) (err error) {
	table, _, idv, _, values := e.genInfo(m)
	var vals []string
	var fields []string
	for k, v := range values {
		fields = append(fields, fmt.Sprintf("%v", k))
		vals = append(vals, fmt.Sprintf("%v", v))
	}
	q := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s) ", table, strings.Join(fields, ","), strings.Join(vals, ","))
	if e.debug {
		log.Println(q)
	}
	result, err := e.conn.Exec(q)
	newId, _ := result.LastInsertId()
	idv.SetInt(newId)
	return
}

func (e *DbHelper) Update(m interface{}) (err error) {
	table, id, idv, _, values := e.genInfo(m)
	if !idv.IsValid() {
		return fmt.Errorf("Update must set id")
	}
	var updates []string
	for k, v := range values {
		if k != id {
			updates = append(updates, fmt.Sprintf("%s=%v", k, v))
		}
	}
	q := fmt.Sprintf("UPDATE %s SET %s WHERE %s=%v", table, strings.Join(updates, ","), id, idv)
	if e.debug {
		log.Println(q)
	}
	_, err = e.conn.Exec(q)
	return
}

func (e *DbHelper) Delete(m interface{}) (err error) {
	table, id, idv, _, _ := e.genInfo(m)
	q := fmt.Sprintf("DELETE FROM %s WHERE %s=%v ", table, id, idv)
	if e.debug {
		log.Println(q)
	}
	_, err = e.conn.Exec(q)
	return
}

func (e *DbHelper) SqlMap(query string)([]map[string]string, error){
	rows, _ := e.conn.Query(query)
	cols, _ := rows.Columns()
	var result []map[string]string
	for rows.Next() {
		columns := make([]string, len(cols))
		columnPointers := make([]interface{}, len(cols))
		for i := range columns {
			columnPointers[i] = &columns[i]
		}

		rows.Scan(columnPointers...)
		data := make(map[string]string)
		for i, colName := range cols {
			data[colName] = columns[i]
		}
		result = append(result, data)
	}
	return result,nil
}

func (e *DbHelper) SqlSlice(query string)([][]string, error){
	rows, _ := e.conn.Query(query)
	cols, _ := rows.Columns()
	var result [][]string
	for rows.Next() {
		columns := make([]string, len(cols))
		columnPointers := make([]interface{}, len(cols))
		for i:= range columns {
			columnPointers[i] = &columns[i]
		}

		rows.Scan(columnPointers...)
		var data  []string
		for i := range cols {
			data =append(data, columns[i])
		}
		result = append(result, data)
	}
	return result,nil
}

func (e *DbHelper) SqlStructMap(where string,outMap interface{})(err error) {
	valuePtr := reflect.ValueOf(outMap)
	value := valuePtr.Elem()
	elemType := valuePtr.Type().Elem()
	elemKind := elemType.Kind()
	if elemKind!=reflect.Map{
		return fmt.Errorf("params in must is slice")
	}

	if valuePtr.Kind()==reflect.Ptr{
		elemType =reflect.TypeOf(value.Interface()).Elem()
		elemKind = elemType.Kind()
	}
	if elemKind ==reflect.Ptr{
		if elemType.Elem().Kind()!= reflect.Struct {
			return fmt.Errorf("params element  must is struct")
		}
	}else{
		if elemType.Kind()!= reflect.Struct {
			return fmt.Errorf("params element  must is struct")
		}
	}
	var m interface{}
	if elemKind ==reflect.Ptr { //for *Object
		m = reflect.New(elemType.Elem()).Interface()
	}else{ //for Object
		m = reflect.New(elemType).Elem().Interface()
	}

	table, _, _, fields, _ := e.genInfo(m)

	q := fmt.Sprintf("SELECT %s FROM %s  %s ", strings.Join(fields, ","), table, where)
	if e.debug {
		log.Println(q)
	}
	rows, err := e.conn.Query(q)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		newObj := e.newBy(m)
		values := e.genValues(newObj)
		newValue := e.getElem(newObj)
		idv:=newValue.FieldByName("ID")
		rows.Scan(values...)

		if elemKind ==reflect.Ptr{ //for outSlice= []*Object
			value.SetMapIndex(idv, reflect.ValueOf(newObj))
		}else{
			value.SetMapIndex(idv, reflect.ValueOf(newObj).Elem())
		}
	}

	return nil

}
func (e *DbHelper) SqlStructSlice(where string,outSlice interface{})(err error){
	valuePtr := reflect.ValueOf(outSlice)
	value := valuePtr.Elem()
	elemType := valuePtr.Type().Elem()
	elemKind := elemType.Kind()
	if elemKind!=reflect.Slice{
		return fmt.Errorf("params in must is slice")
	}

	if valuePtr.Kind()==reflect.Ptr{
		elemType =reflect.TypeOf(value.Interface()).Elem()
		elemKind = elemType.Kind()
	}
	if elemKind ==reflect.Ptr{
		if elemType.Elem().Kind()!= reflect.Struct {
			return fmt.Errorf("params element  must is struct")
		}
	}else{
		if elemType.Kind()!= reflect.Struct {
			return fmt.Errorf("params element  must is struct")
		}
	}


	var m interface{}
	if elemKind ==reflect.Ptr { //for *Object
		m = reflect.New(elemType.Elem()).Interface()
	}else{ //for Object
		m = reflect.New(elemType).Elem().Interface()
	}

	table, _, _, fields, _ := e.genInfo(m)

	q := fmt.Sprintf("SELECT %s FROM %s  %s ", strings.Join(fields, ","), table, where)
	if e.debug {
		log.Println(q)
	}
	rows, err := e.conn.Query(q)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		newObj := e.newBy(m)
		values := e.genValues(newObj)
		rows.Scan(values...)
		if elemKind ==reflect.Ptr{ //for outSlice= []*Object
			value.Set(reflect.Append(value, reflect.ValueOf(newObj)))
		}else{//for outSlice= []Object
			value.Set(reflect.Append(value, reflect.ValueOf(newObj).Elem()))
		}
	}

	return nil
}

func (e *DbHelper) Close() error  {
	return e.db.Close()
}

func (e *DbHelper) StructToMap(m interface{}) map[string]interface{} {
	var result map[string]interface{}
	temp, _ := json.Marshal(m)
	json.Unmarshal(temp, &result)
	return result
}

func (e *DbHelper) MapToStruct(m map[string]interface{}) interface{}  {
	var result interface{}
	temp, _ := json.Marshal(m)
	json.Unmarshal(temp, &result)
	return result
}

func (e *DbHelper) genInfo(in interface{}) (table string, pk string, pkv reflect.Value, fields []string, values map[string]string) {
	elem := e.getElem(in)
	values = make(map[string]string)
	var fieldName string
	typ := elem.Type()
	fCount := elem.NumField()
	for i := 0; i < fCount; i++ {
		fieldType := typ.Field(i)
		field := elem.Field(i)
		if tag, ok := fieldType.Tag.Lookup("db"); ok {
			if tag == "-" {
				continue
			}
			fieldName = tag
		} else {
			fieldName = toSnake(fieldType.Name)
		}

		if tag, ok := fieldType.Tag.Lookup("pk"); ok {
			pk = tag
			pkv = field
		} else {
			if fieldName == "id" {
				pk = fieldName
				pkv = field
			}
		}
		fields = append(fields, fieldName)
		//todo 这里先注释，因为更新时即使是空的，就是要更新成空的，如何解决
		//if elemHaveValue(field) {
			values[fieldName]= getFieldValue(field)
			//values[fieldName] = fmt.Sprintf("\"%v\"", getFieldValue(field))
		//}
	}
	if pk == "" {
		pk = fields[0]
		pkv = elem.Field(0)
	}
	table = strings.ToLower(typ.Name())
	return
}

func getFieldValue(v reflect.Value) string {
	fieldValue := v.Interface()

	switch v := fieldValue.(type) {
	case int64:
		return strconv.FormatInt(v, 10)
	case int32:
		return strconv.FormatInt(int64(v), 10)
	case int:
		return strconv.FormatInt(int64(v), 10)
	case string:
		return fmt.Sprintf("\"%v\"", v)
	case bool:
		if v {
			return "true"
		}
		return "false"
	case sql.NullTime:
		return fmt.Sprintf("\"%v\"", v.Time.Format(mySQLTimeFormat))
	case time.Time:
		//return v.String()
		return fmt.Sprintf("\"%v\"", v.Format(mySQLTimeFormat))
	default:
		return ""
	}
}
func (e *DbHelper) genValues(in interface{}) (out []interface{}) {
	v := e.getElem(in)
	typ := v.Type()
	fCount := v.NumField()
	for i := 0; i < fCount; i++ {
		ft := typ.Field(i)
		if tag, ok := ft.Tag.Lookup("db"); ok {
			if tag == "-" {
				continue
			}
		}
		out = append(out, v.Field(i).Addr().Interface())
	}
	return
}

//Create new instance from exists obj type
//reflect.New(reflect.TypeOf(m)).Interface()
func (e *DbHelper) newBy(in interface{}) (out interface{}) {
	v := reflect.ValueOf(in)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	typ := v.Type()
	out = reflect.New(typ).Interface()
	return
}

func (e *DbHelper) getElem(in interface{}) reflect.Value {
	v := reflect.ValueOf(in)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		fmt.Errorf(" Only accepts structs; got %T", v)
	}
	return v
}

func (e *DbHelper) setID(m interface{}, id string, idv int64) {
	reflect.ValueOf(m).Elem().FieldByName(id).SetInt(idv)
}

func elemHaveValue(field reflect.Value) bool {
	return !reflect.DeepEqual(field.Interface(), reflect.Zero(field.Type()).Interface())
}

var camel = regexp.MustCompile("(^[^A-Z]*|[A-Z]*)([A-Z][^A-Z]+|$)")

func toSnake(s string) string {
	s = strings.TrimSpace(s)
	var a []string
	for _, sub := range camel.FindAllStringSubmatch(s, -1) {
		if sub[1] != "" {
			a = append(a, strings.TrimSpace(sub[1]))
		}
		if sub[2] != "" {
			a = append(a, strings.TrimSpace(sub[2]))
		}
	}
	return strings.ToLower(strings.Join(a, "_"))
}

var link = regexp.MustCompile("(^[A-Za-z])|_([A-Za-z])")

func toCamelCase(str string) string {
	return link.ReplaceAllStringFunc(str, func(s string) string {
		return strings.ToUpper(strings.Replace(s, "_", "", -1))
	})
}
