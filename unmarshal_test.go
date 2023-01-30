package fauna_test

import (
	"log"
	"testing"
	"time"

	"github.com/fauna/fauna-go"
)

func TestUnmarshal(t *testing.T) {
	type Result struct {
		Foobar  string
		Updated time.Time `json:"time"`
		Child   struct {
			Yesterday time.Time
			Money     float64
			Id        int64
			Units     int32
			Ancestor  struct {
				Birthday time.Time
			}
		}
	}

	var result Result
	err := fauna.Unmarshal([]byte(`
	{
		"foobar": "steve",
		"time": { "@time": "2023-01-28T15:09:32.099Z" },
		"child": {
			"yesterday": { "@time": "2023-01-28T15:09:32.099Z" },
			"money": { "@double": "3.14" },
			"id": { "@long": "81237613421" },
			"units": { "@int": "1834" },
			"ancestor": { "birthday": { "@date": "1971-06-13" } }
		}
}`), &result)
	if err != nil {
		log.Fatalf("err: %s", err.Error())
	}

	if result.Foobar != "steve" {
		t.Errorf("expected steve, got: %s", result.Foobar)
	}

	if result.Child.Yesterday.IsZero() {
		t.Errorf("should have a time for yesterday")
	}

	if result.Child.Units != int32(1834) {
		t.Errorf("expected 1834, got %v", result.Child.Units)
	}
}
