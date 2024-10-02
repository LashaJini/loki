package audit

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/go-kit/log"
	"github.com/stretchr/testify/require"

	"github.com/grafana/loki/v3/pkg/compactor"
	"github.com/grafana/loki/v3/pkg/compactor/retention"
	"github.com/grafana/loki/v3/pkg/storage/chunk/client"
)

var errObjectNotFound = errors.New("object not found")

type testObjClient struct {
	client.ObjectClient
}

func (t testObjClient) ObjectExists(ctx context.Context, object string) (bool, error) {
	if _, err := t.GetAttributes(ctx, object); err != nil {
		if t.IsObjectNotFoundErr(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (t testObjClient) IsObjectNotFoundErr(err error) bool {
	return errors.Is(err, errObjectNotFound)
}

func (t testObjClient) GetAttributes(_ context.Context, object string) (client.ObjectAttributes, error) {
	if strings.Contains(object, "missing") {
		return client.ObjectAttributes{}, errObjectNotFound
	}
	return client.ObjectAttributes{}, nil
}

type testCompactedIdx struct {
	compactor.CompactedIndex

	chunks []retention.ChunkEntry
}

func (t testCompactedIdx) ForEachChunk(_ context.Context, f retention.ChunkEntryCallback) error {
	for _, chunk := range t.chunks {
		if _, err := f(chunk); err != nil {
			return err
		}
	}
	return nil
}

func TestAuditIndex(t *testing.T) {
	ctx := context.Background()
	objClient := testObjClient{}
	compactedIdx := testCompactedIdx{
		chunks: []retention.ChunkEntry{
			{ChunkRef: retention.ChunkRef{ChunkID: []byte("found-1")}},
			{ChunkRef: retention.ChunkRef{ChunkID: []byte("found-2")}},
			{ChunkRef: retention.ChunkRef{ChunkID: []byte("found-3")}},
			{ChunkRef: retention.ChunkRef{ChunkID: []byte("found-4")}},
			{ChunkRef: retention.ChunkRef{ChunkID: []byte("missing-1")}},
		},
	}
	logger := log.NewNopLogger()
	found, missing, err := ValidateCompactedIndex(ctx, objClient, compactedIdx, 1, logger)
	require.NoError(t, err)
	require.Equal(t, 4, found)
	require.Equal(t, 1, missing)
}
