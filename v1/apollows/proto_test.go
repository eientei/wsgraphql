package apollows

import (
	"encoding/json"
	"errors"
	"io"
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

func TestDataReadPayloadDataError(t *testing.T) {
	var ndata *Data

	pd, err := ndata.ReadPayloadData()

	assert.Error(t, err)
	assert.Nil(t, pd)

	ndata = &Data{
		Value:      nil,
		RawMessage: json.RawMessage(`foo`),
	}

	pd, err = ndata.ReadPayloadData()

	assert.Error(t, err)
	assert.Nil(t, pd)
}

func TestDataReadPayloadError(t *testing.T) {
	data := Data{
		Value: PayloadError{
			Message: "123",
		},
	}

	bs, err := json.Marshal(data)

	assert.NoError(t, err)

	var ndata Data

	err = json.Unmarshal(bs, &ndata)

	assert.NoError(t, err)

	pd, err := ndata.ReadPayloadError()

	assert.NoError(t, err)
	assert.Equal(t, "123", pd.Message)
}

func TestDataReadPayloadErrorError(t *testing.T) {
	var ndata *Data

	pd, err := ndata.ReadPayloadError()

	assert.Error(t, err)
	assert.Nil(t, pd)

	ndata = &Data{
		Value:      nil,
		RawMessage: json.RawMessage(`foo`),
	}

	pd, err = ndata.ReadPayloadError()

	assert.Error(t, err)
	assert.Nil(t, pd)
}

func TestDataReadPayloadErrors(t *testing.T) {
	data := Data{
		Value: []PayloadError{
			{
				Message: "123",
			},
		},
	}

	bs, err := json.Marshal(data)

	assert.NoError(t, err)

	var ndata Data

	err = json.Unmarshal(bs, &ndata)

	assert.NoError(t, err)

	pd, err := ndata.ReadPayloadErrors()

	assert.NoError(t, err)
	assert.Len(t, pd, 1)
	assert.Equal(t, "123", pd[0].Message)
}

func TestDataReadPayloadErrorsError(t *testing.T) {
	var ndata *Data

	pd, err := ndata.ReadPayloadErrors()

	assert.Error(t, err)
	assert.Nil(t, pd)

	ndata = &Data{
		Value:      nil,
		RawMessage: json.RawMessage(`foo`),
	}

	pd, err = ndata.ReadPayloadErrors()

	assert.Error(t, err)
	assert.Nil(t, pd)
}

func TestWrapError(t *testing.T) {
	err := WrapError(io.EOF, EventUnauthorized)

	assert.True(t, errors.Is(err, io.EOF))
}
