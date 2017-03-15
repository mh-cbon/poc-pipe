package t

import (
	"sort"

	"github.com/mh-cbon/semver"
)

type VersionPipeWriter interface {
	Flusher
	Write(*semver.Version) error
}

type VersionStream struct {
	Streams []VersionPipeWriter
}

func (p *VersionStream) Pipe(s Piper) Piper {
	// add lock
	p.Sink(s)
	return s
}
func (p *VersionStream) Sink(s Flusher) {
	// add lock
	x, ok := s.(VersionPipeWriter)
	if !ok {
		panic("nop")
	}
	p.Streams = append(p.Streams, x)
}
func (p *VersionStream) Unpipe(s Flusher) {
	// add lock
}
func (p *VersionStream) Flush() error {
	for _, pp := range p.Streams {
		if err := pp.Flush(); err != nil {
			return err
		}
	}
	return nil
}
func (p *VersionStream) Write(d *semver.Version) error {
	for _, pp := range p.Streams {
		if err := pp.Write(d); err != nil {
			return err
		}
	}
	return nil
}

type VersionFromByte struct {
	VersionStream
}

func (p *VersionFromByte) Write(d []byte) error {
	s, err := semver.NewVersion(string(d))
	if err != nil {
		return err
	}
	return p.VersionStream.Write(s)
}

type VersionConstraint struct {
	VersionStream
	c *semver.Constraints
}

func NewVersionContraint(c *semver.Constraints) *VersionConstraint {
	return &VersionConstraint{c: c}
}

func (p *VersionConstraint) Write(v *semver.Version) error {
	if p.c.Check(v) {
		return p.VersionStream.Write(v)
	}
	return nil
}

type VersionSorter struct {
	VersionStream
	all []*semver.Version
	Asc bool
}

func (p *VersionSorter) Write(v *semver.Version) error {
	p.all = append(p.all, v)
	return nil
}

func (p *VersionSorter) Flush() error {
	if p.Asc {
		sort.Sort(semver.Collection(p.all))
	} else {
		sort.Sort(sort.Reverse(semver.Collection(p.all)))
	}
	for _, v := range p.all {
		p.VersionStream.Write(v)
	}
	p.all = p.all[:0]
	return p.VersionStream.Flush()
}

type InvalidVersionFromByte struct {
	ByteStream
}

func (p *InvalidVersionFromByte) Write(d []byte) error {
	_, err := semver.NewVersion(string(d))
	if err == nil {
		return nil
	}
	return p.ByteStream.Write(d)
}

type VersionToByte struct {
	ByteStream
}

func (p *VersionToByte) Write(d *semver.Version) error {
	return p.ByteStream.Write([]byte(d.String()))
}
