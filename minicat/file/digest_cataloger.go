package file

import (
	"crypto"
	"errors"
	"fmt"
	"hash"
	"io"
	"strings"

	"github.com/lovewebshell/minicat/internal"
	"github.com/lovewebshell/minicat/internal/log"
	"github.com/lovewebshell/minicat/minicat/source"
)

var errUndigestableFile = errors.New("undigestable file")

type DigestsCataloger struct {
	hashes []crypto.Hash
}

func NewDigestsCataloger(hashes []crypto.Hash) (*DigestsCataloger, error) {
	return &DigestsCataloger{
		hashes: hashes,
	}, nil
}

func (i *DigestsCataloger) Catalog(resolver source.FileResolver) (map[source.Coordinates][]Digest, error) {
	results := make(map[source.Coordinates][]Digest)
	locations := allRegularFiles(resolver)

	for _, location := range locations {

		result, err := i.catalogLocation(resolver, location)

		if errors.Is(err, errUndigestableFile) {
			continue
		}

		if internal.IsErrPathPermission(err) {
			log.Debugf("file digests cataloger skipping %q: %+v", location.RealPath, err)
			continue
		}

		if err != nil {
			return nil, err
		}

		results[location.Coordinates] = result
	}

	return results, nil
}

func (i *DigestsCataloger) catalogLocation(resolver source.FileResolver, location source.Location) ([]Digest, error) {
	meta, err := resolver.FileMetadataByLocation(location)
	if err != nil {
		return nil, err
	}

	if meta.Type != source.RegularFile {
		return nil, errUndigestableFile
	}

	contentReader, err := resolver.FileContentsByLocation(location)
	if err != nil {
		return nil, err
	}
	defer internal.CloseAndLogError(contentReader, location.VirtualPath)

	digests, err := DigestsFromFile(contentReader, i.hashes)
	if err != nil {
		return nil, internal.ErrPath{Context: "digests-cataloger", Path: location.RealPath, Err: err}
	}

	return digests, nil
}

func DigestsFromFile(closer io.ReadCloser, hashes []crypto.Hash) ([]Digest, error) {

	hashers := make([]hash.Hash, len(hashes))
	writers := make([]io.Writer, len(hashes))
	for idx, hashObj := range hashes {
		hashers[idx] = hashObj.New()
		writers[idx] = hashers[idx]
	}

	size, err := io.Copy(io.MultiWriter(writers...), closer)
	if err != nil {
		return nil, err
	}

	if size == 0 {
		return make([]Digest, 0), nil
	}

	result := make([]Digest, len(hashes))

	for idx, hasher := range hashers {
		result[idx] = Digest{
			Algorithm: DigestAlgorithmName(hashes[idx]),
			Value:     fmt.Sprintf("%+x", hasher.Sum(nil)),
		}
	}

	return result, nil
}

func DigestAlgorithmName(hash crypto.Hash) string {
	return CleanDigestAlgorithmName(hash.String())
}

func CleanDigestAlgorithmName(name string) string {
	lower := strings.ToLower(name)
	return strings.ReplaceAll(lower, "-", "")
}
