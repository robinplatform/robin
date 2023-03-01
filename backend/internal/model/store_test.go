package model

import (
	"path/filepath"
	"testing"
)

func TestStoreWrite(t *testing.T) {
	type Data struct {
		Id   string
		Name string
	}

	dir := t.TempDir()
	db, err := NewStore[Data](filepath.Join(dir, "test"))
	if err != nil {
		t.Fatal(err)
	}

	{
		r := db.ReadHandle()

		_, found := r.Find(func(row Data) bool {
			return row.Name == "Hello"
		})

		if found {
			t.Fatalf("found a row when there were none inserted")
		}

		r.Close()
	}

	{
		w := db.WriteHandle()

		w.Insert(Data{Id: "hello", Name: "hello"})

		_, found := w.Find(func(row Data) bool {
			return row.Name == "Hello"
		})

		if found {
			t.Fatalf("found a row when there were none that should have matched")
		}

		w.Close()
	}

	{
		w := db.WriteHandle()

		w.Insert(Data{Id: "meh", Name: "Hello"})

		d, found := w.Find(func(row Data) bool {
			return row.Name == "Hello"
		})

		if !found {
			t.Fatalf("didn't find a row when a matching row exists")
		}

		if d.Id != "meh" {
			t.Fatalf("ID was not the right ID")
		}

		w.Close()
	}

	{
		w := db.WriteHandle()

		w.Delete(func(row Data) bool {
			return row.Name == "Hello"
		})

		_, found := w.Find(func(row Data) bool {
			return row.Name == "Hello"
		})

		if found {
			t.Fatalf("found a row that should have been deleted")
		}

		w.Close()
	}
}
