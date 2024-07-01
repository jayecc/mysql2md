package main

import (
	"errors"
	"flag"
	"fmt"
	"github.com/duke-git/lancet/v2/fileutil"
	"github.com/schollz/progressbar/v3"
	"golang.org/x/sync/errgroup"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"log"
	"os"
	"runtime"
	"strings"
)

var (
	dsn     = flag.String("dsn", "username:password@tcp(localhost:3306)/database?charset=utf8mb4&parseTime=True&loc=Local&timeout=10s", "database connection string")
	dir     = flag.String("dir", "./output", "directory to save the file")
	isWhole = flag.Bool("whole", false, "generate whole file (default false)")
	isDDL   = flag.Bool("ddl", false, "generate ddl info (default false)")
	db      *gorm.DB
)

func handleError(err error) {
	if err == nil {
		return
	}
	log.Fatal("ERROR", " ", err)
}

func initDB() (err error) {
	db, err = gorm.Open(mysql.Open(*dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	return err
}

func initDir() (err error) {
	if fileutil.IsExist(*dir) {
		return nil
	}
	return fileutil.CreateDir(*dir)
}

type Table struct {
	Name       string `gorm:"column:Name"`
	Engine     string `gorm:"column:Engine"`
	Collation  string `gorm:"column:Collation"`
	Comment    string `gorm:"column:Comment"`
	CreateTime string `gorm:"column:Create_time"`
}

type TableDDL struct {
	Table       string `gorm:"column:Table"`
	CreateTable string `gorm:"column:Create Table"`
}

type TableColumn struct {
	TableCatalog           string `gorm:"column:TABLE_CATALOG"`
	TableSchema            string `gorm:"column:TABLE_SCHEMA"`
	TableName              string `gorm:"column:TABLE_NAME"`
	ColumnName             string `gorm:"column:COLUMN_NAME"`
	OrdinalPosition        int    `gorm:"column:ORDINAL_POSITION"`
	ColumnDefault          string `gorm:"column:COLUMN_DEFAULT"`
	IsNullable             string `gorm:"column:IS_NULLABLE"`
	DataType               string `gorm:"column:DATA_TYPE"`
	CharacterMaximumLength int    `gorm:"column:CHARACTER_MAXIMUM_LENGTH"`
	CharacterOctetLength   int    `gorm:"column:CHARACTER_OCTET_LENGTH"`
	NumericPrecision       int    `gorm:"column:NUMERIC_PRECISION"`
	NumericScale           int    `gorm:"column:NUMERIC_SCALE"`
	DatetimePrecision      int    `gorm:"column:DATETIME_PRECISION"`
	CharacterSetName       string `gorm:"column:CHARACTER_SET_NAME"`
	CollationName          string `gorm:"column:COLLATION_NAME"`
	ColumnType             string `gorm:"column:COLUMN_TYPE"`
	ColumnKey              string `gorm:"column:COLUMN_KEY"`
	Extra                  string `gorm:"column:EXTRA"`
	Privileges             string `gorm:"column:PRIVILEGES"`
	ColumnComment          string `gorm:"column:COLUMN_COMMENT"`
	GenerationExpression   string `gorm:"column:GENERATION_EXPRESSION"`
}

func databaseName() (name string, err error) {
	err = db.Raw("SELECT DATABASE()").Scan(&name).Error
	return
}

func tableList() (tables []Table, err error) {
	err = db.Raw("SHOW TABLE STATUS").Scan(&tables).Error
	return
}

func tableColumns(databaseName, tableName string) (columns []TableColumn, err error) {
	err = db.Raw("SELECT * FROM information_schema.columns WHERE table_schema = ? AND table_name = ?", databaseName, tableName).Scan(&columns).Error
	return
}

func tableDDL(tableName string) (ddl *TableDDL, err error) {
	ddl = &TableDDL{}
	err = db.Raw(fmt.Sprintf("SHOW CREATE TABLE `%s`", tableName)).Scan(ddl).Error
	return
}

func tableListToFile(databaseName string, tables []Table) (err error) {

	f, err := os.OpenFile(fmt.Sprintf("%s/%s.%s.md", *dir, databaseName, "tables"), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.ModePerm)
	if err != nil {
		return err
	}
	defer f.Close()

	_, _ = f.WriteString(fmt.Sprintf("# %s tables list \n", databaseName))
	_, _ = f.WriteString("| Name | Engine | Create_time | Collation | Comment |\n")
	_, _ = f.WriteString("| ---- | ------ | ----------- | --------- | ------- |\n")

	for _, table := range tables {
		_, _ = f.WriteString(fmt.Sprintf("| [%s](%s.%s.md) | %s | %s | %s | %s |\n", table.Name, databaseName, table.Name, table.Engine, table.CreateTime, table.Collation, strings.ReplaceAll(table.Comment, "\n", "")))
	}

	return
}

func tableToFile(databaseName string, table Table, columns []TableColumn, ddl *TableDDL) (err error) {

	f, err := os.OpenFile(fmt.Sprintf("%s/%s.%s.md", *dir, databaseName, table.Name), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.ModePerm)
	if err != nil {
		return err
	}
	defer f.Close()

	_, _ = f.WriteString(fmt.Sprintf("# %s.%s\n> %s\n", databaseName, table.Name, table.Comment))
	_, _ = f.WriteString("### COLUMNS\n")
	_, _ = f.WriteString("| COLUMN_NAME | COLUMN_DEFAULT | IS_NULLABLE | COLLATION_NAME | COLUMN_TYPE | COLUMN_KEY | EXTRA | COLUMN_COMMENT |\n")
	_, _ = f.WriteString("| ----------- | -------------- | ----------- | -------------- | ----------- | ---------- | ----- | -------------- |\n")

	for _, column := range columns {
		_, _ = f.WriteString(fmt.Sprintf("| %s | %s | %s | %s | %s | %s | %s | %s |\n", column.ColumnName, column.ColumnDefault, column.IsNullable, column.CollationName, column.ColumnType, column.ColumnKey, column.Extra, strings.ReplaceAll(column.ColumnComment, "\n", "")))
	}

	if !*isDDL {
		return
	}
	_, _ = f.WriteString("### DDL\n")
	_, _ = f.WriteString(fmt.Sprintf("```sql\n%s\n```\n", ddl.CreateTable))

	return
}

func wholeToFile(databaseName string, tables []Table) (err error) {

	f, err := os.OpenFile(fmt.Sprintf("%s/%s.%s.md", *dir, databaseName, "tables"), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.ModePerm)
	if err != nil {
		return err
	}
	defer f.Close()

	bar := progressbar.Default(int64(len(tables)))
	defer bar.Close()

	wg := errgroup.Group{}
	wg.SetLimit(runtime.NumCPU())

	for _, table := range tables {
		table := table
		wg.Go(func() error {

			columns, err := tableColumns(databaseName, table.Name)
			handleError(err)

			ddl, err := tableDDL(table.Name)
			handleError(err)

			_, _ = f.WriteString(fmt.Sprintf("# %s.%s\n> %s\n", databaseName, table.Name, table.Comment))
			_, _ = f.WriteString("### COLUMNS\n")
			_, _ = f.WriteString("| COLUMN_NAME | COLUMN_DEFAULT | IS_NULLABLE | COLLATION_NAME | COLUMN_TYPE | COLUMN_KEY | EXTRA | COLUMN_COMMENT |\n")
			_, _ = f.WriteString("| ----------- | -------------- | ----------- | -------------- | ----------- | ---------- | ----- | -------------- |\n")

			for _, column := range columns {
				_, _ = f.WriteString(fmt.Sprintf("| %s | %s | %s | %s | %s | %s | %s | %s |\n", column.ColumnName, column.ColumnDefault, column.IsNullable, column.CollationName, column.ColumnType, column.ColumnKey, column.Extra, strings.ReplaceAll(column.ColumnComment, "\n", "")))
			}

			defer bar.Add(1)

			if !*isDDL {
				return nil
			}

			_, _ = f.WriteString("### DDL\n")
			_, _ = f.WriteString(fmt.Sprintf("```sql\n%s\n```\n", ddl.CreateTable))

			return nil
		})
	}

	return wg.Wait()
}

func main() {

	flag.Parse()

	if *dsn == "" {
		handleError(errors.New("dsn is empty"))
	}

	if err := initDB(); err != nil {
		handleError(err)
	}

	if err := initDir(); err != nil {
		handleError(err)
	}

	dbName, err := databaseName()
	handleError(err)

	tables, err := tableList()
	handleError(err)

	if *isWhole {
		err = wholeToFile(dbName, tables)
		handleError(err)
		return
	}

	err = tableListToFile(dbName, tables)
	handleError(err)

	bar := progressbar.Default(int64(len(tables)))
	defer bar.Close()

	wg := errgroup.Group{}
	wg.SetLimit(runtime.NumCPU())

	for _, table := range tables {
		table := table
		wg.Go(func() error {

			columns, err := tableColumns(dbName, table.Name)
			handleError(err)

			ddl, err := tableDDL(table.Name)
			handleError(err)

			err = tableToFile(dbName, table, columns, ddl)
			handleError(err)

			return bar.Add(1)
		})
	}
	_ = wg.Wait()
}
