package wsgraphql

import (
	"bufio"
	"bytes"
	"encoding/json"
	"testing"

	"github.com/eientei/wsgraphql/v1/apollows"
	"github.com/stretchr/testify/assert"
)

func TestNewServerPlain(t *testing.T) {
	srv := testNewServer(t, apollows.WebsocketSubprotocolGraphqlWS)

	defer srv.Close()

	client := srv.Client()

	query := srv.URL + "/query"

	var pd apollows.PayloadDataResponse

	bs, err := json.Marshal(apollows.PayloadOperation{
		Query: `query { getFoo }`,
	})

	assert.NoError(t, err)

	resp, err := client.Post(query, "application/json", bytes.NewReader(bs))

	assert.NoError(t, err)

	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&pd))

	assert.NoError(t, resp.Body.Close())

	assert.NoError(t, err)
	assert.Len(t, pd.Errors, 0)
	assert.EqualValues(t, 123, pd.Data["getFoo"])

	bs, err = json.Marshal(apollows.PayloadOperation{
		Query: `mutation { setFoo(value: 3) }`,
	})

	assert.NoError(t, err)

	resp, err = client.Post(query, "application/json", bytes.NewReader(bs))

	assert.NoError(t, err)

	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&pd))

	assert.NoError(t, resp.Body.Close())

	assert.NoError(t, err)
	assert.Len(t, pd.Errors, 0)
	assert.EqualValues(t, true, pd.Data["setFoo"])

	bs, err = json.Marshal(apollows.PayloadOperation{
		Query: `mutation { setFoo }`,
	})

	assert.NoError(t, err)

	resp, err = client.Post(query, "application/json", bytes.NewReader(bs))

	assert.NoError(t, err)

	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&pd))

	assert.NoError(t, resp.Body.Close())

	assert.NoError(t, err)
	assert.Len(t, pd.Errors, 0)
	assert.EqualValues(t, false, pd.Data["setFoo"])

	bs, err = json.Marshal(apollows.PayloadOperation{
		Query: `mutation { bar }`,
	})

	assert.NoError(t, err)

	resp, err = client.Post(query, "application/json", bytes.NewReader(bs))

	assert.NoError(t, err)

	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&pd))

	assert.NoError(t, resp.Body.Close())

	assert.NoError(t, err)
	assert.Greater(t, len(pd.Errors), 0)

	bs, err = json.Marshal(apollows.PayloadOperation{
		Query: `subscription { fooUpdates }`,
	})

	assert.NoError(t, err)

	resp, err = client.Post(query, "application/json", bytes.NewReader(bs))

	assert.NoError(t, err)

	scanner := bufio.NewScanner(resp.Body)

	idx := 1

	for scanner.Scan() {
		if len(scanner.Bytes()) > 0 {
			pd = apollows.PayloadDataResponse{}

			assert.NoError(t, json.Unmarshal(scanner.Bytes(), &pd))

			assert.Len(t, pd.Errors, 0)
			assert.EqualValues(t, idx, pd.Data["fooUpdates"])
			idx++
		}
	}

	assert.NoError(t, resp.Body.Close())
}
