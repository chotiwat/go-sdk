package migration

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand"
	"testing"

	"github.com/blend/go-sdk/assert"
	"github.com/blend/go-sdk/db"
)

var (
	// lowerLetters is a runset of lowercase letters.
	lowerLetters = []rune("abcdefghijklmnopqrstuvwxyz")
)

func randomName() string {
	runes := make([]rune, 12)
	letterCount := len(lowerLetters)
	for index := range runes {
		runes[index] = lowerLetters[rand.Intn(letterCount)]
	}
	return string(runes)
}

func createTestTable(tableName string, tx *sql.Tx) error {
	body := fmt.Sprintf("CREATE TABLE %s (id int, name varchar(32));", tableName)
	step := Step(TableNotExists(tableName), Statements(body))
	return step.Action(context.Background(), db.Default(), tx)
}

func insertTestValue(tableName string, id int, name string, tx *sql.Tx) error {
	body := fmt.Sprintf("INSERT INTO %s (id, name) VALUES ($1, $2);", tableName)
	return db.Default().Invoke(db.OptTx(tx)).Exec(body, id, name)
}

func createTestColumn(tableName, columnName string, tx *sql.Tx) error {
	body := fmt.Sprintf("ALTER TABLE %s ADD %s varchar(32);", tableName, columnName)
	step := Step(ColumnNotExists(tableName, columnName), Statements(body))
	return step.Action(context.Background(), db.Default(), tx)
}

func createTestConstraint(tableName, constraintName string, tx *sql.Tx) error {
	body := fmt.Sprintf("ALTER TABLE %s ADD CONSTRAINT %s UNIQUE (name);", tableName, constraintName)
	step := Step(ColumnNotExists(tableName, constraintName), Statements(body))
	return step.Action(context.Background(), db.Default(), tx)
}

func createTestIndex(tableName, indexName string, tx *sql.Tx) error {
	body := fmt.Sprintf("CREATE INDEX %s ON %s (name);", indexName, tableName)
	step := Step(IndexNotExists(tableName, indexName), Statements(body))
	return step.Action(context.Background(), db.Default(), tx)
}

func createTestRole(roleName string, tx *sql.Tx) error {
	body := fmt.Sprintf("CREATE ROLE %s;", roleName)
	step := Step(RoleNotExists(roleName), Statements(body))
	return step.Action(context.Background(), db.Default(), tx)
}

func TestCreateTable(t *testing.T) {
	assert := assert.New(t)
	tx, err := db.Default().Begin()
	assert.Nil(err)
	defer tx.Rollback()

	tableName := randomName()
	err = createTestTable(tableName, nil)
	assert.Nil(err)

	exists, err := PredicateTableExists(db.Default(), nil, tableName)
	assert.Nil(err)
	assert.True(exists, "table does not exist")
}

func TestCreateColumn(t *testing.T) {
	assert := assert.New(t)
	tx, err := db.Default().Begin()
	assert.Nil(err)
	defer tx.Rollback()

	tableName := randomName()
	err = createTestTable(tableName, tx)
	assert.Nil(err)

	columnName := randomName()
	err = createTestColumn(tableName, columnName, tx)
	assert.Nil(err)

	exists, err := PredicateColumnExists(db.Default(), tx, tableName, columnName)
	assert.Nil(err)
	assert.True(exists, "column does not exist on table")
}

func TestCreateConstraint(t *testing.T) {
	assert := assert.New(t)
	tx, err := db.Default().Begin()
	assert.Nil(err)
	defer tx.Rollback()

	tableName := randomName()
	err = createTestTable(tableName, tx)
	assert.Nil(err)

	constraintName := fmt.Sprintf("uk_%s_%s", tableName, randomName())
	err = createTestConstraint(tableName, constraintName, tx)
	assert.Nil(err)

	exists, err := PredicateConstraintExists(db.Default(), tx, constraintName)
	assert.Nil(err)
	assert.True(exists, "constraint does not exist")
}

func TestCreateIndex(t *testing.T) {
	assert := assert.New(t)
	tx, err := db.Default().Begin()
	assert.Nil(err)
	defer tx.Rollback()

	tableName := randomName()
	err = createTestTable(tableName, tx)
	assert.Nil(err)

	indexName := fmt.Sprintf("ix_%s_%s", tableName, randomName())
	err = createTestIndex(tableName, indexName, tx)
	assert.Nil(err)

	exists, err := PredicateIndexExists(db.Default(), tx, tableName, indexName)
	assert.Nil(err)
	assert.True(exists, "constraint should exist")
}

func TestCreateRole(t *testing.T) {
	assert := assert.New(t)
	tx, err := db.Default().Begin()
	assert.Nil(err)
	defer tx.Rollback()

	roleName := randomName()
	err = createTestRole(roleName, tx)
	assert.Nil(err)

	exists, err := PredicateRoleExists(db.Default(), tx, roleName)
	assert.Nil(err)
	assert.True(exists, "role does exist")
}
