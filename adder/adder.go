// Simplified copy of ipfs-cluster/adder/ipfsadd/add.go
package adder

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	gopath "path"

	cid "github.com/ipfs/go-cid"
	chunker "github.com/ipfs/go-ipfs-chunker"
	posinfo "github.com/ipfs/go-ipfs-posinfo"
	ipld "github.com/ipfs/go-ipld-format"
	dag "github.com/ipfs/go-merkledag"
	mfs "github.com/ipfs/go-mfs"
	unixfs "github.com/ipfs/go-unixfs"
	balanced "github.com/ipfs/go-unixfs/importer/balanced"
	ihelper "github.com/ipfs/go-unixfs/importer/helpers"
	w3fs "github.com/web3-storage/go-w3s-client/fs"
)

const chnkr = "size-1048576"
const maxLinks = 1024
const rawLeaves = true

var cidBuilder = dag.V1CidPrefix()
var liveCacheSize = uint64(256 << 10)

// NewAdder Returns a new Adder used for a file add operation.
func NewAdder(ctx context.Context, ds ipld.DAGService) (*Adder, error) {
	return &Adder{
		ctx:        ctx,
		dagService: ds,
	}, nil
}

// Adder is an filesystem interface adder for web3.storage.
type Adder struct {
	ctx        context.Context
	dagService ipld.DAGService
	mroot      *mfs.Root
	liveNodes  uint64
}

func (adder *Adder) Add(file fs.File, dirname string, fsys fs.FS) (cid.Cid, error) {
	if fsys == nil {
		fsys = &w3fs.OsFs{}
	}

	fi, err := file.Stat()
	if err != nil {
		return cid.Undef, err
	}

	nd, err := adder.addAll(file, fi, dirname, fsys)
	if err != nil {
		return cid.Undef, err
	}
	return nd.Cid(), nil
}

func (adder *Adder) MfsRoot() (*mfs.Root, error) {
	if adder.mroot != nil {
		return adder.mroot, nil
	}
	rnode := unixfs.EmptyDirNode()
	rnode.SetCidBuilder(cidBuilder)
	mr, err := mfs.NewRoot(adder.ctx, adder.dagService, rnode, nil)
	if err != nil {
		return nil, err
	}
	adder.mroot = mr
	return adder.mroot, nil
}

// Constructs a node from reader's data, and adds it.
func (adder *Adder) add(reader io.Reader) (ipld.Node, error) {
	chnk, err := chunker.FromString(reader, chnkr)
	if err != nil {
		return nil, err
	}

	params := ihelper.DagBuilderParams{
		Dagserv:    adder.dagService,
		RawLeaves:  rawLeaves,
		Maxlinks:   maxLinks,
		CidBuilder: cidBuilder,
	}

	db, err := params.New(chnk)
	if err != nil {
		return nil, err
	}

	nd, err := balanced.Layout(db)
	if err != nil {
		return nil, err
	}

	return nd, nil
}

func (adder *Adder) addNode(node ipld.Node, path string) error {
	// patch it into the root
	if path == "" {
		path = node.Cid().String()
	}

	if pi, ok := node.(*posinfo.FilestoreNode); ok {
		node = pi.Node
	}

	mr, err := adder.MfsRoot()
	if err != nil {
		return err
	}
	dir := gopath.Dir(path)
	if dir != "." {
		opts := mfs.MkdirOpts{
			Mkparents:  true,
			Flush:      false,
			CidBuilder: cidBuilder,
		}
		if err := mfs.Mkdir(mr, dir, opts); err != nil {
			return err
		}
	}

	if err := mfs.PutNode(mr, path, node); err != nil {
		return err
	}

	_, err = mfs.NewFile(path, node, nil, adder.dagService)
	if err != nil {
		return err
	}

	return nil
}

func (adder *Adder) addAll(f fs.File, fi fs.FileInfo, dirname string, fsys fs.FS) (ipld.Node, error) {
	if err := adder.addFileOrDir(fi.Name(), f, fi, dirname, fsys, true); err != nil {
		return nil, err
	}

	// get root
	mr, err := adder.MfsRoot()
	if err != nil {
		return nil, err
	}
	var root mfs.FSNode
	rootdir := mr.GetDirectory()
	root = rootdir

	err = root.Flush()
	if err != nil {
		return nil, err
	}

	err = mr.Close()
	if err != nil {
		return nil, err
	}

	nd, err := root.GetNode()
	if err != nil {
		return nil, err
	}

	err = adder.dagService.Add(adder.ctx, nd)
	if err != nil {
		return nil, err
	}

	return nd, nil
}

func (adder *Adder) addFileOrDir(path string, f fs.File, fi fs.FileInfo, dirname string, fsys fs.FS, toplevel bool) error {
	defer f.Close()

	if adder.liveNodes >= liveCacheSize {
		// TODO: A smarter cache that uses some sort of lru cache with an eviction handler
		mr, err := adder.MfsRoot()
		if err != nil {
			return err
		}
		if err := mr.FlushMemFree(adder.ctx); err != nil {
			return err
		}

		adder.liveNodes = 0
	}
	adder.liveNodes++

	// TODO: re-add symlink support by inspecting fi.Mode()
	if fi.IsDir() {
		return adder.addDir(path, f, dirname, fsys, toplevel)
	}
	return adder.addFile(path, f)
}

func (adder *Adder) addFile(path string, f fs.File) error {
	dagnode, err := adder.add(f)
	if err != nil {
		return err
	}
	// patch it into the root
	return adder.addNode(dagnode, path)
}

func (adder *Adder) addDir(path string, dir fs.File, dirname string, fsys fs.FS, toplevel bool) error {
	if !(toplevel && path == "") {
		mr, err := adder.MfsRoot()
		if err != nil {
			return err
		}
		err = mfs.Mkdir(mr, path, mfs.MkdirOpts{
			Mkparents:  true,
			Flush:      false,
			CidBuilder: cidBuilder,
		})
		if err != nil {
			return err
		}
	}

	// TODO: stream entries
	var ents []fs.DirEntry
	var err error
	if d, ok := dir.(fs.ReadDirFile); ok {
		ents, err = d.ReadDir(0)
	} else if dfsys, ok := fsys.(fs.ReadDirFS); ok {
		ents, err = dfsys.ReadDir(gopath.Join(dirname, path))
	} else {
		return fmt.Errorf("directory not readable: %s", gopath.Join(dirname, path))
	}
	if err != nil {
		return fmt.Errorf("reading directory %s: %w", gopath.Join(dirname, path), err)
	}

	for _, ent := range ents {
		var f fs.File
		// If the DirEntry implements Opener then use it, otherwise open using filesystem.
		if ef, ok := ent.(w3fs.Opener); ok {
			f, err = ef.Open()
		} else {
			f, err = fsys.Open(gopath.Join(dirname, path, ent.Name()))
		}
		if err != nil {
			return fmt.Errorf("opening file %s: %w", gopath.Join(dirname, path, ent.Name()), err)
		}

		fi, err := ent.Info()
		if err != nil {
			return err
		}

		path := gopath.Join(path, ent.Name())
		err = adder.addFileOrDir(path, f, fi, dirname, fsys, false)
		if err != nil {
			return err
		}
	}
	return nil
}
