package fauna_test

import (
	"os"
	"sync"
	"testing"

	"github.com/fauna/fauna-go/v3"
	"github.com/stretchr/testify/assert"
)

func TestClientRetriesWithQueryLimits(t *testing.T) {
	t.Run("Query limits succeed on retry", func(t *testing.T) {
		dbName, dbNameSet := os.LookupEnv("QUERY_LIMITS_DB")
		collName, collNameSet := os.LookupEnv("QUERY_LIMITS_COLL")

		// If run in a pipeline, these will be empty strings, so check both
		if (!dbNameSet || !collNameSet) ||
			(dbName == "" || collName == "") {
			t.Skip("Skipping query limits test due to missing env var(s)")
		}

		if _, found := os.LookupEnv(fauna.EnvFaunaSecret); !found {
			t.Setenv(fauna.EnvFaunaSecret, "secret")
		}

		client, clientErr := fauna.NewDefaultClient()
		if !assert.NoError(t, clientErr) {
			return
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
