package orm

import (
	"database/sql"
	"errors"
	"fmt"
	zormlog "github.com/caixr9527/zorm/log"
	"reflect"
	"strings"
	"time"
)

type ZDb struct {
	db     *sql.DB
	logger *zormlog.Logger
	Prefix string
}

type DbSession struct {
	db          *ZDb
	tableName   string
	fieldName   []string
	placeHolder []string
	values      []any
	updateParam strings.Builder
	whereParam  strings.Builder
	whereValues []any
}

func Open(driverName, source string) *ZDb {
	db, err := sql.Open(driverName, source)
	if err != nil {
		panic(err)
	}

	db.SetMaxIdleConns(5)
	db.SetMaxOpenConns(100)
	db.SetConnMaxLifetime(time.Minute * 3)
	db.SetConnMaxIdleTime(time.Minute * 1)

	zdb := &ZDb{
		db:     db,
		logger: zormlog.Default(),
	}
	err = db.Ping()
	if err != nil {
		panic(err)
	}
	return zdb
}

func (db *ZDb) Close() error {
	return db.db.Close()
}

func (db *ZDb) New() *DbSession {
	return &DbSession{
		db: db,
	}
}

func (db *ZDb) SetMaxIdleConns(n int) {
	db.db.SetMaxIdleConns(n)
}

func (session *DbSession) Table(name string) *DbSession {
	session.tableName = name
	return session
}

func (session *DbSession) Insert(data any) (int64, int64, error) {
	err := session.fieldNames(data)
	if err != nil {
		return -1, -1, err
	}
	query := fmt.Sprintf("insert into %s (%s) values (%s)", session.tableName, strings.Join(session.fieldName, ","), strings.Join(session.placeHolder, ","))
	session.db.logger.Info(query)
	stmt, err := session.db.db.Prepare(query)
	if err != nil {
		return -1, -1, err
	}
	r, err := stmt.Exec(session.values...)
	if err != nil {
		return -1, -1, err
	}
	id, err := r.LastInsertId()
	if err != nil {
		return -1, -1, err
	}
	affected, err := r.RowsAffected()
	if err != nil {
		return -1, -1, err
	}
	return id, affected, nil
}

func (session *DbSession) fieldNames(data any) error {
	t := reflect.TypeOf(data)
	v := reflect.ValueOf(data)
	if t.Kind() != reflect.Pointer {
		return errors.New("data type must be pointer")
	}
	tVar := t.Elem()
	vVar := v.Elem()
	if session.tableName == "" {
		session.tableName = session.db.Prefix + strings.ToLower(Name(tVar.Name()))
	}
	for i := 0; i < tVar.NumField(); i++ {
		fieldName := tVar.Field(i).Name
		tag := tVar.Field(i).Tag
		sqlTag := tag.Get("zorm")
		if sqlTag == "" {
			sqlTag = strings.ToLower(Name(fieldName))
		} else {
			if strings.Contains(sqlTag, "auto_increment") {
				continue
			}
			if strings.Contains(sqlTag, ",") {
				sqlTag = sqlTag[:strings.Index(sqlTag, ",")]
			}
		}
		id := vVar.Field(i).Interface()
		if strings.ToLower(sqlTag) == "id" && IsAutoId(id) {
			continue
		}
		session.fieldName = append(session.fieldName, sqlTag)
		session.placeHolder = append(session.placeHolder, "?")
		session.values = append(session.values, vVar.Field(i).Interface())
	}
	return nil
}

func (session *DbSession) InsertBatch(data []any) (int64, int64, error) {
	if len(data) == 0 {
		return -1, -1, errors.New("not data insert")
	}
	err := session.fieldNames(data[0])
	if err != nil {
		return -1, -1, err
	}
	query := fmt.Sprintf("insert into %s (%s) values", session.tableName, strings.Join(session.fieldName, ","))
	var sb strings.Builder
	sb.WriteString(query)
	for index, _ := range data {
		sb.WriteString("(")
		sb.WriteString(strings.Join(session.placeHolder, ","))
		sb.WriteString(")")
		if index < len(data)-1 {
			sb.WriteString(",")
		}

	}

	err = session.batchValues(data)
	if err != nil {
		return -1, -1, err
	}
	session.db.logger.Info(sb.String())
	stmt, err := session.db.db.Prepare(sb.String())
	if err != nil {
		return -1, -1, err
	}
	r, err := stmt.Exec(session.values...)
	if err != nil {
		return -1, -1, err
	}
	id, err := r.LastInsertId()
	if err != nil {
		return -1, -1, err
	}
	affected, err := r.RowsAffected()
	if err != nil {
		return -1, -1, err
	}
	return id, affected, nil
}

func (session *DbSession) batchValues(data []any) error {
	session.values = make([]any, 0)
	for _, v := range data {
		t := reflect.TypeOf(v)
		v := reflect.ValueOf(v)
		if t.Kind() != reflect.Pointer {
			return errors.New("data type must be pointer")
		}
		tVar := t.Elem()
		vVar := v.Elem()
		for i := 0; i < tVar.NumField(); i++ {
			fieldName := tVar.Field(i).Name
			tag := tVar.Field(i).Tag
			sqlTag := tag.Get("zorm")
			if sqlTag == "" {
				sqlTag = strings.ToLower(Name(fieldName))
			} else {
				if strings.Contains(sqlTag, "auto_increment") {
					continue
				}
			}
			id := vVar.Field(i).Interface()
			if strings.ToLower(sqlTag) == "id" && IsAutoId(id) {
				continue
			}
			session.values = append(session.values, vVar.Field(i).Interface())
		}
	}
	return nil
}

func (session *DbSession) Update(data ...any) (int64, int64, error) {
	if len(data) == 0 || len(data) > 2 {
		return -1, -1, errors.New("param not valid")
	}
	single := true
	if len(data) == 2 {
		single = false
	}
	if !single {
		if session.updateParam.String() != "" {
			session.updateParam.WriteString(",")
		}
		session.updateParam.WriteString(data[0].(string))
		session.updateParam.WriteString(" = ?")
		session.values = append(session.values, data[1])
	}
	query := fmt.Sprintf("update %s set %s", session.tableName, session.updateParam.String())
	var sb strings.Builder
	sb.WriteString(query)
	sb.WriteString(session.whereParam.String())
	session.db.logger.Info(sb.String())
	stmt, err := session.db.db.Prepare(sb.String())
	if err != nil {
		return -1, -1, err
	}
	session.values = append(session.values, session.whereValues...)
	r, err := stmt.Exec(session.values...)
	if err != nil {
		return -1, -1, err
	}
	id, err := r.LastInsertId()
	if err != nil {
		return -1, -1, err
	}
	affected, err := r.RowsAffected()
	if err != nil {
		return -1, -1, err
	}
	return id, affected, nil

}

func (session *DbSession) Where(field string, value any) *DbSession {
	if session.whereParam.String() == "" {
		session.whereParam.WriteString(" where ")
	} else {
		session.whereParam.WriteString(", ")
	}
	session.whereParam.WriteString(field)
	session.whereParam.WriteString(" = ")
	session.whereParam.WriteString(" ? ")
	session.whereValues = append(session.whereValues, value)
	return session
}

func IsAutoId(id any) bool {
	t := reflect.TypeOf(id)
	switch t.Kind() {
	case reflect.Int64:
		if id.(int64) <= 0 {
			return true
		}
	case reflect.Int32:
		if id.(int32) <= 0 {
			return true
		}
	case reflect.Int:
		if id.(int) <= 0 {
			return true
		}
	}
	return false
}

func Name(name string) string {
	var names = name[:]
	lastIndex := 0
	var sb strings.Builder
	for index, value := range names {
		if value >= 65 && value <= 90 {
			// 大写
			if index == 0 {
				continue
			}
			sb.WriteString(name[:index])
			sb.WriteString("_")
			lastIndex = index
		}
	}
	sb.WriteString(names[lastIndex:])
	return sb.String()
}
