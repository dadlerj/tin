package storage

import (
	"testing"

	"github.com/dadlerj/tin/internal/model"
)

func TestRepository_SaveAndLoadCommit(t *testing.T) {
	tmpDir := t.TempDir()

	repo, err := Init(tmpDir)
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// Create a commit
	threads := []model.ThreadRef{
		{ThreadID: "thread-1", MessageCount: 10},
	}
	commit := model.NewTinCommit("Test commit", threads, "gitabc123", "")

	// Save
	if err := repo.SaveCommit(commit); err != nil {
		t.Fatalf("SaveCommit failed: %v", err)
	}

	// Load
	loaded, err := repo.LoadCommit(commit.ID)
	if err != nil {
		t.Fatalf("LoadCommit failed: %v", err)
	}

	if loaded.ID != commit.ID {
		t.Errorf("expected ID %s, got %s", commit.ID, loaded.ID)
	}
	if loaded.Message != "Test commit" {
		t.Errorf("expected message 'Test commit', got %s", loaded.Message)
	}
	if loaded.GitCommitHash != "gitabc123" {
		t.Errorf("expected git hash 'gitabc123', got %s", loaded.GitCommitHash)
	}
	if len(loaded.Threads) != 1 {
		t.Errorf("expected 1 thread, got %d", len(loaded.Threads))
	}
}

func TestRepository_LoadCommit_NotFound(t *testing.T) {
	tmpDir := t.TempDir()

	repo, err := Init(tmpDir)
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	_, err = repo.LoadCommit("nonexistent")
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestRepository_ListCommits(t *testing.T) {
	tmpDir := t.TempDir()

	repo, err := Init(tmpDir)
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// Initially empty
	commits, err := repo.ListCommits()
	if err != nil {
		t.Fatalf("ListCommits failed: %v", err)
	}
	if len(commits) != 0 {
		t.Errorf("expected 0 commits, got %d", len(commits))
	}

	// Create and save commits
	threads := []model.ThreadRef{{ThreadID: "t1", MessageCount: 1}}
	for i := 0; i < 3; i++ {
		commit := model.NewTinCommit("Commit message", threads, "git123", "")
		if err := repo.SaveCommit(commit); err != nil {
			t.Fatalf("SaveCommit failed: %v", err)
		}
	}

	// List should return 3
	commits, err = repo.ListCommits()
	if err != nil {
		t.Fatalf("ListCommits failed: %v", err)
	}
	if len(commits) != 3 {
		t.Errorf("expected 3 commits, got %d", len(commits))
	}
}

func TestRepository_BranchOperations(t *testing.T) {
	tmpDir := t.TempDir()

	repo, err := Init(tmpDir)
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// Initially no branches (branch file only created when commit is written)
	branches, err := repo.ListBranches()
	if err != nil {
		t.Fatalf("ListBranches failed: %v", err)
	}
	if len(branches) != 0 {
		t.Errorf("expected 0 branches initially, got %d", len(branches))
	}

	// Create new branch (with slash to test nested directory creation)
	if err := repo.WriteBranch("feature/test", "commit-123"); err != nil {
		t.Fatalf("WriteBranch failed: %v", err)
	}

	// Verify branch exists
	if !repo.BranchExists("feature/test") {
		t.Error("expected 'feature/test' branch to exist")
	}
	if repo.BranchExists("nonexistent") {
		t.Error("expected 'nonexistent' branch to not exist")
	}

	// Read branch
	commitID, err := repo.ReadBranch("feature/test")
	if err != nil {
		t.Fatalf("ReadBranch failed: %v", err)
	}
	if commitID != "commit-123" {
		t.Errorf("expected 'commit-123', got %s", commitID)
	}

	// List branches
	branches, err = repo.ListBranches()
	if err != nil {
		t.Fatalf("ListBranches failed: %v", err)
	}
	if len(branches) != 1 {
		t.Errorf("expected 1 branch, got %d", len(branches))
	}
}

func TestRepository_DeleteBranch(t *testing.T) {
	tmpDir := t.TempDir()

	repo, err := Init(tmpDir)
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// Create branch (with slash to test nested directory creation)
	repo.WriteBranch("feature/test", "commit-123")

	// Delete it
	if err := repo.DeleteBranch("feature/test"); err != nil {
		t.Fatalf("DeleteBranch failed: %v", err)
	}

	// Verify deleted
	if repo.BranchExists("feature/test") {
		t.Error("expected 'feature/test' branch to be deleted")
	}
}

func TestRepository_GetBranchCommit(t *testing.T) {
	tmpDir := t.TempDir()

	repo, err := Init(tmpDir)
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// Branch with no commit
	commit, err := repo.GetBranchCommit("main")
	if err != nil {
		t.Fatalf("GetBranchCommit failed: %v", err)
	}
	if commit != nil {
		t.Error("expected nil commit for branch with no commits")
	}

	// Create and save a commit
	threads := []model.ThreadRef{{ThreadID: "t1", MessageCount: 1}}
	newCommit := model.NewTinCommit("Test", threads, "git123", "")
	repo.SaveCommit(newCommit)

	// Point branch to commit
	repo.WriteBranch("main", newCommit.ID)

	// Get branch commit
	commit, err = repo.GetBranchCommit("main")
	if err != nil {
		t.Fatalf("GetBranchCommit failed: %v", err)
	}
	if commit == nil {
		t.Fatal("expected non-nil commit")
	}
	if commit.ID != newCommit.ID {
		t.Errorf("expected commit %s, got %s", newCommit.ID, commit.ID)
	}
}

func TestRepository_GetHeadCommit(t *testing.T) {
	tmpDir := t.TempDir()

	repo, err := Init(tmpDir)
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// Initially no commit
	commit, err := repo.GetHeadCommit()
	if err != nil {
		t.Fatalf("GetHeadCommit failed: %v", err)
	}
	if commit != nil {
		t.Error("expected nil commit initially")
	}

	// Create commit
	threads := []model.ThreadRef{{ThreadID: "t1", MessageCount: 1}}
	newCommit := model.NewTinCommit("Test", threads, "git123", "")
	repo.SaveCommit(newCommit)
	repo.WriteBranch("main", newCommit.ID)

	// Get HEAD commit
	commit, err = repo.GetHeadCommit()
	if err != nil {
		t.Fatalf("GetHeadCommit failed: %v", err)
	}
	if commit == nil {
		t.Fatal("expected non-nil HEAD commit")
	}
	if commit.ID != newCommit.ID {
		t.Errorf("expected %s, got %s", newCommit.ID, commit.ID)
	}
}

func TestRepository_GetCommitHistory(t *testing.T) {
	tmpDir := t.TempDir()

	repo, err := Init(tmpDir)
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// Create chain of commits
	threads := []model.ThreadRef{{ThreadID: "t1", MessageCount: 1}}

	commit1 := model.NewTinCommit("First", threads, "git1", "")
	repo.SaveCommit(commit1)

	commit2 := model.NewTinCommit("Second", threads, "git2", commit1.ID)
	repo.SaveCommit(commit2)

	commit3 := model.NewTinCommit("Third", threads, "git3", commit2.ID)
	repo.SaveCommit(commit3)

	// Get full history
	history, err := repo.GetCommitHistory(commit3.ID, 0)
	if err != nil {
		t.Fatalf("GetCommitHistory failed: %v", err)
	}
	if len(history) != 3 {
		t.Errorf("expected 3 commits in history, got %d", len(history))
	}

	// Verify order (newest first)
	if history[0].ID != commit3.ID {
		t.Error("expected newest commit first")
	}
	if history[2].ID != commit1.ID {
		t.Error("expected oldest commit last")
	}

	// Test with limit
	history, err = repo.GetCommitHistory(commit3.ID, 2)
	if err != nil {
		t.Fatalf("GetCommitHistory with limit failed: %v", err)
	}
	if len(history) != 2 {
		t.Errorf("expected 2 commits with limit, got %d", len(history))
	}
}

func TestRepository_ReadBranch_NonExistent(t *testing.T) {
	tmpDir := t.TempDir()

	repo, err := Init(tmpDir)
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// Reading non-existent branch should return empty string, not error
	commitID, err := repo.ReadBranch("nonexistent")
	if err != nil {
		t.Fatalf("ReadBranch for nonexistent should not error, got: %v", err)
	}
	if commitID != "" {
		t.Errorf("expected empty string for nonexistent branch, got %s", commitID)
	}
}
