package adapter

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"strings"
	"time"

	"github.com/ipfs/go-blockservice"
	"github.com/ipfs/go-cid"
	bsfetcher "github.com/ipfs/go-fetcher/impl/blockservice"
	files "github.com/ipfs/go-ipfs-files"
	format "github.com/ipfs/go-ipld-format"
	"github.com/ipfs/go-merkledag"
	path "github.com/ipfs/go-path"
	pathresolver "github.com/ipfs/go-path/resolver"
	unixfile "github.com/ipfs/go-unixfs/file"
	"github.com/ipfs/go-unixfsnode"
	dagpb "github.com/ipld/go-codec-dagpb"
	"github.com/ipld/go-ipld-prime"
	basicnode "github.com/ipld/go-ipld-prime/node/basic"
	"github.com/ipld/go-ipld-prime/schema"
)

type unixfsFs struct {
	ctx      context.Context
	rootCid  cid.Cid
	bsvc     blockservice.BlockService
	dsvc     format.DAGService
	resolver pathresolver.Resolver
}

func NewFs(root cid.Cid, bsvc blockservice.BlockService) (fs.FS, error) {
	return NewFsWithContext(context.Background(), root, bsvc)
}

func NewFsWithContext(ctx context.Context, root cid.Cid, bsvc blockservice.BlockService) (fs.FS, error) {
	ipldFetcher := bsfetcher.NewFetcherConfig(bsvc)
	ipldFetcher.PrototypeChooser = dagpb.AddSupportToChooser(func(lnk ipld.Link, lnkCtx ipld.LinkContext) (ipld.NodePrototype, error) {
		if tlnkNd, ok := lnkCtx.LinkNode.(schema.TypedLinkNode); ok {
			return tlnkNd.LinkTargetNodePrototype(), nil
		}
		return basicnode.Prototype.Any, nil
	})
	unixFSFetcher := ipldFetcher.WithReifier(unixfsnode.Reify)

	return &unixfsFs{
		ctx:      ctx,
		rootCid:  root,
		bsvc:     bsvc,
		dsvc:     merkledag.NewDAGService(bsvc),
		resolver: pathresolver.NewBasicResolver(unixFSFetcher),
	}, nil
}

func (fs *unixfsFs) Open(name string) (fs.File, error) {
	var ipfsPath path.Path
	if name == "/" {
		ipfsPath = path.FromString("/ipfs/" + fs.rootCid.String())
	} else {
		if !strings.HasPrefix(name, "/") {
			return nil, errors.New("path must start with \"/\"")
		}
		ipfsPath = path.FromString(fmt.Sprintf("/ipfs/%s%s", fs.rootCid, name))
	}

	cid, rest, err := fs.resolver.ResolveToLastNode(fs.ctx, ipfsPath)
	if err != nil {
		return nil, err
	}
	if len(rest) > 0 {
		return nil, errors.New("cannot resolve to last node")
	}

	nd, err := fs.dsvc.Get(fs.ctx, cid)
	if err != nil {
		return nil, err
	}

	f, err := unixfile.NewUnixfsFile(fs.ctx, fs.dsvc, nd)
	if err != nil {
		return nil, err
	}

	_, fname, err := ipfsPath.PopLastSegment()
	if err != nil {
		return nil, err
	}

	return NewFile(fname, f)
}

var _ fs.FS = (*unixfsFs)(nil)

type unixfsFile struct {
	info fs.FileInfo
	node files.Node
}

// NewFile creates an fs.File that is backed by an IPFS files.Node.
func NewFile(name string, node files.Node) (fs.File, error) {
	size, err := node.Size()
	if err != nil {
		return nil, err
	}
	_, isDir := node.(files.Directory)
	return &unixfsFile{
		info: &unixfsFileInfo{
			name:    name,
			size:    size,
			modTime: time.Now(),
			isDir:   isDir,
		},
		node: node,
	}, nil
}

func (uf *unixfsFile) Stat() (fs.FileInfo, error) {
	return uf.info, nil
}

func (uf *unixfsFile) Read(p []byte) (int, error) {
	if ff, ok := uf.node.(files.File); ok {
		return ff.Read(p)
	}
	return 0, errors.New("file not readable")
}

func (uf *unixfsFile) Close() error {
	return uf.node.Close()
}

func (uf *unixfsFile) ReadDir(n int) ([]fs.DirEntry, error) {
	fd, isDir := uf.node.(files.Directory)
	if !isDir {
		return nil, errors.New("not a directory")
	}

	var ents []fs.DirEntry
	it := fd.Entries()
	for it.Next() {
		f, err := NewFile(it.Name(), it.Node())
		if err != nil {
			return nil, err
		}
		i, err := f.Stat()
		if err != nil {
			return nil, err
		}
		ents = append(ents, &unixfsDirEntry{i})
	}
	if it.Err() != nil {
		return nil, it.Err()
	}

	return ents, nil
}

var _ fs.File = (*unixfsFile)(nil)
var _ fs.ReadDirFile = (*unixfsFile)(nil)

type unixfsFileInfo struct {
	name    string
	size    int64
	modTime time.Time
	isDir   bool
}

func (i *unixfsFileInfo) Name() string {
	return i.name
}

func (i *unixfsFileInfo) Size() int64 {
	return i.size
}

func (i *unixfsFileInfo) Mode() fs.FileMode {
	if i.isDir {
		return fs.ModeDir | 0555
	}
	return fs.ModePerm
}

func (i *unixfsFileInfo) ModTime() time.Time {
	return i.modTime
}

func (i *unixfsFileInfo) IsDir() bool {
	return i.Mode().IsDir()
}

func (i *unixfsFileInfo) Sys() interface{} {
	return nil
}

var _ fs.FileInfo = (*unixfsFileInfo)(nil)

type unixfsDirEntry struct {
	info fs.FileInfo
}

func (e *unixfsDirEntry) Name() string {
	return e.info.Name()
}

func (e *unixfsDirEntry) IsDir() bool {
	return e.info.IsDir()
}

func (e *unixfsDirEntry) Type() fs.FileMode {
	return e.info.Mode().Type()
}

func (e *unixfsDirEntry) Info() (fs.FileInfo, error) {
	return e.info, nil
}

var _ fs.DirEntry = (*unixfsDirEntry)(nil)
