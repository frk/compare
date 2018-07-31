### compare
---------------------------------------

The package compare facilitates the comparison of two Go values providing
a detailed error message in case when the comparison fails.

It is intended to be used during testing with its main objective being the
comparison of large and deeply nested values.



#### Example

```go

type Author struct {
	FirstName string
	LastName  string
}

type Publisher struct {
	Name string
	HQ   interface{}
}

type Book struct {
	ISBN       interface{}
	Title      string
	ReleasedAt time.Time
	Authors    []*Author
	Publisher  *Publisher
}

func TestExample(t *testing.T) {
	got := &Book{
		ISBN:       4101001545,
		Title:      "海辺のカフカ",
		ReleasedAt: time.Date(2005, time.March, 1, 0, 0, 0, 0, time.UTC),
		Authors: []*Author{{
			LastName:  "村上",
			FirstName: "春樹",
		}},
		Publisher: &Publisher{
			Name: "新潮社",
			HQ:   "JP",
		},
	}
	want := &Book{
		ISBN:       "0099458322",
		Title:      "Kafka on the Shore",
		ReleasedAt: time.Date(2005, time.October, 6, 0, 0, 0, 0, time.UTC),
		Authors: []*Author{{
			FirstName: "Haruki",
			LastName:  "Murakami",
		}},
		Publisher: &Publisher{
			Name: "Vintage",
			HQ:   nil,
		},
	}

	if err := Compare(got, want); err != nil {
		t.Error(err)
	}
}

```

The output of which would look something like this:

![example output](https://github.com/frk/compare/raw/master/images/output_example2.png)