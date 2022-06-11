package apollows

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDataMarshal(t *testing.T) {
	data := Data{
		Value: 123,
	}

	bs, err := json.Marshal(data)

	assert.NoError(t, err)
	assert.Equal(t, "123", string(bs))
}

func TestDataUnmarshal(t *testing.T) {
	var data Data

	err := json.Unmarshal([]byte(`123`), &data)

	assert.NoError(t, err)
	assert.Equal(t, "123", string(data.RawMessage))
}

func TestDataUnmarshalCycle(t *testing.T) {
	var data Data

	err := json.Unmarshal([]byte(`123`), &data)

	assert.NoError(t, err)
	assert.Equal(t, "123", string(data.RawMessage))

	bs, err := json.Marshal(data)

	assert.NoError(t, err)
	assert.Equal(t, "123", string(bs))
}

func TestDataReadPayloadData(t *testing.T) {
	data := Data{
		Value: PayloadData{
			Data: Data{
				Value: map[string]interface{}{
					"foo": "123",
				},
			},
		},
	}

	bs, err := json.Marshal(data)

	assert.NoError(t, err)

	var ndata Data

	err = json.Unmarshal(bs, &ndata)

	assert.NoError(t, err)

	pd, err := ndata.ReadPayloadData()

	assert.NoError(t, err)

	assert.Equal(t, "123", pd.Data["foo"])
}
