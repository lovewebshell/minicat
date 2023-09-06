package source

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/anchore/stereoscope/pkg/file"
	"github.com/anchore/stereoscope/pkg/filetree"
	"github.com/lovewebshell/minicat/internal"
	"github.com/lovewebshell/minicat/internal/log"
)

const WindowsOS = "windows"

var unixSystemRuntimePrefixes = []string{
	"/proc",
	"/dev",
	"/sys",
}

var _ FileResolver = (*directoryResolver)(nil)

type pathFilterFn func(string, os.FileInfo) bool

type directoryResolver struct {
	path                    string
	currentWdRelativeToRoot string
	currentWd               string
	fileTree                *filetree.FileTree
	metadata                map[file.ID]FileMetadata

	pathFilterFns  []pathFilterFn
	refsByMIMEType map[string][]file.Reference
	errPaths       map[string]error
}

func newDirectoryResolver(root string, pathFilters ...pathFilterFn) (*directoryResolver, error) {
	currentWD, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("could not gret CWD: %w", err)
	}

	cleanCWD, err := filepath.EvalSymlinks(currentWD)
	if err != nil {
		return nil, fmt.Errorf("could not evaluate CWD symlinks: %w", err)
	}

	cleanRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return nil, fmt.Errorf("could not evaluate root=%q symlinks: %w", root, err)
	}

	var currentWdRelRoot string
	if path.IsAbs(cleanRoot) {
		currentWdRelRoot, err = filepath.Rel(cleanCWD, cleanRoot)
		if err != nil {
			return nil, fmt.Errorf("could not determine given root path to CWD: %w", err)
		}
	} else {
		currentWdRelRoot = filepath.Clean(cleanRoot)
	}

	resolver := directoryResolver{
		path:                    cleanRoot,
		currentWd:               cleanCWD,
		currentWdRelativeToRoot: currentWdRelRoot,
		fileTree:                filetree.NewFileTree(),
		metadata:                make(map[file.ID]FileMetadata),
		pathFilterFns:           append([]pathFilterFn{isUnallowableFileType, isUnixSystemRuntimePath}, pathFilters...),
		refsByMIMEType:          make(map[string][]file.Reference),
		errPaths:                make(map[string]error),
	}

	return &resolver, indexAllRoots(cleanRoot, resolver.indexTree)
}

func (r *directoryResolver) indexTree(root string) ([]string, error) {
	log.Debugf("indexing filesystem path=%q", root)

	var roots []string
	var err error

	root, err = filepath.Abs(root)
	if err != nil {
		return nil, err
	}

	fi, err := os.Stat(root)
	if err != nil && fi != nil && !fi.IsDir() {

		newRoot, _ := r.indexPath(root, fi, nil)
		if newRoot != "" {
			roots = append(roots, newRoot)
		}
		return roots, nil
	}

	return roots, filepath.Walk(root,
		func(path string, info os.FileInfo, err error) error {

			newRoot, err := r.indexPath(path, info, err)

			if err != nil {
				return err
			}

			if newRoot != "" {
				roots = append(roots, newRoot)
			}

			return nil
		})
}

func (r *directoryResolver) indexPath(path string, info os.FileInfo, err error) (string, error) {

	if r.hasBeenIndexed(path) {
		return "", nil
	}

	for _, filterFn := range r.pathFilterFns {
		if filterFn != nil && filterFn(path, info) {
			if info != nil && info.IsDir() {
				return "", fs.SkipDir
			}
			return "", nil
		}
	}

	if r.isFileAccessErr(path, err) {
		return "", nil
	}

	if info == nil {

		r.errPaths[path] = fmt.Errorf("no file info observable at path=%q", path)
		return "", nil
	}

	if runtime.GOOS == WindowsOS {
		path = windowsToPosix(path)
	}

	newRoot, err := r.addPathToIndex(path, info)
	if r.isFileAccessErr(path, err) {
		return "", nil
	}

	return newRoot, nil
}

func (r *directoryResolver) isFileAccessErr(path string, err error) bool {

	if err != nil {
		log.Warnf("unable to access path=%q: %+v", path, err)
		r.errPaths[path] = err
		return true
	}
	return false
}

func (r directoryResolver) addPathToIndex(p string, info os.FileInfo) (string, error) {
	switch t := newFileTypeFromMode(info.Mode()); t {
	case SymbolicLink:
		return r.addSymlinkToIndex(p, info)
	case Directory:
		return "", r.addDirectoryToIndex(p, info)
	case RegularFile:
		return "", r.addFileToIndex(p, info)
	default:
		return "", fmt.Errorf("unsupported file type: %s", t)
	}
}

func (r directoryResolver) hasBeenIndexed(p string) bool {
	filePath := file.Path(p)
	if !r.fileTree.HasPath(filePath) {
		return false
	}

	exists, ref, err := r.fileTree.File(filePath)
	if err != nil || !exists || ref == nil {
		return false
	}

	_, exists = r.metadata[ref.ID()]
	return exists
}

func (r directoryResolver) addDirectoryToIndex(p string, info os.FileInfo) error {
	ref, err := r.fileTree.AddDir(file.Path(p))
	if err != nil {
		return err
	}

	location := NewLocationFromDirectory(p, *ref)
	metadata := fileMetadataFromPath(p, info, r.isInIndex(location))
	r.addFileMetadataToIndex(ref, metadata)

	return nil
}

func (r directoryResolver) addFileToIndex(p string, info os.FileInfo) error {
	ref, err := r.fileTree.AddFile(file.Path(p))
	if err != nil {
		return err
	}

	location := NewLocationFromDirectory(p, *ref)
	metadata := fileMetadataFromPath(p, info, r.isInIndex(location))
	r.addFileMetadataToIndex(ref, metadata)

	return nil
}

func (r directoryResolver) addSymlinkToIndex(p string, info os.FileInfo) (string, error) {
	var usedInfo = info

	linkTarget, err := os.Readlink(p)
	if err != nil {
		return "", fmt.Errorf("unable to readlink for path=%q: %w", p, err)
	}

	if !filepath.IsAbs(linkTarget) {
		linkTarget = filepath.Join(filepath.Dir(p), linkTarget)
	}

	ref, err := r.fileTree.AddSymLink(file.Path(p), file.Path(linkTarget))
	if err != nil {
		return "", err
	}

	targetAbsPath := linkTarget
	if !filepath.IsAbs(targetAbsPath) {
		targetAbsPath = filepath.Clean(filepath.Join(path.Dir(p), linkTarget))
	}

	location := NewLocationFromDirectory(p, *ref)
	location.VirtualPath = p
	metadata := fileMetadataFromPath(p, usedInfo, r.isInIndex(location))
	metadata.LinkDestination = linkTarget
	r.addFileMetadataToIndex(ref, metadata)

	return targetAbsPath, nil
}

func (r directoryResolver) addFileMetadataToIndex(ref *file.Reference, metadata FileMetadata) {
	if ref != nil {
		if metadata.MIMEType != "" {
			r.refsByMIMEType[metadata.MIMEType] = append(r.refsByMIMEType[metadata.MIMEType], *ref)
		}
		r.metadata[ref.ID()] = metadata
	}
}

func (r directoryResolver) requestPath(userPath string) (string, error) {
	if filepath.IsAbs(userPath) {

		userPath = path.Join(r.path, userPath)
	} else {

		userPath = path.Join(r.currentWdRelativeToRoot, userPath)
	}

	var err error
	userPath, err = filepath.Abs(userPath)
	if err != nil {
		return "", err
	}
	return userPath, nil
}

func (r directoryResolver) responsePath(path string) string {

	if runtime.GOOS == WindowsOS {
		path = posixToWindows(path)
	}

	if filepath.IsAbs(path) {

		prefix := filepath.Clean(filepath.Join(r.currentWd, r.currentWdRelativeToRoot))
		return strings.TrimPrefix(path, prefix+string(filepath.Separator))
	}
	return path
}

func (r *directoryResolver) HasPath(userPath string) bool {
	requestPath, err := r.requestPath(userPath)
	if err != nil {
		return false
	}
	return r.fileTree.HasPath(file.Path(requestPath))
}

func (r directoryResolver) String() string {
	return fmt.Sprintf("dir:%s", r.path)
}

func (r directoryResolver) FilesByPath(userPaths ...string) ([]Location, error) {
	var references = make([]Location, 0)

	for _, userPath := range userPaths {
		userStrPath, err := r.requestPath(userPath)
		if err != nil {
			log.Warnf("unable to get file by path=%q : %+v", userPath, err)
			continue
		}

		evaluatedPath, err := filepath.EvalSymlinks(userStrPath)
		if err != nil {
			log.Debugf("directory resolver unable to evaluate symlink for path=%q : %+v", userPath, err)
			continue
		}

		fileMeta, err := os.Stat(evaluatedPath)
		if errors.Is(err, os.ErrNotExist) {

			continue
		} else if err != nil {

			var pathErr *os.PathError
			if !errors.As(err, &pathErr) {
				log.Warnf("path is not valid (%s): %+v", evaluatedPath, err)
			}
			continue
		}

		if fileMeta.IsDir() {
			continue
		}

		if runtime.GOOS == WindowsOS {
			userStrPath = windowsToPosix(userStrPath)
		}

		exists, ref, err := r.fileTree.File(file.Path(userStrPath), filetree.FollowBasenameLinks)
		if err == nil && exists {
			loc := NewVirtualLocationFromDirectory(
				r.responsePath(string(ref.RealPath)),
				r.responsePath(userStrPath),
				*ref,
			)
			references = append(references, loc)
		}
	}

	return references, nil
}

func (r directoryResolver) FilesByGlob(patterns ...string) ([]Location, error) {
	result := make([]Location, 0)

	for _, pattern := range patterns {
		globResults, err := r.fileTree.FilesByGlob(pattern, filetree.FollowBasenameLinks)
		if err != nil {
			return nil, err
		}
		for _, globResult := range globResults {
			loc := NewVirtualLocationFromDirectory(
				r.responsePath(string(globResult.Reference.RealPath)),
				r.responsePath(string(globResult.MatchPath)),
				globResult.Reference,
			)
			result = append(result, loc)
		}
	}

	return result, nil
}

func (r *directoryResolver) RelativeFileByPath(_ Location, path string) *Location {
	paths, err := r.FilesByPath(path)
	if err != nil {
		return nil
	}
	if len(paths) == 0 {
		return nil
	}

	return &paths[0]
}

func (r directoryResolver) FileContentsByLocation(location Location) (io.ReadCloser, error) {
	if location.ref.RealPath == "" {
		return nil, errors.New("empty path given")
	}
	if !r.isInIndex(location) {

		return nil, fmt.Errorf("file content is inaccessible path=%q", location.ref.RealPath)
	}

	filePath := string(location.ref.RealPath)
	if runtime.GOOS == WindowsOS {
		filePath = posixToWindows(filePath)
	}
	return file.NewLazyReadCloser(filePath), nil
}

func (r directoryResolver) isInIndex(location Location) bool {
	if location.ref.RealPath == "" {
		return false
	}
	return r.fileTree.HasPath(location.ref.RealPath, filetree.FollowBasenameLinks)
}

func (r *directoryResolver) AllLocations() <-chan Location {
	results := make(chan Location)
	go func() {
		defer close(results)

		for _, ref := range r.fileTree.AllFiles(file.TypeReg, file.TypeSymlink, file.TypeHardLink, file.TypeBlockDevice, file.TypeCharacterDevice, file.TypeFifo) {
			results <- NewLocationFromDirectory(r.responsePath(string(ref.RealPath)), ref)
		}
	}()
	return results
}

func (r *directoryResolver) FileMetadataByLocation(location Location) (FileMetadata, error) {
	metadata, exists := r.metadata[location.ref.ID()]
	if !exists {
		return FileMetadata{}, fmt.Errorf("location: %+v : %w", location, os.ErrNotExist)
	}

	return metadata, nil
}

func (r *directoryResolver) FilesByMIMEType(types ...string) ([]Location, error) {
	var locations []Location
	for _, ty := range types {
		if refs, ok := r.refsByMIMEType[ty]; ok {
			for _, ref := range refs {
				locations = append(locations, NewLocationFromDirectory(r.responsePath(string(ref.RealPath)), ref))
			}
		}
	}
	return locations, nil
}

func windowsToPosix(windowsPath string) (posixPath string) {

	volumeName := filepath.VolumeName(windowsPath)
	pathWithoutVolume := strings.TrimPrefix(windowsPath, volumeName)
	volumeLetter := strings.ToLower(strings.TrimSuffix(volumeName, ":"))

	translatedPath := strings.ReplaceAll(pathWithoutVolume, "\\", "/")

	return path.Clean("/" + strings.Join([]string{volumeLetter, translatedPath}, "/"))
}

func posixToWindows(posixPath string) (windowsPath string) {

	pathFields := strings.Split(posixPath, "/")
	volumeName := strings.ToUpper(pathFields[1]) + `:\\`

	remainingTranslatedPath := strings.Join(pathFields[2:], "\\")

	return filepath.Clean(volumeName + remainingTranslatedPath)
}

func isUnixSystemRuntimePath(path string, _ os.FileInfo) bool {
	return internal.HasAnyOfPrefixes(path, unixSystemRuntimePrefixes...)
}

func isUnallowableFileType(_ string, info os.FileInfo) bool {
	if info == nil {

		return false
	}
	switch newFileTypeFromMode(info.Mode()) {
	case CharacterDevice, Socket, BlockDevice, FIFONode, IrregularFile:
		return true

	}

	return false
}

func indexAllRoots(root string, indexer func(string) ([]string, error)) error {

	pathsToIndex := []string{root}
	fullPathsMap := map[string]struct{}{}

loop:
	for {
		var currentPath string
		switch len(pathsToIndex) {
		case 0:
			break loop
		case 1:
			currentPath, pathsToIndex = pathsToIndex[0], nil
		default:
			currentPath, pathsToIndex = pathsToIndex[0], pathsToIndex[1:]
		}

		additionalRoots, err := indexer(currentPath)
		if err != nil {
			return fmt.Errorf("unable to index filesystem path=%q: %w", currentPath, err)
		}

		for _, newRoot := range additionalRoots {
			if _, ok := fullPathsMap[newRoot]; !ok {
				fullPathsMap[newRoot] = struct{}{}
				pathsToIndex = append(pathsToIndex, newRoot)
			}
		}
	}

	return nil
}
