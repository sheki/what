package what

const query = `
{
  viewer {
    login
  }
  repository(owner: "flexport", name: "flexport") {
    pullRequests(states: [OPEN], last: 100) {
      totalCount
      edges {
        node {
          number
          title
					url
          reviewRequests(first: 100) {
            edges {
              node {
                reviewer {
                  login
                }
              }
            }
          }
          reviews(first: 100) {
            edges {
              node {
                state
                author {
                  login
                }
              }
            }
          }
          author {
            login
          }
        }
      }
    }
  }
}`
