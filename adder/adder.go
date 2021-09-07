// Simplified copy of ipfs-cluster/adder/ipfsadd/add.go
package adder

import (
	"context"
	"errors"
	"fmt"
	"io"
	gopath "path"

	cid "github.com/ipfs/go-cid"
	chunker "github.com/ipfs/go-ipfs-chunker"
	files "github.com/ipfs/go-ipfs-files"
	posinfo "github.com/ipfs/go-ipfs-posinfo"
	ipld "github.com/ipfs/go-ipld-format"
	dag "github.com/ipfs/go-merkledag"
	mfs "github.com/ipfs/go-mfs"
	unixfs "github.com/ipfs/go-unixfs"
	balanced "github.com/ipfs/go-unixfs/importer/balanced"
	ihelper "github.com/ipfs/go-unixfs/importer/helpers"
)

const chnkr = "size-1048576"
const maxLinks = 1024
const rawLeaves = true

var cidBuilder = dag.V1CidPrefix()
var liveCacheSize = uint64(256 << 10)

// NewAdder Returns a new Adder used for a file add operation.
func NewAdder(ctx context.Context, ds ipld.DAGService) (*Adder, error) {
	// Cluster: we don't use pinner nor GCLocker.
	return &Adder{
		ctx:        ctx,
		dagService: ds,
	}, nil
}

// Adder holds the switches passed to the `add` command.
type Adder struct {
	ctx        context.Context
	dagService ipld.DAGService
	mroot      *mfs.Root
	tempRoot   cid.Cid
	liveNodes  uint64
	lastFile   mfs.FSNode
	// Cluster: ipfs does a hack in commands/add.go to set the filenames
	// in emitted events correctly. We carry a root folder name (or a
	// filename in the case of single files here and emit those events
	// correctly from the beginning).
	outputPrefix string
}

func (adder *Adder) Add(name string, f files.Node) (cid.Cid, error) {
	// In order to set the AddedOutput names right, we use
	// outputPrefix:
	//
	// When adding a folder, this is the root folder name which is
	// prepended to the addedpaths.  When adding a single file,
	// this is the name of the file which overrides the empty
	// AddedOutput name.
	//
	// After coreunix/add.go was refactored in go-ipfs and we
	// followed suit, it no longer receives the name of the
	// file/folder being added and does not emit AddedOutput
	// events with the right names. We addressed this by adding
	// outputPrefix to our version. go-ipfs modifies emitted
	// events before sending to user).
	adder.outputPrefix = name

	nd, err := adder.addAllAndPin(f)
	if err != nil {
		return cid.Undef, err
	}
	return nd.Cid(), nil
}

func (adder *Adder) mfsRoot() (*mfs.Root, error) {
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

// Constructs a node from reader's data, and adds it. Doesn't pin.
func (adder *Adder) add(reader io.Reader) (ipld.Node, error) {
	chnk, err := chunker.FromString(reader, chnkr)
	if err != nil {
		return nil, err
	}

	// Cluster: we don't do batching/use BufferedDS.

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

// pinRoot recursively pins the root node of Adder and
// writes the pin state to the backing datastore.
// Cluster: we don't pin. Former Finalize().
func (adder *Adder) pinRoot(root ipld.Node) error {
	rnk := root.Cid()

	err := adder.dagService.Add(adder.ctx, root)
	if err != nil {
		return err
	}

	if adder.tempRoot.Defined() {
		adder.tempRoot = rnk
	}

	return nil
}

func (adder *Adder) outputDirs(path string, fsn mfs.FSNode) error {
	switch fsn := fsn.(type) {
	case *mfs.File:
		return nil
	case *mfs.Directory:
		names, err := fsn.ListNames(adder.ctx)
		if err != nil {
			return err
		}

		for _, name := range names {
			child, err := fsn.Child(name)
			if err != nil {
				// This fails when Child is of type *mfs.File
				// because it tries to get them from the DAG
				// service (does not implement this and returns
				// a "not found" error)
				// *mfs.Files are ignored in the recursive call
				// anyway.
				// For Cluster, we just ignore errors here.
				continue
			}

			childpath := gopath.Join(path, name)
			err = adder.outputDirs(childpath, child)
			if err != nil {
				return err
			}

			fsn.Uncache(name)
		}

		return nil
	default:
		return fmt.Errorf("unrecognized fsn type: %#v", fsn)
	}
}

func (adder *Adder) addNode(node ipld.Node, path string) error {
	// patch it into the root
	if path == "" {
		path = node.Cid().String()
	}

	if pi, ok := node.(*posinfo.FilestoreNode); ok {
		node = pi.Node
	}

	mr, err := adder.mfsRoot()
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

	// Cluster: cache the last file added.
	// This avoids using the DAGService to get the first children
	// if the MFS root when not wrapping.
	lastFile, err := mfs.NewFile(path, node, nil, adder.dagService)
	if err != nil {
		return err
	}
	adder.lastFile = lastFile

	return nil
}

// addAllAndPin adds the given request's files and pin them.
// Cluster: we don'pin. Former AddFiles.
func (adder *Adder) addAllAndPin(file files.Node) (ipld.Node, error) {
	if err := adder.addFileNode("", file, true); err != nil {
		return nil, err
	}

	// get root
	mr, err := adder.mfsRoot()
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

	// if adding a file without wrapping, swap the root to it (when adding a
	// directory, mfs root is the directory)
	_, dir := file.(files.Directory)
	var name string
	if !dir {
		children, err := rootdir.ListNames(adder.ctx)
		if err != nil {
			return nil, err
		}

		if len(children) == 0 {
			return nil, fmt.Errorf("expected at least one child dir, got none")
		}

		// Replace root with the first child
		name = children[0]
		root, err = rootdir.Child(name)
		if err != nil {
			// Cluster: use the last file we added
			// if we have one.
			if adder.lastFile == nil {
				return nil, err
			}
			root = adder.lastFile
		}
	}

	err = mr.Close()
	if err != nil {
		return nil, err
	}

	nd, err := root.GetNode()
	if err != nil {
		return nil, err
	}

	// output directory events
	err = adder.outputDirs(name, root)
	if err != nil {
		return nil, err
	}

	// Cluster: call PinRoot which adds the root cid to the DAGService.
	// Unsure if this a bug in IPFS when not pinning. Or it would get added
	// twice.
	return nd, adder.pinRoot(nd)
}

// Cluster: we don't Pause for GC
func (adder *Adder) addFileNode(path string, file files.Node, toplevel bool) error {
	defer file.Close()

	if adder.liveNodes >= liveCacheSize {
		// TODO: A smarter cache that uses some sort of lru cache with an eviction handler
		mr, err := adder.mfsRoot()
		if err != nil {
			return err
		}
		if err := mr.FlushMemFree(adder.ctx); err != nil {
			return err
		}

		adder.liveNodes = 0
	}
	adder.liveNodes++

	switch f := file.(type) {
	case files.Directory:
		return adder.addDir(path, f, toplevel)
	case *files.Symlink:
		return adder.addSymlink(path, f)
	case files.File:
		return adder.addFile(path, f)
	default:
		return errors.New("unknown file type")
	}
}

func (adder *Adder) addSymlink(path string, l *files.Symlink) error {
	sdata, err := unixfs.SymlinkData(l.Target)
	if err != nil {
		return err
	}

	dagnode := dag.NodeWithData(sdata)
	dagnode.SetCidBuilder(cidBuilder)
	err = adder.dagService.Add(adder.ctx, dagnode)
	if err != nil {
		return err
	}

	return adder.addNode(dagnode, path)
}

func (adder *Adder) addFile(path string, file files.File) error {
	dagnode, err := adder.add(file)
	if err != nil {
		return err
	}

	// patch it into the root
	return adder.addNode(dagnode, path)
}

func (adder *Adder) addDir(path string, dir files.Directory, toplevel bool) error {
	if !(toplevel && path == "") {
		mr, err := adder.mfsRoot()
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

	it := dir.Entries()
	for it.Next() {
		fpath := gopath.Join(path, it.Name())
		err := adder.addFileNode(fpath, it.Node(), false)
		if err != nil {
			return err
		}
	}

	return it.Err()
}
