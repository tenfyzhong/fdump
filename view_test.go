package fdump

import (
	"testing"

	"github.com/google/gopacket"
	"github.com/rivo/tview"
	"github.com/stretchr/testify/assert"
)

func brief(record *Record) []string {
	return []string{}
}

func detail(record *Record) string {
	return ""
}

func decode(net, transport gopacket.Flow, data []byte) (bodies []interface{}, n int, err error) {
	return
}

func TestNewView(t *testing.T) {
	tapp := tview.NewApplication()
	capacity := 100
	replayHook := &ReplayHook{}
	briefAttributes := []*BriefColumnAttribute{
		&BriefColumnAttribute{
			Title:    "title0",
			MaxWidth: 10,
		},
		&BriefColumnAttribute{
			Title:    "title1",
			MaxWidth: 20,
		},
	}

	v := newView(tapp, capacity, brief, detail, decode, replayHook, briefAttributes)
	assert.NotNil(t, v)
	assert.Equal(t, tapp, v.app)
	assert.Equal(t, capacity, v.capacity)
	assert.Equal(t, briefAttributes, v.briefAttributes)
	assert.NotNil(t, v.multis)
	assert.NotNil(t, v.messages)
	assert.Equal(t, capacity, len(v.messages))

	maxWidth := 2 + seqColumnAttribute.MaxWidth
	for _, brief := range briefAttributes {
		maxWidth += brief.MaxWidth + 1
	}
	assert.Equal(t, maxWidth, v.briefWidth)
}

func TestBitSet(t *testing.T) {
	status := uint64(0)
	bitSet(&status, uint64(1))
	assert.Equal(t, status, uint64(1))
}

func TestBitClear(t *testing.T) {
	status := uint64(3)
	bitClear(&status, uint64(1))
	assert.Equal(t, status, uint64(2))
}

func TestIsSet(t *testing.T) {
	assert.True(t, isSet(uint64(1), uint64(1)))
	assert.False(t, isSet(uint64(2), uint64(1)))
}
