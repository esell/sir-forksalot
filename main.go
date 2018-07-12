package main

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	gh "github.com/google/go-github/github"
	"github.com/gregjones/httpcache"
	git "github.com/libgit2/git2go"
	"golang.org/x/oauth2"
)

func main() {
	ctx := context.Background()
	tp := httpcache.NewMemoryCacheTransport()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: os.Getenv("GITHUB_TOKEN")},
	)
	httpClient := oauth2.NewClient(ctx, ts)

	httpClient.Transport = &oauth2.Transport{
		Source: ts,
		Base:   tp,
	}

	client := gh.NewClient(httpClient)
	opts := &gh.RepositoryListByOrgOptions{Type: "forks"}
	repos, _, err := client.Repositories.ListByOrg(context.Background(), os.Getenv("GITHUB_ORG"), opts)
	if err != nil {
		log.Fatal(err)
	}

	for _, v := range repos {

		tempRepo, _, err := client.Repositories.Get(context.Background(), os.Getenv("GITHUB_ORG"), *v.Name)
		if err != nil {
			log.Fatal(err)
		}

		clonePath, err := ioutil.TempDir("", *tempRepo.Name)
		if err != nil {
			log.Fatal(err)
		}
		// clone origin repo
		log.Println("cloning " + *tempRepo.FullName)
		clonedRepo, err := git.Clone(*tempRepo.CloneURL, clonePath, &git.CloneOptions{})
		if err != nil {
			log.Fatal(err)
		}

		// add upstream repo
		log.Println("adding upstream " + *tempRepo.Source.FullName)
		_, err = clonedRepo.Remotes.Create("upstream", *tempRepo.Source.CloneURL)
		if err != nil {
			log.Fatal(err)
		}

		err = pullAndMerge(clonedRepo)

		if err != nil {
			log.Fatal(err)
		}
	}
}

func credentialsCallback(url string, username string, allowedTypes git.CredType) (git.ErrorCode, *git.Cred) {
	ret, cred := git.NewCredUserpassPlaintext(os.Getenv("GITHUB_USERNAME"), os.Getenv("GITHUB_TOKEN"))
	return git.ErrorCode(ret), &cred
}

func pullAndMerge(repo *git.Repository) error {
	remote, err := repo.Remotes.Lookup("upstream")
	if err != nil {
		return err
	}

	if err := remote.Fetch([]string{}, nil, ""); err != nil {
		return err
	}
	remoteBranch, err := repo.References.Lookup("refs/remotes/upstream/master")
	if err != nil {
		return err
	}
	remoteBranchID := remoteBranch.Target()
	annotatedCommit, err := repo.AnnotatedCommitFromRef(remoteBranch)
	if err != nil {
		return err
	}

	// Do the merge analysis
	mergeHeads := make([]*git.AnnotatedCommit, 1)
	mergeHeads[0] = annotatedCommit
	analysis, _, err := repo.MergeAnalysis(mergeHeads)
	if err != nil {
		return err
	}

	head, err := repo.Head()
	if err != nil {
		return err
	}
	if analysis&git.MergeAnalysisUpToDate != 0 {
		log.Println("Repo already up-to-date, skipping")
		return nil
	} else if analysis&git.MergeAnalysisNormal != 0 {
		// Just merge changes
		if err := repo.Merge([]*git.AnnotatedCommit{annotatedCommit}, nil, nil); err != nil {
			return err
		}
		// Check for conflicts
		index, err := repo.Index()
		if err != nil {
			return err
		}

		if index.HasConflicts() {
			return errors.New("conflicts encountered, please resolve them")
		}

		// Make the merge commit
		sig, err := repo.DefaultSignature()
		if err != nil {
			return err
		}

		// Get Write Tree
		treeID, err := index.WriteTree()
		if err != nil {
			return err
		}

		tree, err := repo.LookupTree(treeID)
		if err != nil {
			return err
		}

		localCommit, err := repo.LookupCommit(head.Target())
		if err != nil {
			return err
		}

		remoteCommit, err := repo.LookupCommit(remoteBranchID)
		if err != nil {
			return err
		}

		repo.CreateCommit("HEAD", sig, sig, "", tree, localCommit, remoteCommit)

		// push changes to origin
		pushToOrigin(repo)

		// Clean up
		repo.StateCleanup()
	} else if analysis&git.MergeAnalysisFastForward != 0 {
		// Fast-forward changes
		// Get remote tree
		remoteTree, err := repo.LookupTree(remoteBranchID)
		if err != nil {
			return err
		}

		if err := repo.CheckoutTree(remoteTree, nil); err != nil {
			return err
		}

		branchRef, err := repo.References.Lookup("refs/heads/master")
		if err != nil {
			return err
		}

		// Point branch to the object
		branchRef.SetTarget(remoteBranchID, "")
		if _, err := head.SetTarget(remoteBranchID, ""); err != nil {
			return err
		}

		// push changes to origin
		pushToOrigin(repo)
	} else {
		return fmt.Errorf("Unexpected merge analysis result %d", analysis)
	}

	return nil
}

func pushToOrigin(repo *git.Repository) error {
	// push to origin
	originRepo, err := repo.Remotes.Lookup("origin")
	if err != nil {
		return err
	}

	log.Println("pushing changes to origin")
	pushOptions := &git.PushOptions{}
	pushOptions.RemoteCallbacks.CredentialsCallback = credentialsCallback

	err = originRepo.Push([]string{"refs/heads/master"}, pushOptions)
	if err != nil {
		return err
	}
	return nil
}
