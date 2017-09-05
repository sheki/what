package what

import (
	"fmt"

	"github.com/davecgh/go-spew/spew"
)

type GithubResponse struct {
	Data struct {
		Viewer struct {
			Login string `json:"login"`
		} `json:"viewer"`
		Organization struct {
			Teams struct {
				Edges []struct {
					Node struct {
						Name    string `json:"name"`
						Members struct {
							Edges []struct {
								Node struct {
									Login string `json:"login"`
								} `json:"node"`
							} `json:"edges"`
						} `json:"members"`
					} `json:"node"`
				} `json:"edges"`
			} `json:"teams"`
		} `json:"organization"`
		Repository struct {
			PullRequests struct {
				TotalCount int `json:"totalCount"`
				Edges      []struct {
					Node struct {
						Number         int    `json:"number"`
						Title          string `json:"title"`
						URL            string `json:"url"`
						ReviewRequests struct {
							Edges []ReviewRequest `json:"edges"`
						} `json:"reviewRequests"`
						Reviews struct {
							Edges ReviewEdges `json:"edges"`
						} `json:"reviews"`
						Author struct {
							Login string `json:"login"`
						} `json:"author"`
					} `json:"node"`
				} `json:"edges"`
			} `json:"pullRequests"`
		} `json:"repository"`
	} `json:"data"`
}

func (g *GithubResponse) UsersInTeam(team string) []string {
	spew.Dump(g.Data)
	return nil
	// return g.Data.Organization.Teams.Edges.UsersInTeam(team)
}

type ReviewRequest struct {
	Node struct {
		Reviewer struct {
			Login string `json:"login"`
		} `json:"reviewer"`
	} `json:"node"`
}

type Review struct {
	Node struct {
		State  string `json:"state"`
		Author struct {
			Login string `json:"login"`
		} `json:"author"`
	} `json:"node"`
}

type ReviewEdges []Review

func (r ReviewEdges) ActorsByState(state string) []string {
	var res []string
	for _, a := range r {
		if a.Node.State == state {
			res = append(res, a.Node.Author.Login)
		}
	}
	return res
}

func (r ReviewRequest) Reviewer() string {
	return r.Node.Reviewer.Login
}

func (g GithubResponse) UserLogin() string {
	return g.Data.Viewer.Login
}

type PullRequest struct {
	Author     string
	Number     int
	Title      string
	Approvers  []string
	Rejectors  []string
	Commentors []string
	URL        string
}

func (p PullRequest) Approved() bool {
	if len(p.Approvers) > 0 {
		return true
	}
	return false
}

func (p PullRequest) Rejected() bool {
	if len(p.Rejectors) > 0 {
		return true
	}
	return false
}

const (
	APPROVED        = "APPROVED"
	CHANGES_REQUEST = "CHANGES_REQUESTED"
	COMMENTED       = "COMMENTED"
)

func (g GithubResponse) UserPRs() []PullRequest {
	login := g.UserLogin()
	var res []PullRequest
	squadUsers := g.UsersInTeam("Squad")
	fmt.Println("squadUsers", squadUsers)

	for _, pr := range g.Data.Repository.PullRequests.Edges {
		current := pr.Node
		if current.Author.Login == login {
			var approvers = mergeApprovers(
				current.Reviews.Edges.ActorsByState(APPROVED),
				squadUsers)
			var rejectors = current.Reviews.Edges.ActorsByState(CHANGES_REQUEST)
			var commentors = current.Reviews.Edges.ActorsByState(COMMENTED)
			r := PullRequest{
				Author:     login,
				Number:     pr.Node.Number,
				Title:      pr.Node.Title,
				Approvers:  approvers,
				Rejectors:  rejectors,
				Commentors: commentors,
				URL:        current.URL,
			}
			res = append(res, r)
		}
	}
	return res
}

func mergeApprovers(approvers, squadUsers []string) []string {
	var res []string
	for _, a := range approvers {
		for _, usr := range squadUsers {
			if usr == a {
				res = append(res, a)
			}
		}
	}
	return res
}

func (g GithubResponse) ParticipatingPRs() []PullRequest {
	login := g.UserLogin()
	var res []PullRequest
	squadUsers := g.UsersInTeam("Squad")
	for _, pr := range g.Data.Repository.PullRequests.Edges {
		current := pr.Node
		if current.Author.Login == login {
			continue
		}
		for _, a := range current.ReviewRequests.Edges {
			if a.Reviewer() != login {
				continue
			}
			var approvers = mergeApprovers(
				current.Reviews.Edges.ActorsByState(APPROVED),
				squadUsers)
			var rejectors = current.Reviews.Edges.ActorsByState(CHANGES_REQUEST)
			var commentors = current.Reviews.Edges.ActorsByState(COMMENTED)

			res = append(res, PullRequest{
				Author:     current.Author.Login,
				Approvers:  approvers,
				Rejectors:  rejectors,
				Commentors: commentors,
				URL:        current.URL,
				Number:     pr.Node.Number,
				Title:      pr.Node.Title,
			})
		}

		for _, review := range current.Reviews.Edges {
			if review.Node.Author.Login != login {
				continue
			}
			var approvers = current.Reviews.Edges.ActorsByState(APPROVED)
			var rejectors = current.Reviews.Edges.ActorsByState(CHANGES_REQUEST)
			var commentors = current.Reviews.Edges.ActorsByState(COMMENTED)
			res = append(res, PullRequest{
				Author:     current.Author.Login,
				Approvers:  approvers,
				Rejectors:  rejectors,
				Commentors: commentors,
				URL:        current.URL,
				Number:     pr.Node.Number,
				Title:      pr.Node.Title,
			})
		}
	}
	return dedupPRs(res)
}

func dedupPRs(input []PullRequest) []PullRequest {
	var newRes []PullRequest
	var m = map[int]PullRequest{}
	for _, v := range input {
		m[v.Number] = v
	}
	for _, v := range m {
		newRes = append(newRes, v)
	}
	return newRes
}
