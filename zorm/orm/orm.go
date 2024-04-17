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

func (db *ZDb) New(data any) *DbSession {
	m := &DbSession{
		db: db,
	}
	t := reflect.TypeOf(data)
	if t.Kind() != reflect.Pointer {
		panic(errors.New("data type must be pointer"))
	}
	tVar := t.Elem()
	if m.tableName == "" {
		m.tableName = m.db.Prefix + strings.ToLower(Name(tVar.Name()))
	}
	return m
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

func (session *DbSession) UpdateParam(field string, value any) *DbSession {
	if session.updateParam.String() != "" {
		session.updateParam.WriteString(",")
	}
	session.updateParam.WriteString(field)
	session.updateParam.WriteString(" = ?")
	session.values = append(session.values, value)
	return session
}

func (session *DbSession) UpdateMap(data map[string]any) *DbSession {
	for k, v := range data {
		if session.updateParam.String() != "" {
			session.updateParam.WriteString(",")
		}
		session.updateParam.WriteString(k)
		session.updateParam.WriteString(" = ?")
		session.values = append(session.values, v)
	}
	return session
}

func (session *DbSession) Update(data ...any) (int64, int64, error) {
	if len(data) > 2 {
		return -1, -1, errors.New("param not valid")
	}
	if len(data) == 0 {
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
	} else {
		updateData := data[0]
		t := reflect.TypeOf(updateData)
		v := reflect.ValueOf(updateData)
		if t.Kind() != reflect.Pointer {
			return -1, -1, errors.New("updateData type must be pointer")
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
			if session.updateParam.String() != "" {
				session.updateParam.WriteString(",")
			}
			session.updateParam.WriteString(sqlTag)
			session.updateParam.WriteString(" = ?")
			session.values = append(session.values, vVar.Field(i).Interface())
		}
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

func (session *DbSession) Delete() (int64, error) {
	query := fmt.Sprintf("delete from %s ", session.tableName)
	var sb strings.Builder
	sb.WriteString(query)
	sb.WriteString(session.whereParam.String())
	session.db.logger.Info(sb.String())
	stmt, err := session.db.db.Prepare(sb.String())
	if err != nil {
		return 0, err
	}
	exec, err := stmt.Exec(session.whereParam)
	if err != nil {
		return 0, err
	}
	return exec.RowsAffected()
}

func (session *DbSession) Select(data any, fields ...string) ([]any, error) {
	t := reflect.TypeOf(data)
	if t.Kind() != reflect.Pointer {
		return nil, errors.New("data must be pointer")
	}
	fieldStr := "*"
	if len(fields) > 0 {
		fieldStr = strings.Join(fields, ",")
	}
	query := fmt.Sprintf("select %s from %s", fieldStr, session.tableName)
	var sb strings.Builder
	sb.WriteString(query)
	sb.WriteString(session.whereParam.String())
	session.db.logger.Info(sb.String())
	stmt, err := session.db.db.Prepare(sb.String())
	if err != nil {
		return nil, err
	}
	rows, err := stmt.Query(session.whereValues...)
	if err != nil {
		return nil, err
	}
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	result := make([]any, 0)
	for {
		if rows.Next() {
			data := reflect.New(t.Elem()).Interface()
			values := make([]any, len(columns))
			fieldScan := make([]any, len(columns))
			for i := range fieldScan {
				fieldScan[i] = &values[i]
			}
			err := rows.Scan(fieldScan...)
			if err != nil {
				return nil, err
			}
			tVar := t.Elem()
			vVar := reflect.ValueOf(data).Elem()
			for i := 0; i < tVar.NumField(); i++ {
				name := tVar.Field(i).Name
				tag := tVar.Field(i).Tag
				sqlTag := tag.Get("zorm")
				if sqlTag == "" {
					sqlTag = strings.ToLower(Name(name))
				} else {
					if strings.Contains(sqlTag, ",") {
						sqlTag = sqlTag[:strings.Index(sqlTag, ",")]
					}
				}
				for j, colName := range columns {
					if sqlTag == colName {
						target := values[j]
						targetValue := reflect.ValueOf(target)
						fieldType := tVar.Field(i).Type
						result := reflect.ValueOf(targetValue.Interface()).Convert(fieldType)
						vVar.Field(i).Set(result)
					}
				}
			}
			result = append(result, data)
		} else {
			break
		}
	}
	return result, nil
}

func (session *DbSession) SelectOne(data any, fields ...string) error {
	t := reflect.TypeOf(data)
	if t.Kind() != reflect.Pointer {
		return errors.New("data must be pointer")
	}
	fieldStr := "*"
	if len(fields) > 0 {
		fieldStr = strings.Join(fields, ",")
	}
	query := fmt.Sprintf("select %s from %s", fieldStr, session.tableName)
	var sb strings.Builder
	sb.WriteString(query)
	sb.WriteString(session.whereParam.String())
	session.db.logger.Info(sb.String())
	stmt, err := session.db.db.Prepare(sb.String())
	if err != nil {
		return err
	}
	rows, err := stmt.Query(session.whereValues...)
	if err != nil {
		return err
	}
	columns, err := rows.Columns()
	if err != nil {
		return err
	}
	values := make([]any, len(columns))
	fieldScan := make([]any, len(columns))
	for i := range fieldScan {
		fieldScan[i] = &values[i]
	}
	if rows.Next() {
		err := rows.Scan(fieldScan...)
		if err != nil {
			return err
		}
		tVar := t.Elem()
		vVar := reflect.ValueOf(data).Elem()
		for i := 0; i < tVar.NumField(); i++ {
			name := tVar.Field(i).Name
			tag := tVar.Field(i).Tag
			sqlTag := tag.Get("zorm")
			if sqlTag == "" {
				sqlTag = strings.ToLower(Name(name))
			} else {
				if strings.Contains(sqlTag, ",") {
					sqlTag = sqlTag[:strings.Index(sqlTag, ",")]
				}
			}
			for j, colName := range columns {
				if sqlTag == colName {
					target := values[j]
					targetValue := reflect.ValueOf(target)
					fieldType := tVar.Field(i).Type
					result := reflect.ValueOf(targetValue.Interface()).Convert(fieldType)
					vVar.Field(i).Set(result)
				}
			}
		}
	}
	return nil
}

func (session *DbSession) Where(field string, value any) *DbSession {
	if session.whereParam.String() == "" {
		session.whereParam.WriteString(" where ")
	}
	session.whereParam.WriteString(field)
	session.whereParam.WriteString(" = ")
	session.whereParam.WriteString(" ? ")
	session.whereValues = append(session.whereValues, value)
	return session
}

func (session *DbSession) Like(field string, value any) *DbSession {
	if session.whereParam.String() == "" {
		session.whereParam.WriteString(" where ")
	}
	session.whereParam.WriteString(field)
	session.whereParam.WriteString(" like ")
	session.whereParam.WriteString(" ? ")
	session.whereValues = append(session.whereValues, "%"+value.(string)+"%")
	return session
}

func (session *DbSession) LikeRight(field string, value any) *DbSession {
	if session.whereParam.String() == "" {
		session.whereParam.WriteString(" where ")
	}
	session.whereParam.WriteString(field)
	session.whereParam.WriteString(" like ")
	session.whereParam.WriteString(" ? ")
	session.whereValues = append(session.whereValues, value.(string)+"%")
	return session
}

func (session *DbSession) LikeLeft(field string, value any) *DbSession {
	if session.whereParam.String() == "" {
		session.whereParam.WriteString(" where ")
	}
	session.whereParam.WriteString(field)
	session.whereParam.WriteString(" like ")
	session.whereParam.WriteString(" ? ")
	session.whereValues = append(session.whereValues, "%"+value.(string)+"%")
	return session
}

func (session *DbSession) Group(field ...string) *DbSession {

	session.whereParam.WriteString(" group by ")
	session.whereParam.WriteString(strings.Join(field, ","))
	return session
}

func (session *DbSession) OrderDesc(field ...string) *DbSession {

	session.whereParam.WriteString(" order by ")
	session.whereParam.WriteString(strings.Join(field, ","))
	session.whereParam.WriteString(" desc ")
	return session
}

func (session *DbSession) OrderAsc(field ...string) *DbSession {

	session.whereParam.WriteString(" order by ")
	session.whereParam.WriteString(strings.Join(field, ","))
	session.whereParam.WriteString(" asc ")
	return session
}

func (session *DbSession) Order(field ...string) *DbSession {
	if len(field)%2 != 0 {
		panic("field num not true")
	}
	session.whereParam.WriteString(" order by ")
	for index, v := range field {
		session.whereParam.WriteString(v + " ")
		if index%2 != 0 && index < len(field)-1 {
			session.whereParam.WriteString(",")
		}
	}
	return session
}

func (session *DbSession) And() *DbSession {
	session.whereParam.WriteString(" and ")
	return session
}

func (session *DbSession) Or() *DbSession {
	session.whereParam.WriteString(" or ")
	return session
}

func (session *DbSession) Count() (int64, error) {
	return 0, nil
}

func (session *DbSession) Aggregate() (int64, error) {
	return 0, nil
}

func (session *DbSession) Exec(sql string, values ...any) (int64, error) {
	stmt, err := session.db.db.Prepare(sql)
	if err != nil {
		return 0, err
	}
	r, err := stmt.Exec(values)
	if err != nil {
		return 0, err
	}
	if strings.Contains(strings.ToLower(sql), "insert") {
		return r.LastInsertId()
	}
	return r.RowsAffected()
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
