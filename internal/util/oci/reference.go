package oci

import digest "github.com/opencontainers/go-digest"

type Reference struct {
	Tag    string
	Digest string
}

func ParseReference(s string) (Reference, error) {
	tryDigest, err := digest.Parse(s)
	if err == nil {
		return Reference{
			Digest: tryDigest.String(),
		}, nil
	}
	tag := s
	return Reference{
		Tag: tag,
	}, nil
}

func (r Reference) String() string {
	if r.IsDigest() {
		return r.Digest
	}
	if r.IsTag() {
		return r.Tag
	}
	return ""
}

func (r Reference) IsTag() bool {
	return r.Tag != ""
}

func (r Reference) IsDigest() bool {
	return r.Digest != ""
}
