package rdb_test

import (
	"os"
	"strings"
	"testing"

	rdb "github.com/ross96D/rdb-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Must[T any](s T, err error) T {
	if err != nil {
		panic(err)
	}
	return s
}

func Get(db rdb.Database, t *testing.T, key string) []byte {
	r, err := db.Get([]byte(key))
	require.NoError(t, err)
	return r.Bytes
}

func TestDatabase(t *testing.T) {
	db, err := rdb.New([]byte("test_db"))
	require.NoError(t, err)
	defer os.Remove("test_db")

	require.NoError(t, db.Set([]byte("key1"), []byte("value1")))
	require.NoError(t, db.Set([]byte("key2"), []byte("value2")))
	require.NoError(t, db.Set([]byte("key3"), []byte("value3")))
	require.NoError(t, db.Set([]byte("key4"), []byte("value4")))

	assert.Equal(t, []byte("value1"), Get(db, t, "key1"))
	assert.Equal(t, []byte("value2"), Get(db, t, "key2"))
	assert.Equal(t, []byte("value3"), Get(db, t, "key3"))
	assert.Equal(t, []byte("value4"), Get(db, t, "key4"))

	require.NoError(t, db.Set([]byte("key1"), []byte("val1")))
	require.NoError(t, db.Set([]byte("key2"), []byte("val2")))
	require.NoError(t, db.Set([]byte("key3"), []byte("val3")))
	require.NoError(t, db.Set([]byte("key4"), []byte("val4")))

	assert.Equal(t, []byte("val1"), Get(db, t, "key1"))
	assert.Equal(t, []byte("val2"), Get(db, t, "key2"))
	assert.Equal(t, []byte("val3"), Get(db, t, "key3"))
	assert.Equal(t, []byte("val4"), Get(db, t, "key4"))

	var count int = 0
	var keys [][]byte = [][]byte{}
	var values [][]byte = [][]byte{}
	require.NoError(t, db.ForEach(func(b1, b2 []byte) bool {
		count += 1
		keys = append(keys, b1)
		values = append(values, b2)
		return true
	}))
	assert.Equal(t, 4, count)

	for i, s := range keys {
		require.NoError(t, db.Remove(s))
		assert.True(t, strings.HasPrefix(string(values[i]), "val"))
		assert.True(t, strings.HasPrefix(string(s), "key"))
	}

	_, err = db.Get([]byte("key1"))
	assert.ErrorAs(t, err, &rdb.ErrNotFound{})
	_, err = db.Get([]byte("key2"))
	assert.ErrorAs(t, err, &rdb.ErrNotFound{})
	_, err = db.Get([]byte("key3"))
	assert.ErrorAs(t, err, &rdb.ErrNotFound{})
	_, err = db.Get([]byte("key4"))
	assert.ErrorAs(t, err, &rdb.ErrNotFound{})

	db.Close()
	_, err = db.Get([]byte("key1"))
	assert.ErrorIs(t, err, rdb.ErrCloseDB)
	err = db.Remove([]byte("key1"))
	assert.ErrorIs(t, err, rdb.ErrCloseDB)
	err = db.Set([]byte("key1"), []byte("value1"))
	assert.ErrorIs(t, err, rdb.ErrCloseDB)
	assert.ErrorIs(t, db.ForEach(func(b1, b2 []byte) bool {
		count += 1
		keys = append(keys, b1)
		values = append(values, b2)
		return true
	}), rdb.ErrCloseDB)

}
