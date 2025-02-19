package label

import (
	"net/http"

	"github.com/cli/cli/v2/api"
	"github.com/cli/cli/v2/internal/ghrepo"
)

const maxPageSize = 100

type listLabelsResponseData struct {
	Repository struct {
		Labels struct {
			TotalCount int
			Nodes      []label
			PageInfo   struct {
				HasNextPage bool
				EndCursor   string
			}
		}
	}
}

// listLabels lists the labels in the given repo. Pass -1 for limit to list all labels;
// otherwise, only that number of labels is returned for any number of pages.
func listLabels(client *http.Client, repo ghrepo.Interface, limit int) ([]label, int, error) {
	apiClient := api.NewClientFromHTTP(client)
	query := `
	query LabelList($owner: String!,$repo: String!, $limit: Int!, $endCursor: String) {
		repository(owner: $owner, name: $repo) {
			labels(first: $limit, after: $endCursor) {
				totalCount,
				nodes {
					name,
					description,
					color
				}
				pageInfo {
					hasNextPage
					endCursor
				}
			}
		}
	}`

	variables := map[string]interface{}{
		"owner": repo.RepoOwner(),
		"repo":  repo.RepoName(),
	}

	var labels []label
	var totalCount int

loop:
	for {
		var response listLabelsResponseData
		variables["limit"] = determinePageSize(limit - len(labels))
		err := apiClient.GraphQL(repo.RepoHost(), query, variables, &response)
		if err != nil {
			return nil, 0, err
		}

		totalCount = response.Repository.Labels.TotalCount

		for _, label := range response.Repository.Labels.Nodes {
			labels = append(labels, label)
			if len(labels) == limit {
				break loop
			}
		}

		if response.Repository.Labels.PageInfo.HasNextPage {
			variables["endCursor"] = response.Repository.Labels.PageInfo.EndCursor
		} else {
			break
		}
	}

	return labels, totalCount, nil
}

func determinePageSize(numRequestedItems int) int {
	// If numRequestedItems is -1 then retrieve maxPageSize
	if numRequestedItems < 0 || numRequestedItems > maxPageSize {
		return maxPageSize
	}
	return numRequestedItems
}
