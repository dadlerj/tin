package storage

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/dadlerj/tin/internal/model"
)

// SaveCommit saves a commit to the repository
func (r *Repository) SaveCommit(commit *model.TinCommit) error {
	path := filepath.Join(r.TinPath, CommitsDir, commit.ID+".json")
	data, err := json.MarshalIndent(commit, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// LoadCommit loads a commit by ID
func (r *Repository) LoadCommit(id string) (*model.TinCommit, error) {
	path := filepath.Join(r.TinPath, CommitsDir, id+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	var commit model.TinCommit
	if err := json.Unmarshal(data, &commit); err != nil {
		return nil, err
	}
	return &commit, nil
}

// ListCommits returns all commits in the repository
func (r *Repository) ListCommits() ([]*model.TinCommit, error) {
	commitsPath := filepath.Join(r.TinPath, CommitsDir)
	entries, err := os.ReadDir(commitsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []*model.TinCommit{}, nil
		}
		return nil, err
	}

	var commits []*model.TinCommit
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		id := strings.TrimSuffix(entry.Name(), ".json")
		commit, err := r.LoadCommit(id)
		if err != nil {
			continue // Skip invalid commits
		}
		commits = append(commits, commit)
	}

	// Sort by timestamp, newest first
	sort.Slice(commits, func(i, j int) bool {
		return commits[i].Timestamp.After(commits[j].Timestamp)
	})

	return commits, nil
}

// GetBranchCommit returns the commit that a branch points to
func (r *Repository) GetBranchCommit(branchName string) (*model.TinCommit, error) {
	commitID, err := r.ReadBranch(branchName)
	if err != nil {
		return nil, err
	}
	if commitID == "" {
		return nil, nil // Branch exists but points to nothing
	}
	return r.LoadCommit(commitID)
}

// GetHeadCommit returns the commit that HEAD points to
func (r *Repository) GetHeadCommit() (*model.TinCommit, error) {
	branchName, err := r.ReadHead()
	if err != nil {
		return nil, err
	}
	return r.GetBranchCommit(branchName)
}

// ReadBranch reads the commit ID a branch points to
func (r *Repository) ReadBranch(name string) (string, error) {
	path := filepath.Join(r.TinPath, RefsDir, HeadsDir, name)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil // Branch doesn't exist yet
		}
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

// WriteBranch writes a branch reference
func (r *Repository) WriteBranch(name string, commitID string) error {
	path := filepath.Join(r.TinPath, RefsDir, HeadsDir, name)
	// Create parent directories if they don't exist (for branches like "feature/foo")
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(commitID), 0644)
}

// ListBranches returns all branch names
func (r *Repository) ListBranches() ([]string, error) {
	headsPath := filepath.Join(r.TinPath, RefsDir, HeadsDir)
	if _, err := os.Stat(headsPath); os.IsNotExist(err) {
		return []string{}, nil
	}

	var branches []string
	err := filepath.WalkDir(headsPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			// Get branch name relative to heads directory
			relPath, _ := filepath.Rel(headsPath, path)
			branches = append(branches, relPath)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Strings(branches)
	return branches, nil
}

// BranchExists checks if a branch exists
func (r *Repository) BranchExists(name string) bool {
	path := filepath.Join(r.TinPath, RefsDir, HeadsDir, name)
	_, err := os.Stat(path)
	return err == nil
}

// DeleteBranch deletes a branch
func (r *Repository) DeleteBranch(name string) error {
	path := filepath.Join(r.TinPath, RefsDir, HeadsDir, name)
	return os.Remove(path)
}

// GetCommitHistory returns commits from the given commit back to the root
func (r *Repository) GetCommitHistory(startID string, limit int) ([]*model.TinCommit, error) {
	var history []*model.TinCommit
	currentID := startID

	for currentID != "" && (limit <= 0 || len(history) < limit) {
		commit, err := r.LoadCommit(currentID)
		if err != nil {
			break
		}
		history = append(history, commit)
		currentID = commit.ParentCommitID
	}

	return history, nil
}
