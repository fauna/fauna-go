package fauna_test

import (
	"os"
	"sync"
	"testing"

	"github.com/fauna/fauna-go"
	"github.com/stretchr/testify/assert"
)

func TestClientRetriesWithQueryLimits(t *testing.T) {
	t.Setenv(fauna.EnvFaunaSecret, "secret")
	dbName := os.Getenv("QUERY_LIMITS_DB")
	collName := os.Getenv("QUERY_LIMITS_COLL")

	t.Logf("%s, %s", dbName, collName)

	client, clientErr := fauna.NewDefaultClient()
	if !assert.NoError(t, clientErr) {
		return
	}

	t.Run("Query limits succeed on retry", func(t *testing.T) {
		if dbName == "" || collName == "" {
			t.Skip("Skipping query limits test due to missing env var")
		}

		type secretObj struct {
			Secret string `fauna:"secret"`
		}

		query, _ := fauna.FQL(`
		if (Database.byName(${dbName}).exists()) {
      Key.create({ role: "admin", database: ${dbName} }) { secret }
    } else {
      abort("Database not found.")
    }`, map[string]any{"dbName": dbName})

		res, queryErr := client.Query(query)
		if !assert.NoError(t, queryErr) {
			t.FailNow()
		}

		var secret secretObj
		marshalErr := res.Unmarshal(&secret)
		if assert.NoError(t, marshalErr) {
			clients := make([]*fauna.Client, 5)
			results := make(chan int, len(clients))

			var wg sync.WaitGroup
			wg.Add(len(clients))

			for i := range clients {
				clients[i] = fauna.NewClient(secret.Secret, fauna.DefaultTimeouts(), fauna.URL(os.Getenv(fauna.EnvFaunaEndpoint)))

				go func(collName string, client *fauna.Client, result chan int) {
					defer wg.Done()
					coll, _ := fauna.FQL(collName, nil)
					q, _ := fauna.FQL(`${coll}.all().paginate(50)`, map[string]any{"coll": coll})
					res, err := client.Query(q)
					if err != nil {
						result <- -1
					} else {
						result <- res.Stats.Attempts
					}
				}(collName, clients[i], results)
			}

			go func() {
				wg.Wait()
				close(results)
			}()

			throttled := false

			for result := range results {
				throttled = throttled || result > 1
			}

			assert.True(t, throttled)
		}
	})
}
