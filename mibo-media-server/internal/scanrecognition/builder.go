package scanrecognition

import (
	"errors"
	"path"
	"sort"
	"strings"
)

func BuildTree(input Input) (*Tree, error) {
	rootPath, ok := normalizeInputPath(input.RootPath)
	if !ok {
		return nil, errors.New("root path must be an absolute path without traversal")
	}

	tree := &Tree{index: map[string]*DirectoryNode{}}
	tree.Root = &DirectoryNode{
		Path: rootPath,
		Name: nodeName(rootPath),
		Kind: DirectoryKindRoot,
	}
	tree.index[rootPath] = tree.Root

	seenFiles := map[string]struct{}{}
	for _, file := range input.Files {
		filePath, ok := normalizeInputPath(file.Path)
		if !ok || !isUnderRoot(rootPath, filePath) {
			continue
		}
		if _, ok := seenFiles[filePath]; ok {
			continue
		}
		seenFiles[filePath] = struct{}{}

		directoryPath := path.Dir(filePath)
		if directoryPath == "." {
			continue
		}
		directory := ensureDirectory(tree, rootPath, directoryPath)
		file.Path = filePath
		if file.IsVideo {
			directory.DirectVideos = append(directory.DirectVideos, file)
			continue
		}
		if file.IsNFO {
			directory.Sidecars = append(directory.Sidecars, file)
		}
	}

	sortTree(tree.Root)
	return tree, nil
}

func ensureDirectory(tree *Tree, rootPath string, directoryPath string) *DirectoryNode {
	directoryPath = normalizePath(directoryPath)
	if node, ok := tree.index[directoryPath]; ok {
		return node
	}
	if directoryPath == rootPath {
		return tree.Root
	}

	parentPath := path.Dir(directoryPath)
	if parentPath == "." || !isUnderRoot(rootPath, parentPath) {
		parentPath = rootPath
	}
	parent := ensureDirectory(tree, rootPath, parentPath)
	node := &DirectoryNode{
		Path:   directoryPath,
		Name:   nodeName(directoryPath),
		Kind:   DirectoryKindUnknown,
		parent: parent,
	}
	parent.Children = append(parent.Children, node)
	tree.index[directoryPath] = node
	return node
}

func normalizeInputPath(input string) (string, bool) {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return "", false
	}
	normalized := strings.ReplaceAll(trimmed, "\\", "/")
	if strings.Contains(normalized, "://") || !strings.HasPrefix(normalized, "/") {
		return "", false
	}
	for _, segment := range strings.Split(normalized, "/") {
		if segment == ".." {
			return "", false
		}
	}
	return path.Clean(normalized), true
}

func normalizePath(input string) string {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return "."
	}
	trimmed = strings.ReplaceAll(trimmed, "\\", "/")
	return path.Clean(trimmed)
}

func isUnderRoot(rootPath string, candidatePath string) bool {
	if rootPath == "" || candidatePath == "" || rootPath == "." || candidatePath == "." {
		return false
	}
	if !strings.HasPrefix(rootPath, "/") || !strings.HasPrefix(candidatePath, "/") {
		return false
	}
	if rootPath == "/" {
		return true
	}
	return candidatePath == rootPath || strings.HasPrefix(candidatePath, rootPath+"/")
}

func nodeName(pathValue string) string {
	pathValue = normalizePath(pathValue)
	if pathValue == "/" {
		return "/"
	}
	return path.Base(pathValue)
}

func sortTree(node *DirectoryNode) {
	if node == nil {
		return
	}
	sort.Slice(node.Children, func(i, j int) bool {
		return node.Children[i].Path < node.Children[j].Path
	})
	sort.Slice(node.DirectVideos, func(i, j int) bool {
		if node.DirectVideos[i].Path == node.DirectVideos[j].Path {
			return node.DirectVideos[i].ID < node.DirectVideos[j].ID
		}
		return node.DirectVideos[i].Path < node.DirectVideos[j].Path
	})
	sort.Slice(node.Sidecars, func(i, j int) bool {
		if node.Sidecars[i].Path == node.Sidecars[j].Path {
			return node.Sidecars[i].ID < node.Sidecars[j].ID
		}
		return node.Sidecars[i].Path < node.Sidecars[j].Path
	})
	for _, child := range node.Children {
		sortTree(child)
	}
}
